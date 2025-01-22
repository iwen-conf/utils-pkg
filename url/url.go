package url

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"
)

// URLBuilder URL 构建器
type URLBuilder struct {
	baseURL    string
	params     map[string][]string
	fragment   string
	sortParams bool
}

// NewURLBuilder 创建新的 URL 构建器
func NewURLBuilder(baseURL string) *URLBuilder {
	return &URLBuilder{
		baseURL:    baseURL,
		params:     make(map[string][]string),
		sortParams: true,
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
