package errors

import "sync"

// ErrorRegistry 错误码注册表
type ErrorRegistry struct {
	mu       sync.RWMutex
	codes    map[string]string // 错误码到消息的映射
	prefixes map[string]string // 前缀到分类的映射
}

// 全局错误注册表
var globalRegistry = &ErrorRegistry{
	codes:    make(map[string]string),
	prefixes: make(map[string]string),
}

// RegisterErrorCode 注册错误码
func RegisterErrorCode(code, message string) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.codes[code] = message
}

// RegisterErrorCodes 批量注册错误码
func RegisterErrorCodes(codes map[string]string) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	for code, message := range codes {
		globalRegistry.codes[code] = message
	}
}

// RegisterErrorPrefix 注册错误码前缀分类
func RegisterErrorPrefix(prefix, category string) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.prefixes[prefix] = category
}

// GetMessageByCode 根据错误码获取错误消息
func GetMessageByCode(code string) (string, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	message, exists := globalRegistry.codes[code]
	return message, exists
}

// GetCategoryByCode 根据错误码获取分类
func GetCategoryByCode(code string) string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	// 根据前缀查找分类
	for prefix, category := range globalRegistry.prefixes {
		if len(code) >= len(prefix) && code[:len(prefix)] == prefix {
			return category
		}
	}
	return "unknown"
}

// IsSystemError 判断是否为系统级错误
func IsSystemError(code string) bool {
	return len(code) >= 1 && code[0] == '5'
}

// IsClientError 判断是否为客户端错误
func IsClientError(code string) bool {
	return len(code) >= 1 && code[0] == '4'
}

// IsBusinessErrorCode 判断是否为业务错误码
func IsBusinessErrorCode(code string) bool {
	return len(code) >= 1 && code[0] == '6'
}

// IsRetryableErrorCode 判断错误码是否为可重试的
func IsRetryableErrorCode(code string) bool {
	// 系统级错误通常是可重试的
	if IsSystemError(code) {
		return true
	}

	// 客户端错误通常不可重试，除了超时
	if IsClientError(code) {
		return code == CodeRequestTimeout
	}

	return false
}

// IsPermanentErrorCode 判断错误码是否为永久性的
func IsPermanentErrorCode(code string) bool {
	// 权限相关错误通常是永久的
	if code == CodeUnauthorized || code == CodeForbidden {
		return true
	}

	// 资源冲突通常是永久的
	if code == CodeConflict {
		return true
	}

	return false
}

// GetAllErrorCodes 获取所有已注册的错误码
func GetAllErrorCodes() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	codes := make([]string, 0, len(globalRegistry.codes))
	for code := range globalRegistry.codes {
		codes = append(codes, code)
	}
	return codes
}

// ClearRegistry 清空注册表（用于测试）
func ClearRegistry() {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.codes = make(map[string]string)
	globalRegistry.prefixes = make(map[string]string)
}

// 预定义的通用错误码（可选使用）
const (
	// 通用成功
	CodeSuccess = "0"

	// 客户端错误 (4xxx)
	CodeBadRequest       = "4000"
	CodeUnauthorized     = "4001"
	CodeForbidden        = "4003"
	CodeNotFound         = "4004"
	CodeMethodNotAllowed = "4005"
	CodeRequestTimeout   = "4008"
	CodeConflict         = "4009"
	CodeTooManyRequests  = "4029"

	// 服务端错误 (5xxx)
	CodeInternalError      = "5000"
	CodeNotImplemented     = "5001"
	CodeBadGateway         = "5002"
	CodeServiceUnavailable = "5003"
	CodeGatewayTimeout     = "5004"

	// 业务错误 (6xxx)
	CodeBusinessError   = "6000"
	CodeValidationError = "6001"
	CodeDataNotFound    = "6002"
	CodeDataExists      = "6003"
	CodeOperationFailed = "6004"
)

// 初始化默认错误码（可选）
func init() {
	// 注册通用错误码
	RegisterErrorCodes(map[string]string{
		CodeSuccess:            "成功",
		CodeBadRequest:         "请求错误",
		CodeUnauthorized:       "未授权",
		CodeForbidden:          "禁止访问",
		CodeNotFound:           "资源不存在",
		CodeMethodNotAllowed:   "方法不允许",
		CodeRequestTimeout:     "请求超时",
		CodeConflict:           "资源冲突",
		CodeTooManyRequests:    "请求过于频繁",
		CodeInternalError:      "内部服务器错误",
		CodeNotImplemented:     "功能未实现",
		CodeBadGateway:         "网关错误",
		CodeServiceUnavailable: "服务不可用",
		CodeGatewayTimeout:     "网关超时",
		CodeBusinessError:      "业务错误",
		CodeValidationError:    "数据验证错误",
		CodeDataNotFound:       "数据不存在",
		CodeDataExists:         "数据已存在",
		CodeOperationFailed:    "操作失败",
	})

	// 注册错误码前缀分类
	RegisterErrorPrefix("4", "client")
	RegisterErrorPrefix("5", "server")
	RegisterErrorPrefix("6", "business")
}
