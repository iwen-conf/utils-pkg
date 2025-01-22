package url

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// URLBuilder URL 构建器
type URLBuilder struct {
	baseURL    string
	params     map[string][]string
	fragment   string
	sortParams bool
	secretKey  string
	timestamp  int64
}

// NewURLBuilder 创建新的 URL 构建器
func NewURLBuilder(baseURL string, secretKey string) *URLBuilder {
	return &URLBuilder{
		baseURL:    baseURL,
		params:     make(map[string][]string),
		sortParams: true,
		secretKey:  secretKey,
		timestamp:  time.Now().Unix(),
	}
}

// AddParam 添加查询参数
func (b *URLBuilder) AddParam(key string, value string) *URLBuilder {
	if b.params[key] == nil {
		b.params[key] = make([]string, 0)
	}
	b.params[key] = append(b.params[key], value)
	return b
}

// AddParams 批量添加查询参数
func (b *URLBuilder) AddParams(params map[string]string) *URLBuilder {
	for k, v := range params {
		b.AddParam(k, v)
	}
	return b
}

// SetFragment 设置 URL 片段
func (b *URLBuilder) SetFragment(fragment string) *URLBuilder {
	b.fragment = fragment
	return b
}

// SetSortParams 设置是否对参数进行排序
func (b *URLBuilder) SetSortParams(sort bool) *URLBuilder {
	b.sortParams = sort
	return b
}

// Build 构建完整的 URL
// SetTimestamp 设置时间戳
func (b *URLBuilder) SetTimestamp(timestamp int64) *URLBuilder {
	b.timestamp = timestamp
	return b
}

// generateSignature 生成签名
func (b *URLBuilder) generateSignature(queryString string) string {
	// 组合待签名字符串：时间戳 + 查询字符串
	signStr := fmt.Sprintf("%d%s", b.timestamp, queryString)

	// 使用 HMAC-SHA256 算法生成签名
	h := hmac.New(sha256.New, []byte(b.secretKey))
	h.Write([]byte(signStr))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

// Build 构建完整的 URL
func (b *URLBuilder) Build() (string, error) {
	baseURL, err := url.Parse(b.baseURL)
	if err != nil {
		return "", err
	}

	// 处理查询参数
	query := baseURL.Query()
	if b.sortParams {
		// 按键排序参数
		keys := make([]string, 0, len(b.params))
		for k := range b.params {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			for _, v := range b.params[k] {
				query.Add(k, v)
			}
		}
	} else {
		for k, values := range b.params {
			for _, v := range values {
				query.Add(k, v)
			}
		}
	}

	// 添加时间戳参数
	query.Set("_ts", fmt.Sprintf("%d", b.timestamp))

	// 生成查询字符串
	queryStr := query.Encode()

	// 生成并添加签名
	signature := b.generateSignature(queryStr)
	query.Set("_sign", signature)

	// 设置最终的查询字符串
	baseURL.RawQuery = query.Encode()
	if b.fragment != "" {
		baseURL.Fragment = b.fragment
	}

	return baseURL.String(), nil
}

// ParseURL 解析 URL 并返回其组成部分
func ParseURL(rawURL string) (map[string]interface{}, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	result["scheme"] = parsedURL.Scheme
	result["host"] = parsedURL.Host
	result["path"] = parsedURL.Path
	result["fragment"] = parsedURL.Fragment

	// 解析查询参数
	queryParams := make(map[string]interface{})
	for k, v := range parsedURL.Query() {
		if len(v) == 1 {
			queryParams[k] = v[0]
		} else {
			queryParams[k] = v
		}
	}
	result["query"] = queryParams

	return result, nil
}

// SerializeParams 序列化参数为 URL 查询字符串
func SerializeParams(params map[string]interface{}) string {
	if len(params) == 0 {
		return ""
	}

	values := url.Values{}
	for k, v := range params {
		switch val := v.(type) {
		case string:
			values.Add(k, val)
		case []string:
			for _, item := range val {
				values.Add(k, item)
			}
		default:
			// 对于其他类型，尝试 JSON 序列化
			if jsonStr, err := json.Marshal(val); err == nil {
				values.Add(k, string(jsonStr))
			}
		}
	}

	return values.Encode()
}

// DeserializeParams 反序列化 URL 查询字符串为参数映射
// ValidateSignature 验证 URL 签名
func ValidateSignature(rawURL string, secretKey string, maxAgeSeconds int64) (bool, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, err
	}

	// 获取查询参数
	query := parsedURL.Query()

	// 获取时间戳和签名
	timestamp := query.Get("_ts")
	signature := query.Get("_sign")

	// 验证参数是否存在
	if timestamp == "" || signature == "" {
		return false, fmt.Errorf("missing timestamp or signature")
	}

	// 验证时间戳
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, err
	}

	// 检查时间戳是否过期
	currentTime := time.Now().Unix()
	if currentTime-ts > maxAgeSeconds {
		return false, fmt.Errorf("url expired")
	}

	// 移除签名参数后重新生成签名
	query.Del("_sign")
	queryStr := query.Encode()

	// 使用相同的算法生成签名
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(fmt.Sprintf("%d%s", ts, queryStr)))
	expectedSignature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// 比较签名
	return signature == expectedSignature, nil
}

// DeserializeParams 反序列化 URL 查询字符串为参数映射
func DeserializeParams(queryString string) map[string]interface{} {
	result := make(map[string]interface{})
	if queryString == "" {
		return result
	}

	// 移除开头的 '?' 如果存在
	queryString = strings.TrimPrefix(queryString, "?")

	values, err := url.ParseQuery(queryString)
	if err != nil {
		return result
	}

	for k, v := range values {
		if len(v) == 1 {
			result[k] = v[0]
		} else {
			result[k] = v
		}
	}

	return result
}
