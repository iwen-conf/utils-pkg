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
	// ErrInvalidCacheCapacity 表示 LRU 容量非法。
	ErrInvalidCacheCapacity = errors.New("invalid cache capacity")
	// ErrInvalidCacheTTL 表示 TTL 为负数。
	ErrInvalidCacheTTL = errors.New("invalid cache ttl")
	// ErrUninitializedCache 表示对 nil cache 调用 Put。
	ErrUninitializedCache = errors.New("uninitialized cache")
)

// BrowserInfo 存储启发式浏览器识别结果，不是安全判定。
type BrowserInfo struct {
	IsBrowser bool
	Name      string
	Version   string
}

type cacheEntry struct {
	key        string
	value      any
	expiration time.Time
}

// LRUCache 使用 list+map 的 O(1) LRU。
type LRUCache struct {
	capacity int
	ttl      time.Duration
	mu       sync.Mutex
	items    map[string]*list.Element
	order    *list.List
	now      func() time.Time
}

// NewLRUCache 创建 LRU 缓存。ttl==0 表示不过期。
func NewLRUCache(capacity int, ttl time.Duration) (*LRUCache, error) {
	if capacity <= 0 {
		return nil, ErrInvalidCacheCapacity
	}
	if ttl < 0 {
		return nil, ErrInvalidCacheTTL
	}
	return &LRUCache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*list.Element, capacity),
		order:    list.New(),
		now:      time.Now,
	}, nil
}

// Get 读取未过期缓存。
func (c *LRUCache) Get(key string) (any, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	entry := el.Value.(*cacheEntry)
	if c.ttl > 0 && !c.now().Before(entry.expiration) {
		c.removeElement(el)
		return nil, false
	}
	c.order.MoveToFront(el)
	return entry.value, true
}

// Put 写入缓存。
func (c *LRUCache) Put(key string, value any) error {
	if c == nil {
		return ErrUninitializedCache
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

func (c *LRUCache) removeElement(el *list.Element) {
	entry := el.Value.(*cacheEntry)
	delete(c.items, entry.key)
	c.order.Remove(el)
}

// ShardedCache 分片缓存。
type ShardedCache struct {
	shards [16]*LRUCache
}

// NewShardedCache 创建分片缓存。
func NewShardedCache(shardCapacity int, ttl time.Duration) (*ShardedCache, error) {
	sc := &ShardedCache{}
	for i := 0; i < 16; i++ {
		cache, err := NewLRUCache(shardCapacity, ttl)
		if err != nil {
			return nil, err
		}
		sc.shards[i] = cache
	}
	return sc, nil
}

func (sc *ShardedCache) getShard(key string) *LRUCache {
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

// Get 从分片缓存获取值（nil 接收者安全）。
func (sc *ShardedCache) Get(key string) (any, bool) {
	if sc == nil {
		return nil, false
	}
	return sc.getShard(key).Get(key)
}

// Put 设置分片缓存值（nil 接收者安全）。
func (sc *ShardedCache) Put(key string, value any) error {
	if sc == nil {
		return ErrUninitializedCache
	}
	return sc.getShard(key).Put(key, value)
}

var browserInfoCache *ShardedCache

func init() {
	cache, err := NewShardedCache(1000, time.Hour)
	if err != nil {
		panic(err)
	}
	browserInfoCache = cache
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

func parseBrowserInfo(userAgent string) BrowserInfo {
	if userAgent == "" {
		return BrowserInfo{}
	}
	userAgent = truncateUTF8Prefix(userAgent, MaxUserAgentLength)
	ua := strings.ToLower(userAgent)
	if fastBotCheck(ua) {
		return BrowserInfo{IsBrowser: false}
	}
	switch {
	case strings.Contains(ua, "edg/") || strings.Contains(ua, "edge/"):
		return extractSubmatchInfo(userAgent, "Edge", edgeRegex)
	case strings.Contains(ua, "opr/"):
		return extractSubmatchInfo(userAgent, "Opera", operaRegex)
	case chromeRegex.MatchString(userAgent):
		return extractSubmatchInfo(userAgent, "Chrome", chromeRegex)
	case firefoxRegex.MatchString(userAgent):
		return extractSubmatchInfo(userAgent, "Firefox", firefoxRegex)
	case msieRegex.MatchString(userAgent):
		return extractSubmatchInfo(userAgent, "Internet Explorer", msieRegex)
	case tridentRvRegex.MatchString(userAgent):
		return extractSubmatchInfo(userAgent, "Internet Explorer", tridentRvRegex)
	case tridentRegex.MatchString(userAgent):
		return extractSubmatchInfo(userAgent, "Internet Explorer", tridentRegex)
	case strings.Contains(ua, "safari/"):
		if versionRegex.MatchString(userAgent) {
			return extractSubmatchInfo(userAgent, "Safari", versionRegex)
		}
		return extractSubmatchInfo(userAgent, "Safari", safariRegex)
	default:
		if fastBrowserCheck(ua) {
			return BrowserInfo{IsBrowser: true, Name: "Unknown"}
		}
		return BrowserInfo{IsBrowser: false}
	}
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
	if cacheable {
		if result, ok := browserInfoCache.Get(userAgent); ok {
			return result.(BrowserInfo)
		}
	}
	result := parseBrowserInfo(userAgent)
	if cacheable {
		_ = browserInfoCache.Put(userAgent, result)
	}
	return result
}

func extractSubmatchInfo(userAgent, browserName string, regex *regexp.Regexp) BrowserInfo {
	submatch := regex.FindStringSubmatch(userAgent)
	version := ""
	if len(submatch) > 1 {
		version = submatch[1]
	}
	return BrowserInfo{IsBrowser: true, Name: browserName, Version: version}
}
