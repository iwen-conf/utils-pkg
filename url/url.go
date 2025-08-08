package url

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 预定义错误类型
var (
	ErrInvalidBaseURL   = errors.New("invalid base URL")
	ErrMissingTimestamp = errors.New("missing timestamp parameter")
	ErrMissingSignature = errors.New("missing signature parameter")
	ErrInvalidTimestamp = errors.New("invalid timestamp format")
	ErrExpiredURL       = errors.New("URL has expired")
	ErrFutureTimestamp  = errors.New("timestamp is in the future")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrEmptySecretKey   = errors.New("secret key cannot be empty")
)

// 时间戳允许的时钟漂移（秒）
const allowedTimeDrift = 300 // 5分钟

// URLBuilder URL 构建器
type URLBuilder struct {
	baseURL    string     // 基础URL
	params     url.Values // 使用url.Values替代map以避免类型转换
	fragment   string     // URL片段
	sortParams bool       // 是否排序参数
	secretKey  string     // 密钥
	timestamp  int64      // 时间戳
	expiration int64      // 过期时间（秒）
}

// NewURLBuilder 创建新的 URL 构建器
func NewURLBuilder(baseURL string, secretKey string) *URLBuilder {
	return &URLBuilder{
		baseURL:    baseURL,
		params:     make(url.Values),
		sortParams: true,
		secretKey:  secretKey,
		timestamp:  time.Now().Unix(),
		expiration: 3600, // 默认1小时过期
	}
}

// Validate 验证构建器的基本参数
func (b *URLBuilder) Validate() error {
	if b.baseURL == "" {
		return ErrInvalidBaseURL
	}
	if b.secretKey == "" {
		return ErrEmptySecretKey
	}
	if _, err := url.Parse(b.baseURL); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidBaseURL, err)
	}
	return nil
}

// AddParam 添加查询参数
func (b *URLBuilder) AddParam(key string, value string) *URLBuilder {
	b.params.Add(key, value)
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

// SetTimestamp 设置时间戳
func (b *URLBuilder) SetTimestamp(timestamp int64) *URLBuilder {
	b.timestamp = timestamp
	return b
}

// SetExpiration 设置URL过期时间（秒）
func (b *URLBuilder) SetExpiration(seconds int64) *URLBuilder {
	b.expiration = seconds
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
	// 基本验证
	if err := b.Validate(); err != nil {
		return "", err
	}

	baseURL, err := url.Parse(b.baseURL)
	if err != nil {
		return "", fmt.Errorf("无效的基础URL: %w", err)
	}

	// 预估URL长度
	estimatedQueryLength := len(b.params) * 20

	// 处理查询参数
	query := baseURL.Query()

	if b.sortParams {
		// 获取所有参数并排序
		keys := make([]string, 0, len(b.params))
		for k := range b.params {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// 按排序添加参数
		for _, k := range keys {
			values := b.params[k]
			for _, v := range values {
				query.Add(k, v)
			}
		}
	} else {
		// 直接合并参数
		for k, values := range b.params {
			for _, v := range values {
				query.Add(k, v)
			}
		}
	}

	// 添加时间戳参数
	query.Set("_ts", fmt.Sprintf("%d", b.timestamp))

	// 如果设置了过期时间，添加过期时间参数
	if b.expiration > 0 {
		query.Set("_exp", fmt.Sprintf("%d", b.expiration))
	}

	// 生成查询字符串
	queryStr := query.Encode()

	// 生成并添加签名
	signature := b.generateSignature(queryStr)
	query.Set("_sign", signature)

	// 构建最终URL
	var sb strings.Builder
	sb.Grow(len(baseURL.String()) + estimatedQueryLength + len(b.fragment) + 10)

	// 添加基础部分
	sb.WriteString(baseURL.Scheme)
	sb.WriteString("://")
	sb.WriteString(baseURL.Host)
	sb.WriteString(baseURL.Path)

	// 添加查询参数
	encodedQuery := query.Encode()
	if encodedQuery != "" {
		sb.WriteString("?")
		sb.WriteString(encodedQuery)
	}

	// 添加片段
	if b.fragment != "" {
		sb.WriteString("#")
		sb.WriteString(b.fragment)
	}

	return sb.String(), nil
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
		case int, int64, float64, bool:
			// 直接使用fmt.Sprint处理基本类型
			values.Add(k, fmt.Sprint(val))
		default:
			// 对于其他类型，尝试 JSON 序列化
			if jsonStr, err := json.Marshal(val); err == nil {
				values.Add(k, string(jsonStr))
			}
		}
	}

	return values.Encode()
}

