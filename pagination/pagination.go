package pagination

import (
	"fmt"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
)

// DefaultMaxLimit 默认的最大limit值
const DefaultMaxLimit = 100

// DefaultMinLimit 默认的最小limit值
const DefaultMinLimit = 1

// ParseLimitOffset 解析并验证limit和offset参数
// 返回验证后的limit, offset和可能的错误
func ParseLimitOffset(limitStr, offsetStr string) (int, int, error) {
	return ParseLimitOffsetWithRange(limitStr, offsetStr, DefaultMinLimit, DefaultMaxLimit)
}

// ParseLimitOffsetWithRange 解析并验证limit和offset参数，允许自定义范围
// 参数:
//   - limitStr: limit参数的字符串值
//   - offsetStr: offset参数的字符串值
//   - minLimit: 允许的最小limit值
//   - maxLimit: 允许的最大limit值
//
// 返回:
//   - int: 验证后的limit值
//   - int: 验证后的offset值
//   - error: 如果参数无效则返回错误
func ParseLimitOffsetWithRange(limitStr, offsetStr string, minLimit, maxLimit int) (int, int, error) {
	// 转换limit参数
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return 0, 0, fmt.Errorf("无效的limit参数: %w", err)
	}

	// 转换offset参数
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return 0, 0, fmt.Errorf("无效的offset参数: %w", err)
	}

	// 验证limit和offset的有效性
	if limit < minLimit || limit > maxLimit {
		return 0, 0, fmt.Errorf("limit必须在%d到%d之间", minLimit, maxLimit)
	}
	if offset < 0 {
		return 0, 0, fmt.Errorf("offset不能为负数")
	}

	return limit, offset, nil
}

// GetPaginationParams 从map中获取并解析分页参数
// 参数:
//   - params: 包含分页参数的map
//   - limitKey: limit参数的键名，默认为"limit"
//   - offsetKey: offset参数的键名，默认为"offset"
//
// 返回:
//   - int: 验证后的limit值
//   - int: 验证后的offset值
//   - error: 如果参数无效则返回错误
func GetPaginationParams(params map[string]string, limitKey, offsetKey string) (int, int, error) {
	if limitKey == "" {
		limitKey = "limit"
	}
	if offsetKey == "" {
		offsetKey = "offset"
	}

	limitStr, ok := params[limitKey]
	if !ok {
		limitStr = strconv.Itoa(DefaultMaxLimit) // 默认使用最大值
	}

	offsetStr, ok := params[offsetKey]
	if !ok {
		offsetStr = "0" // 默认从0开始
	}

	return ParseLimitOffset(limitStr, offsetStr)
}

// GetPaginationParamsFromContext 从Hertz的Context中获取分页参数
// 参数:
//   - c: Hertz框架的Context对象
//
// 返回:
//   - int: 验证后的limit值
//   - int: 验证后的offset值
//   - error: 如果参数无效则返回错误
func GetPaginationParamsFromContext(c *app.RequestContext) (limit int, offset int, err error) {
	limitStr := c.Query("limit")
	if limitStr == "" {
		limitStr = strconv.Itoa(DefaultMaxLimit) // 默认使用最大值
	}

	offsetStr := c.Query("offset")
	if offsetStr == "" {
		offsetStr = "0" // 默认从0开始
	}

	return ParseLimitOffset(limitStr, offsetStr)
}

// CalculateTotalPages 计算总页数
func CalculateTotalPages(totalItems, limit int) int {
	if limit <= 0 {
		return 0
	}

	totalPages := totalItems / limit
	if totalItems%limit > 0 {
		totalPages++
	}
	return totalPages
}

// CalculateCurrentPage 根据offset和limit计算当前页码（从1开始）
func CalculateCurrentPage(offset, limit int) int {
	if limit <= 0 {
		return 0
	}

	page := (offset / limit) + 1
	return page
}
