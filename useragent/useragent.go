package useragent

import (
	"container/list"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	// MaxUserAgentLength 限制解析和全局缓存接受的最大 User-Agent 字节长度，防止恶意超长请求头耗尽内存与正则 CPU。
	MaxUserAgentLength = 512
)

var (
	chromeRegex    = regexp.MustCompile(`(?i)chrome\/([\d.]+)`)
	safariRegex    = regexp.MustCompile(`(?i)safari\/([\d.]+)`)
	firefoxRegex   = regexp.MustCompile(`(?i)firefox\/([\d.]+)`)
	edgeRegex      = regexp.MustCompile(`(?i)(?:edge|edg)\/([\d.]+)`)
	operaRegex     = regexp.MustCompile(`(?i)opr\/([\d.]+)`)
	msieRegex      = regexp.MustCompile(`(?i)msie ([\d.]+)`)
	tridentRvRegex = regexp.MustCompile(`(?i)trident\/[\d.]+.*?;\s*rv:([\d.]+)`)
	tridentRegex   = regexp.MustCompile(`(?i)trident\/([\d.]+)`)
	versionRegex   = regexp.MustCompile(`(?i)version\/([\d.]+)`)

	botIdentifiers = []string{
		"bot",
		"spider",
		"crawler",
		"slurp",
		"bingbot",
		"baiduspider",
		"yandexbot",
		"sogou",
		"exabot",
		"facebookexternalhit",
		"ia_archiver",
	}

	commonBrowserIdentifiers = []string{
		"mozilla",
		"webkit",
		"gecko",
		"presto",
		"trident",
		"chrome",
		"safari",
		"firefox",
		"edge",
		"opera",
		"msie",
		"opr",
		"mobile",
		"android",
		"iphone",
		"ipad",
		"ipod",
		"samsung",
		"miui",
		"ucbrowser",
		"qqbrowser",
		"maxthon",
		"crios",
		"fxios",
	}
)

var (
	errInvalidCacheCapacity = errors.New("invalid cache capacity")
	errInvalidCacheTTL      = errors.New("invalid cache ttl")
	errUninitializedCache   = errors.New("uninitialized cache")
)

// BrowserInfo 存储启发式浏览器识别结果，不是安全判定。
type BrowserInfo struct {
	IsBrowser bool
	Name      string
	Version   string
}

type cacheEntry struct {
	key        string
	value      BrowserInfo
	expiration time.Time
}

type browserLRUCache struct {
	capacity int
	ttl      time.Duration
	mu       sync.Mutex
	items    map[string]*list.Element
	order    *list.List
	now      func() time.Time
}

func newBrowserLRUCache(capacity int, ttl time.Duration) (*browserLRUCache, error) {
	if capacity <= 0 {
		return nil, errInvalidCacheCapacity
	}
	if ttl < 0 {
		return nil, errInvalidCacheTTL
	}
	return &browserLRUCache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*list.Element),
		order:    list.New(),
		now:      time.Now,
	}, nil
}