// ValidateSignature 验证 URL 签名
func ValidateSignature(rawURL string, secretKey string, maxAgeSeconds int64) (bool, error) {
	if secretKey == "" {
		return false, ErrEmptySecretKey
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, fmt.Errorf("无效的URL: %w", err)
	}

	// 获取查询参数
	query := parsedURL.Query()

	// 获取时间戳和签名
	timestamp := query.Get("_ts")
	signature := query.Get("_sign")

	// 获取过期时间（如果有）
	expiration := int64(0)
	if exp := query.Get("_exp"); exp != "" {
		var err error
		expiration, err = strconv.ParseInt(exp, 10, 64)
		if err != nil {
			return false, fmt.Errorf("无效的过期时间: %w", err)
		}
	}

	// 验证参数是否存在
	if timestamp == "" {
		return false, ErrMissingTimestamp
	}
	if signature == "" {
		return false, ErrMissingSignature
	}

	// 验证时间戳
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, ErrInvalidTimestamp
	}

	// 检查时间戳
	currentTime := time.Now().Unix()

	// 检查时间戳是否在未来
	if ts > currentTime+allowedTimeDrift {
		return false, ErrFutureTimestamp
	}

	// 检查时间戳是否过期
	if expiration > 0 && currentTime > ts+expiration {
		return false, ErrExpiredURL
	} else if expiration == 0 && maxAgeSeconds > 0 && currentTime-ts > maxAgeSeconds {
		return false, ErrExpiredURL
	}

	// 移除签名参数后重新生成签名
	query.Del("_sign")
	queryStr := query.Encode()

	// 使用相同的算法生成签名
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(fmt.Sprintf("%d%s", ts, queryStr)))
	expectedSignature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// 使用恒定时间比较签名
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) != 1 {
		return false, ErrInvalidSignature
	}

	return true, nil
}

// BatchValidateSignatures 批量验证URL签名
func BatchValidateSignatures(urls []string, secretKey string, maxAgeSeconds int64) map[string]error {
	results := make(map[string]error, len(urls))

	for _, rawURL := range urls {
		_, err := ValidateSignature(rawURL, secretKey, maxAgeSeconds)
		results[rawURL] = err
	}

	return results
}

// DeserializeParams 反序列化 URL 查询字符串为参数映射
func DeserializeParams(queryString string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if queryString == "" {
		return result, nil
	}

	// 移除开头的 '?' 如果存在
	queryString = strings.TrimPrefix(queryString, "?")

	values, err := url.ParseQuery(queryString)
	if err != nil {
		return nil, fmt.Errorf("无效的查询字符串: %w", err)
	}

	for k, v := range values {
		if len(v) == 1 {
			result[k] = v[0]
		} else {
			result[k] = v
		}
	}

	return result, nil
}

// CreateSignedURL 快速创建带签名的URL（便捷方法）
func CreateSignedURL(baseURL, secretKey string, params map[string]string, expireSeconds int64) (string, error) {
	builder := NewURLBuilder(baseURL, secretKey)
	builder.AddParams(params)
	if expireSeconds > 0 {
		builder.SetExpiration(expireSeconds)
	}
	return builder.Build()
}
