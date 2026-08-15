package useragent

import (
	"container/list"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	chromeRegex  = regexp.MustCompile(`(?i)chrome\/[\d.]+`)
	safariRegex  = regexp.MustCompile(`(?i)safari\/[\d.]+`)
	firefoxRegex = regexp.MustCompile(`(?i)firefox\/[\d.]+`)
	edgeRegex    = regexp.MustCompile(`(?i)(?:edge|edg)\/[\d.]+`)
	operaRegex   = regexp.MustCompile(`(?i)opr\/[\d.]+`)
	ieRegex      = regexp.MustCompile(`(?i)msie [\d.]+|trident\/[\d.]+`)
	versionRegex = regexp.MustCompile(`(?i)version\/[\d.]+`)

	botIdentifiers = map[string]bool{
		"bot":                 true,
		"spider":              true,
		"crawler":             true,
		"slurp":               true,
		"bingbot":             true,
		"baiduspider":         true,
		"yandexbot":           true,
		"sogou":               true,
		"exabot":              true,
		"facebookexternalhit": true,
		"ia_archiver":         true,
	}

	commonBrowserIdentifiers = map[string]bool{
		"mozilla":   true,
		"webkit":    true,
		"gecko":     true,
		"presto":    true,
		"trident":   true,
		"chrome":    true,
		"safari":    true,
		"firefox":   true,
		"edge":      true,
		"opera":     true,
		"msie":      true,
		"opr":       true,
		"mobile":    true,
		"android":   true,
		"iphone":    true,
		"ipad":      true,
		"ipod":      true,
		"samsung":   true,
		"miui":      true,
		"ucbrowser": true,
		"qqbrowser": true,
		"maxthon":   true,
		"crios":     true,
		"fxios":     true,
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
	var sum uint32
	for i := 0; i < len(key); i++ {
		sum += uint32(key[i])
	}
	return sc.shards[sum%16]
}

// Get 从分片缓存获取值
func (sc *ShardedCache) Get(key string) (any, bool) {
	return sc.getShard(key).Get(key)
}

// Put 设置分片缓存值
func (sc *ShardedCache) Put(key string, value any) error {
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

func fastBrowserCheck(ua string) bool {
	for browser := range commonBrowserIdentifiers {
		if strings.Contains(ua, browser) {
			return true
		}
	}
	return false
}

func fastBotCheck(ua string) bool {
	for bot := range botIdentifiers {
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
	ua := strings.ToLower(userAgent)
	if fastBotCheck(ua) {
		return BrowserInfo{IsBrowser: false}
	}
	switch {
	case strings.Contains(ua, "edg/") || strings.Contains(ua, "edge/"):
		return extractBrowserInfo(userAgent, "Edge", edgeRegex)
	case strings.Contains(ua, "opr/"):
		return extractBrowserInfo(userAgent, "Opera", operaRegex)
	case chromeRegex.MatchString(userAgent) && !strings.Contains(ua, "edg/") && !strings.Contains(ua, "opr/"):
		return extractBrowserInfo(userAgent, "Chrome", chromeRegex)
	case firefoxRegex.MatchString(userAgent):
		return extractBrowserInfo(userAgent, "Firefox", firefoxRegex)
	case ieRegex.MatchString(userAgent):
		return extractBrowserInfo(userAgent, "Internet Explorer", ieRegex)
	case strings.Contains(ua, "safari/") && !strings.Contains(ua, "chrome/"):
		if versionRegex.MatchString(userAgent) {
			return extractBrowserInfo(userAgent, "Safari", versionRegex)
		}
		return extractBrowserInfo(userAgent, "Safari", safariRegex)
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
	if result, ok := browserInfoCache.Get(userAgent); ok {
		return result.(BrowserInfo)
	}
	result := parseBrowserInfo(userAgent)
	_ = browserInfoCache.Put(userAgent, result)
	return result
}

func extractBrowserInfo(userAgent, browserName string, regex *regexp.Regexp) BrowserInfo {
	match := regex.FindString(userAgent)
	version := ""
	if match != "" {
		parts := strings.Split(match, "/")
		if len(parts) > 1 {
			version = parts[1]
		}
	}
	return BrowserInfo{IsBrowser: true, Name: browserName, Version: version}
}