func (c *browserLRUCache) get(key string) (BrowserInfo, bool) {
	if c == nil {
		return BrowserInfo{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return BrowserInfo{}, false
	}
	entry := el.Value.(*cacheEntry)
	if c.ttl > 0 && !c.now().Before(entry.expiration) {
		c.removeElement(el)
		return BrowserInfo{}, false
	}
	c.order.MoveToFront(el)
	return entry.value, true
}

func (c *browserLRUCache) put(key string, value BrowserInfo) error {
	if c == nil {
		return errUninitializedCache
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var exp time.Time
	if c.ttl > 0 {
		exp = c.now().Add(c.ttl)
	}
	if el, ok := c.items[key]; ok {
		entry := el.Value.(*cacheEntry)
		entry.value = value
		entry.expiration = exp
		c.order.MoveToFront(el)
		return nil
	}
	if c.order.Len() >= c.capacity {
		if last := c.order.Back(); last != nil {
			c.removeElement(last)
		}
	}
	el := c.order.PushFront(&cacheEntry{key: key, value: value, expiration: exp})
	c.items[key] = el
	return nil
}

func (c *browserLRUCache) removeElement(el *list.Element) {
	entry := el.Value.(*cacheEntry)
	delete(c.items, entry.key)
	c.order.Remove(el)
}

type shardedBrowserCache struct {
	shards [16]*browserLRUCache
}

func newShardedBrowserCache(shardCapacity int, ttl time.Duration) (*shardedBrowserCache, error) {
	sc := &shardedBrowserCache{}
	for i := 0; i < 16; i++ {
		cache, err := newBrowserLRUCache(shardCapacity, ttl)
		if err != nil {
			return nil, err
		}
		sc.shards[i] = cache
	}
	return sc, nil
}

func (sc *shardedBrowserCache) getShard(key string) *browserLRUCache {
	if sc == nil {
		return nil
	}
	// FNV-1a 32-bit hash，避免简单字节求和导致分片倾斜。
	var hash uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	return sc.shards[hash&15]
}

func (sc *shardedBrowserCache) get(key string) (BrowserInfo, bool) {
	if sc == nil {
		return BrowserInfo{}, false
	}
	return sc.getShard(key).get(key)
}

func (sc *shardedBrowserCache) put(key string, value BrowserInfo) error {
	if sc == nil {
		return errUninitializedCache
	}
	return sc.getShard(key).put(key, value)
}

var (
	browserCacheOnce sync.Once
	browserInfoCache *shardedBrowserCache
)

func getBrowserInfoCache() *shardedBrowserCache {
	browserCacheOnce.Do(func() {
		cache, err := newShardedBrowserCache(1000, time.Hour)
		if err != nil {
			panic(err)
		}
		browserInfoCache = cache
	})
	return browserInfoCache
}

func truncateUTF8Prefix(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}

func fastBrowserCheck(ua string) bool {
	for _, browser := range commonBrowserIdentifiers {
		if strings.Contains(ua, browser) {
			return true
		}
	}
	return false
}

func fastBotCheck(ua string) bool {
	for _, bot := range botIdentifiers {
		if strings.Contains(ua, bot) {
			return true
		}
	}
	return false
}

func matchBrowserVersion(userAgent, browserName string, regex *regexp.Regexp) (BrowserInfo, bool) {
	submatch := regex.FindStringSubmatch(userAgent)
	if submatch == nil {
		return BrowserInfo{}, false
	}
	version := ""
	if len(submatch) > 1 {
		version = submatch[1]
	}
	return BrowserInfo{IsBrowser: true, Name: browserName, Version: version}, true
}

func parseBrowserInfo(userAgent string) BrowserInfo {
	if userAgent == "" {
		return BrowserInfo{}
	}
	userAgent = truncateUTF8Prefix(userAgent, MaxUserAgentLength)
	ua := strings.ToLower(userAgent)
	if fastBotCheck(ua) {
		return BrowserInfo{IsBrowser: false}
	}
	if strings.Contains(ua, "edg/") || strings.Contains(ua, "edge/") {
		if info, ok := matchBrowserVersion(userAgent, "Edge", edgeRegex); ok {
			return info
		}
		return BrowserInfo{IsBrowser: true, Name: "Edge"}
	}
	if strings.Contains(ua, "opr/") {
		if info, ok := matchBrowserVersion(userAgent, "Opera", operaRegex); ok {
			return info
		}
		return BrowserInfo{IsBrowser: true, Name: "Opera"}
	}

	versionMatchers := []struct {
		name  string
		regex *regexp.Regexp
	}{
		{"Chrome", chromeRegex},
		{"Firefox", firefoxRegex},
		{"Internet Explorer", msieRegex},
		{"Internet Explorer", tridentRvRegex},
		{"Internet Explorer", tridentRegex},
	}
	for _, m := range versionMatchers {
		if info, ok := matchBrowserVersion(userAgent, m.name, m.regex); ok {
			return info
		}
	}

	if strings.Contains(ua, "safari/") {
		if info, ok := matchBrowserVersion(userAgent, "Safari", versionRegex); ok {
			return info
		}
		if info, ok := matchBrowserVersion(userAgent, "Safari", safariRegex); ok {
			return info
		}
		return BrowserInfo{IsBrowser: true, Name: "Safari"}
	}
	if fastBrowserCheck(ua) {
		return BrowserInfo{IsBrowser: true, Name: "Unknown"}
	}
	return BrowserInfo{IsBrowser: false}
}

// IsBrowser 返回启发式识别结果，不是安全判定。
func IsBrowser(userAgent string) bool {
	return GetBrowserInfo(userAgent).IsBrowser
}

// GetBrowserInfo 获取启发式浏览器信息。
func GetBrowserInfo(userAgent string) BrowserInfo {
	if userAgent == "" {
		return BrowserInfo{IsBrowser: false}
	}
	cacheable := len(userAgent) <= MaxUserAgentLength
	cache := getBrowserInfoCache()
	if cacheable {
		if result, ok := cache.get(userAgent); ok {
			return result
		}
	}
	result := parseBrowserInfo(userAgent)
	if cacheable {
		_ = cache.put(userAgent, result)
	}
	return result
}
