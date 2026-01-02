package errors

import (
	"fmt"
)

// ==================== RichError 预定义业务码 ====================
// 业务码规范：HTTP状态码(3位) + 模块码(3位)
// 例如：400001 = 400(BadRequest) + 001(通用参数错误)

const (
	// 成功
	RichCodeSuccess = 0

	// 4xx 客户端错误
	RichCodeBadRequest   = 400000 // 通用参数错误
	RichCodeUnauthorized = 401000 // 未认证
	RichCodeForbidden    = 403000 // 无权限
	RichCodeNotFound     = 404000 // 资源不存在
	RichCodeConflict     = 409000 // 资源冲突

	// 5xx 服务端错误
	RichCodeInternal    = 500000 // 系统内部错误
	RichCodeDBError     = 500001 // 数据库错误
	RichCodeCacheError  = 500002 // 缓存错误
	RichCodeRPCError    = 500003 // RPC调用错误
	RichCodeExternalAPI = 500004 // 外部API错误
)

// 预定义错误消息
const (
	RichMsgSuccess      = "操作成功"
	RichMsgBadRequest   = "请求参数错误"
	RichMsgUnauthorized = "请先登录"
	RichMsgForbidden    = "无权限访问"
	RichMsgNotFound     = "资源不存在"
	RichMsgInternal     = "系统繁忙，请稍后重试"
)

// ==================== 核心 API ====================

// NewRich 创建一个纯业务错误 (Service 层使用)
// 场景：参数校验失败、逻辑不满足
func NewRich(code int, msg string) *RichError {
	return &RichError{
		Status: Status{
			Code: code,
			Msg:  msg,
		},
		cause: nil,
		stack: callers(),
	}
}

// WrapRich 包装一个底层错误 (Repo 层使用)
// 场景：数据库报错、第三方 API 报错
// 作用：把脏错误藏在 cause 里，给外面返回一个干净的 code/msg
func WrapRich(err error, code int, format string, args ...interface{}) *RichError {
	if err == nil {
		return nil
	}

	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	return &RichError{
		Status: Status{
			Code: code,
			Msg:  msg,
		},
		cause: err,
		stack: callers(),
	}
}

// FromRichError 智能转换 (Controller/Response 层使用)
// 作用：把任意 error 还原成 *RichError，非 RichError 转为系统错误
func FromRichError(err error) *RichError {
	if err == nil {
		return nil
	}

	// 已经是 *RichError，直接返回
	if e, ok := err.(*RichError); ok {
		return e
	}

	// 其他错误统一包装成系统内部错误
	return &RichError{
		Status: Status{
			Code: RichCodeInternal,
			Msg:  RichMsgInternal,
		},
		cause: err,
		stack: callers(),
	}
}

// ==================== HTTP 状态码映射 ====================

// HTTPStatus 根据业务码自动推导 HTTP 状态码
// 规则：取业务码前3位作为 HTTP 状态码，0 返回 200
func (e *RichError) HTTPStatus() int {
	if e.Code == 0 {
		return 200
	}
	// 取前3位：500001 -> 500, 404001 -> 404
	httpCode := e.Code / 1000
	if httpCode >= 100 && httpCode <= 599 {
		return httpCode
	}
	return 500 // 默认服务端错误
}

// ==================== 快捷构造函数 ====================

// RichBadRequest 创建参数错误
func RichBadRequest(msg string) *RichError {
	return NewRich(RichCodeBadRequest, msg)
}

// RichUnauthorized 创建未认证错误
func RichUnauthorized() *RichError {
	return NewRich(RichCodeUnauthorized, RichMsgUnauthorized)
}

// RichForbidden 创建无权限错误
func RichForbidden() *RichError {
	return NewRich(RichCodeForbidden, RichMsgForbidden)
}

// RichNotFound 创建资源不存在错误
func RichNotFound(resource string) *RichError {
	msg := RichMsgNotFound
	if resource != "" {
		msg = resource + "不存在"
	}
	return NewRich(RichCodeNotFound, msg)
}

// RichInternal 创建系统内部错误（隐藏底层错误）
func RichInternal(err error) *RichError {
	return WrapRich(err, RichCodeInternal, RichMsgInternal)
}

// RichDBError 创建数据库错误
func RichDBError(err error) *RichError {
	return WrapRich(err, RichCodeDBError, RichMsgInternal)
}

// ==================== 判断函数 ====================

// RichErrorCode 返回业务码，非 RichError 返回默认值
func RichErrorCode(err error, defaultCode int) int {
	if err == nil {
		return 0
	}
	if e, ok := err.(*RichError); ok {
		return e.Code
	}
	return defaultCode
}

// IsRichErrorCode 判断是否是指定业务码
func IsRichErrorCode(err error, code int) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*RichError); ok {
		return e.Code == code
	}
	return false
}

// IsClientError 判断是否是客户端错误 (4xx)
func IsClientError(err error) bool {
	if e, ok := err.(*RichError); ok {
		return e.Code >= 400000 && e.Code < 500000
	}
	return false
}

// IsServerError 判断是否是服务端错误 (5xx)
func IsServerError(err error) bool {
	if e, ok := err.(*RichError); ok {
		return e.Code >= 500000
	}
	return false
}

// ==================== 链式方法 ====================

// WithCode 修改业务码（返回新对象）
func (e *RichError) WithCode(code int) *RichError {
	return &RichError{
		Status: Status{Code: code, Msg: e.Msg},
		cause:  e.cause,
		stack:  e.stack,
	}
}

// WithMsg 修改提示语（返回新对象）
func (e *RichError) WithMsg(msg string) *RichError {
	return &RichError{
		Status: Status{Code: e.Code, Msg: msg},
		cause:  e.cause,
		stack:  e.stack,
	}
}
