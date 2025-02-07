package useragent

import (
	"regexp"
	"strings"
)

// 预编译正则表达式以提高性能
var (
	chromeRegex  = regexp.MustCompile(`(?i)chrome\/[\d.]+`)
	safariRegex  = regexp.MustCompile(`(?i)safari\/[\d.]+`)
	firefoxRegex = regexp.MustCompile(`(?i)firefox\/[\d.]+`)
	edgeRegex    = regexp.MustCompile(`(?i)edge\/[\d.]+`)
	operaRegex   = regexp.MustCompile(`(?i)opr\/[\d.]+`)
	ieRegex      = regexp.MustCompile(`(?i)msie [\d.]+|trident\/[\d.]+`)
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

	ua := strings.ToLower(userAgent)

	// 检查是否为爬虫或机器人
	if strings.Contains(ua, "bot") ||
		strings.Contains(ua, "spider") ||
		strings.Contains(ua, "crawler") ||
		strings.Contains(ua, "slurp") ||
		strings.Contains(ua, "bingbot") ||
		strings.Contains(ua, "baiduspider") ||
		strings.Contains(ua, "yandexbot") ||
		strings.Contains(ua, "sogou") ||
		strings.Contains(ua, "exabot") ||
		strings.Contains(ua, "facebookexternalhit") ||
		strings.Contains(ua, "ia_archiver") {
		return false
	}

	// 检查常见浏览器引擎和标识
	commonBrowsers := []string{
		"mozilla", "webkit", "gecko", "presto", "trident", // 浏览器引擎
		"chrome", "safari", "firefox", "edge", "opera", "msie", "opr", // 桌面浏览器
		"mobile", "android", "iphone", "ipad", "ipod", // 移动端浏览器
		"samsung", "miui", "ucbrowser", "qqbrowser", "maxthon", // 其他常见浏览器
		"crios", "fxios", // iOS 上的 Chrome 和 Firefox
	}

	// 检查是否包含任一浏览器标识
	for _, browser := range commonBrowsers {
		if strings.Contains(ua, browser) {
			return true
		}
	}

	return false
}

// GetBrowserInfo 获取详细的浏览器信息
func GetBrowserInfo(userAgent string) BrowserInfo {
	if userAgent == "" {
		return BrowserInfo{IsBrowser: false}
	}

	// 检查是否为爬虫或机器人
	if strings.Contains(strings.ToLower(userAgent), "bot") ||
		strings.Contains(strings.ToLower(userAgent), "spider") ||
		strings.Contains(strings.ToLower(userAgent), "crawler") {
		return BrowserInfo{IsBrowser: false}
	}

	// 按优先级检查浏览器类型
	switch {
	case edgeRegex.MatchString(userAgent):
		return extractBrowserInfo(userAgent, "Edge", edgeRegex)
	case operaRegex.MatchString(userAgent):
		return extractBrowserInfo(userAgent, "Opera", operaRegex)
	case chromeRegex.MatchString(userAgent) && !edgeRegex.MatchString(userAgent):
		return extractBrowserInfo(userAgent, "Chrome", chromeRegex)
	case firefoxRegex.MatchString(userAgent):
		return extractBrowserInfo(userAgent, "Firefox", firefoxRegex)
	case ieRegex.MatchString(userAgent):
		return extractBrowserInfo(userAgent, "Internet Explorer", ieRegex)
	case safariRegex.MatchString(userAgent) && !chromeRegex.MatchString(userAgent):
		return extractBrowserInfo(userAgent, "Safari", safariRegex)
	default:
		return BrowserInfo{IsBrowser: IsBrowser(userAgent)}
	}
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