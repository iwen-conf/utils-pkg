package useragent

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// 预编译正则表达式以提高性能
var (
	chromeRegex  = regexp.MustCompile(`(?i)chrome\/[\d.]+`)
	safariRegex  = regexp.MustCompile(`(?i)safari\/[\d.]+`)
	firefoxRegex = regexp.MustCompile(`(?i)firefox\/[\d.]+`)
	edgeRegex    = regexp.MustCompile(`(?i)(?:edge|edg)\/[\d.]+`)
	operaRegex   = regexp.MustCompile(`(?i)opr\/[\d.]+`)
	ieRegex      = regexp.MustCompile(`(?i)msie [\d.]+|trident\/[\d.]+`)

	// 使用 map 存储常见的爬虫标识，提高查找效率
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

	// 使用 map 存储常见浏览器标识，提高查找效率
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

// BrowserInfo 存储浏览器信息
type BrowserInfo struct {
	IsBrowser bool   // 是否是浏览器
	Name      string // 浏览器名称
	Version   string // 浏览器版本
}

// CacheEntry 缓存条目
type CacheEntry struct {
	value      interface{} // 存储的值
	expiration int64       // 过期时间
}

// LRUCache LRU缓存实现
type LRUCache struct {
	capacity    int                   // 最大容量
	mu          sync.RWMutex          // 读写锁
	cache       map[string]CacheEntry // 缓存数据
	keys        []string              // 按使用顺序存储的键列表
	ttl         int64                 // 过期时间（秒）
	cleanupTime int64                 // 上次清理时间
}

// NewLRUCache 创建一个新的LRU缓存
func NewLRUCache(capacity int, ttl int64) *LRUCache {
	return &LRUCache{
		capacity:    capacity,
		cache:       make(map[string]CacheEntry, capacity),
		keys:        make([]string, 0, capacity),
		ttl:         ttl,
		cleanupTime: time.Now().Unix(),
	}
}

// Get 获取缓存值
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// 检查是否过期
	now := time.Now().Unix()
	if entry.expiration > 0 && now > entry.expiration {
		c.mu.Lock()
		delete(c.cache, key)
		c.removeKey(key)
		c.mu.Unlock()
		return nil, false
	}

	// 将键移到最近使用的位置
	c.mu.Lock()
	c.moveToFront(key)

	// 每隔一段时间清理过期项
	if now-c.cleanupTime > 300 { // 每5分钟清理一次
		c.cleanup(now)
		c.cleanupTime = now
	}
	c.mu.Unlock()

	return entry.value, true
}

