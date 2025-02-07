package useragent

import (
	"regexp"
	"strings"
	"sync"
)

// 预编译正则表达式以提高性能
var (
	chromeRegex  = regexp.MustCompile(`(?i)chrome\/[\d.]+`)
	safariRegex  = regexp.MustCompile(`(?i)safari\/[\d.]+`)
	firefoxRegex = regexp.MustCompile(`(?i)firefox\/[\d.]+`)
	edgeRegex    = regexp.MustCompile(`(?i)edge\/[\d.]+`)
	operaRegex   = regexp.MustCompile(`(?i)opr\/[\d.]+`)
	ieRegex      = regexp.MustCompile(`(?i)msie [\d.]+|trident\/[\d.]+`)

	// 使用 map 存储常见的爬虫标识，提高查找效率
	botIdentifiers = map[string]bool{
		"bot":                true,
		"spider":            true,
		"crawler":           true,
		"slurp":             true,
		"bingbot":           true,
		"baiduspider":       true,
		"yandexbot":         true,
		"sogou":             true,
		"exabot":            true,
		"facebookexternalhit": true,
		"ia_archiver":       true,
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

// 缓存结构
type cache struct {
	sync.RWMutex
	data map[string]interface{}
}

// 创建缓存实例
var (
	isBrowserCache = &cache{data: make(map[string]interface{})}
	browserInfoCache = &cache{data: make(map[string]interface{})}
)

// BrowserInfo 存储浏览器信息
type BrowserInfo struct {
	IsBrowser bool
	Name      string
	Version   string
}

// IsBrowser 快速检查是否为浏览器请求
func IsBrowser(userAgent string) bool {
	if userAgent == "" {
		return false
	}

	// 检查缓存
	isBrowserCache.RLock()
	if result, ok := isBrowserCache.data[userAgent].(bool); ok {
		isBrowserCache.RUnlock()
		return result
	}
	isBrowserCache.RUnlock()

	ua := strings.ToLower(userAgent)

	// 检查是否为爬虫或机器人
	for bot := range botIdentifiers {
		if strings.Contains(ua, bot) {
			// 存入缓存
			isBrowserCache.Lock()
			isBrowserCache.data[userAgent] = false
			isBrowserCache.Unlock()
			return false
		}
	}

	// 检查是否包含任一浏览器标识
	for browser := range commonBrowserIdentifiers {
		if strings.Contains(ua, browser) {
			// 存入缓存
			isBrowserCache.Lock()
			isBrowserCache.data[userAgent] = true
			isBrowserCache.Unlock()
			return true
		}
	}

	// 存入缓存
	isBrowserCache.Lock()
	isBrowserCache.data[userAgent] = false
	isBrowserCache.Unlock()
	return false
}

// GetBrowserInfo 获取详细的浏览器信息
func GetBrowserInfo(userAgent string) BrowserInfo {
	if userAgent == "" {
		return BrowserInfo{IsBrowser: false}
	}

	// 检查缓存
	browserInfoCache.RLock()
	if result, ok := browserInfoCache.data[userAgent].(BrowserInfo); ok {
		browserInfoCache.RUnlock()
		return result
	}
	browserInfoCache.RUnlock()

	// 检查是否为爬虫或机器人
	ua := strings.ToLower(userAgent)
	for bot := range botIdentifiers {
		if strings.Contains(ua, bot) {
			result := BrowserInfo{IsBrowser: false}
			// 存入缓存
			browserInfoCache.Lock()
			browserInfoCache.data[userAgent] = result
			browserInfoCache.Unlock()
			return result
		}
	}

	// 按优先级检查浏览器类型
	var result BrowserInfo
	switch {
	case edgeRegex.MatchString(userAgent):
		result = extractBrowserInfo(userAgent, "Edge", edgeRegex)
	case operaRegex.MatchString(userAgent):
		result = extractBrowserInfo(userAgent, "Opera", operaRegex)
	case chromeRegex.MatchString(userAgent) && !edgeRegex.MatchString(userAgent):
		result = extractBrowserInfo(userAgent, "Chrome", chromeRegex)
	case firefoxRegex.MatchString(userAgent):
		result = extractBrowserInfo(userAgent, "Firefox", firefoxRegex)
	case ieRegex.MatchString(userAgent):
		result = extractBrowserInfo(userAgent, "Internet Explorer", ieRegex)
	case safariRegex.MatchString(userAgent) && !chromeRegex.MatchString(userAgent):
		result = extractBrowserInfo(userAgent, "Safari", safariRegex)
	default:
		result = BrowserInfo{IsBrowser: IsBrowser(userAgent)}
	}

	// 存入缓存
	browserInfoCache.Lock()
	browserInfoCache.data[userAgent] = result
	browserInfoCache.Unlock()
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