// Put 设置缓存值
func (c *LRUCache) Put(key string, value interface{}) {
	now := time.Now().Unix()
	expiration := int64(0)
	if c.ttl > 0 {
		expiration = now + c.ttl
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果键已存在，更新值并移动到前面
	if _, ok := c.cache[key]; ok {
		c.cache[key] = CacheEntry{value: value, expiration: expiration}
		c.moveToFront(key)
		return
	}

	// 如果达到容量，删除最久未使用的键
	if len(c.cache) >= c.capacity {
		leastUsed := c.keys[len(c.keys)-1]
		delete(c.cache, leastUsed)
		c.keys = c.keys[:len(c.keys)-1]
	}

	// 添加新键到缓存
	c.cache[key] = CacheEntry{value: value, expiration: expiration}
	c.keys = append([]string{key}, c.keys...)
}

// moveToFront 将键移到最近使用的位置
func (c *LRUCache) moveToFront(key string) {
	for i, k := range c.keys {
		if k == key {
			// 从当前位置删除
			c.keys = append(c.keys[:i], c.keys[i+1:]...)
			// 添加到最前面
			c.keys = append([]string{key}, c.keys...)
			break
		}
	}
}

// removeKey 从keys列表中删除键
func (c *LRUCache) removeKey(key string) {
	for i, k := range c.keys {
		if k == key {
			c.keys = append(c.keys[:i], c.keys[i+1:]...)
			break
		}
	}
}

// cleanup 清理过期项
func (c *LRUCache) cleanup(now int64) {
	for key, entry := range c.cache {
		if entry.expiration > 0 && now > entry.expiration {
			delete(c.cache, key)
			c.removeKey(key)
		}
	}
}

// ShardedCache 分片缓存，用于减少锁竞争
type ShardedCache struct {
	shards [16]*LRUCache // 16个分片
}

// NewShardedCache 创建一个新的分片缓存
func NewShardedCache(shardCapacity int, ttl int64) *ShardedCache {
	sc := &ShardedCache{}
	for i := 0; i < 16; i++ {
		sc.shards[i] = NewLRUCache(shardCapacity, ttl)
	}
	return sc
}

// getShard 获取键对应的分片
func (sc *ShardedCache) getShard(key string) *LRUCache {
	// 简单的哈希函数，用于确定分片
	var sum uint32
	for i := 0; i < len(key); i++ {
		sum += uint32(key[i])
	}
	return sc.shards[sum%16]
}

// Get 从分片缓存获取值
func (sc *ShardedCache) Get(key string) (interface{}, bool) {
	return sc.getShard(key).Get(key)
}

// Put 设置分片缓存值
func (sc *ShardedCache) Put(key string, value interface{}) {
	sc.getShard(key).Put(key, value)
}

// 创建分片缓存实例
var (
	// 1小时过期时间，每个分片容量1000
	isBrowserCache   = NewShardedCache(1000, 3600)
	browserInfoCache = NewShardedCache(1000, 3600)
)

// fastBrowserCheck 快速检查字符串中是否包含浏览器标识
func fastBrowserCheck(ua string) bool {
	for browser := range commonBrowserIdentifiers {
		if strings.Contains(ua, browser) {
			return true
		}
	}
	return false
}

// fastBotCheck 快速检查字符串中是否包含爬虫标识
func fastBotCheck(ua string) bool {
	for bot := range botIdentifiers {
		if strings.Contains(ua, bot) {
			return true
		}
	}
	return false
}

// IsBrowser 快速检查是否为浏览器请求
func IsBrowser(userAgent string) bool {
	if userAgent == "" {
		return false
	}

	// 检查缓存
	if result, ok := isBrowserCache.Get(userAgent); ok {
		return result.(bool)
	}

	// 转为小写（只进行一次转换）
	ua := strings.ToLower(userAgent)

	// 检查是否为爬虫或机器人
	if fastBotCheck(ua) {
		isBrowserCache.Put(userAgent, false)
		return false
	}

	// 检查是否包含任一浏览器标识
	isBrowser := fastBrowserCheck(ua)
	isBrowserCache.Put(userAgent, isBrowser)
	return isBrowser
}

// GetBrowserInfo 获取详细的浏览器信息
func GetBrowserInfo(userAgent string) BrowserInfo {
	if userAgent == "" {
		return BrowserInfo{IsBrowser: false}
	}

	// 检查缓存
	if result, ok := browserInfoCache.Get(userAgent); ok {
		return result.(BrowserInfo)
	}

	// 转为小写（只进行一次转换用于bot检查）
	ua := strings.ToLower(userAgent)

	// 检查是否为爬虫或机器人
	if fastBotCheck(ua) {
		result := BrowserInfo{IsBrowser: false}
		browserInfoCache.Put(userAgent, result)
		return result
	}

	// 按优先级检查浏览器类型 - 仅对原始字符串执行正则
	var result BrowserInfo
	switch {
	case edgeRegex.MatchString(userAgent):
		result = extractBrowserInfo(userAgent, "Edge", edgeRegex)
	case operaRegex.MatchString(userAgent):
		result = extractBrowserInfo(userAgent, "Opera", operaRegex)
	case chromeRegex.MatchString(userAgent):
		if !safariRegex.MatchString(userAgent) || !edgeRegex.MatchString(userAgent) && !operaRegex.MatchString(userAgent) {
			result = extractBrowserInfo(userAgent, "Chrome", chromeRegex)
		}
	case firefoxRegex.MatchString(userAgent):
		result = extractBrowserInfo(userAgent, "Firefox", firefoxRegex)
	case ieRegex.MatchString(userAgent):
		result = extractBrowserInfo(userAgent, "Internet Explorer", ieRegex)
	case safariRegex.MatchString(userAgent) && !chromeRegex.MatchString(userAgent):
		result = extractBrowserInfo(userAgent, "Safari", safariRegex)
	default:
		result = BrowserInfo{IsBrowser: fastBrowserCheck(ua)}
	}

	// 存入缓存
	browserInfoCache.Put(userAgent, result)
	return result
}

// extractBrowserInfo 从User-Agent中提取浏览器版本信息
func extractBrowserInfo(userAgent, browserName string, regex *regexp.Regexp) BrowserInfo {
	match := regex.FindString(userAgent)
	version := ""
	if match != "" {
		parts := strings.Split(match, "/")
		if len(parts) > 1 {
			version = parts[1]
		}
	}

	return BrowserInfo{
		IsBrowser: true,
		Name:      browserName,
		Version:   version,
	}
}
