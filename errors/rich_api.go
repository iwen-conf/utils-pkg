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

func newRichWithSkip(code int, msg string, skip int) *RichError {
	return &RichError{
		Status: Status{
			Code: code,
			Msg:  msg,
		},
		cause: nil,
		stack: callersSkip(skip),
	}
}

func wrapRichWithSkip(err error, code int, skip int, format string, args ...interface{}) *RichError {
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
		stack: callersSkip(skip),
	}
}

// NewRich 创建一个纯业务错误 (Service 层使用)
func NewRich(code int, msg string) *RichError {
	return newRichWithSkip(code, msg, 4)
}

// NewRichNoStack 创建不捕获堆栈的业务错误（高性能场景）
func NewRichNoStack(code int, msg string) *RichError {
	return &RichError{
		Status: Status{
			Code: code,
			Msg:  msg,
		},
		cause: nil,
		stack: nil,
	}
}

// WrapRich 包装一个底层错误 (Repo 层使用)
func WrapRich(err error, code int, format string, args ...interface{}) *RichError {
	return wrapRichWithSkip(err, code, 4, format, args...)
}

// WrapRichNoStack 包装错误但不捕获堆栈（高性能场景）
func WrapRichNoStack(err error, code int, format string, args ...interface{}) *RichError {
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
		stack: nil,
	}
}

// FromRichError 智能转换 (Controller/Response 层使用)
// 把任意 error 还原成 *RichError，非 RichError 转为系统内部错误。
func FromRichError(err error) *RichError {
	if err == nil {
		return nil
	}

	if e, ok := asRichError(err); ok {
		return e
	}

	return &RichError{
		Status: Status{
			Code: RichCodeInternal,
			Msg:  RichMsgInternal,
		},
		cause: err,
		stack: callersSkip(3),
	}
}

// ==================== HTTP 状态码映射 ====================

// HTTPStatus 根据业务码自动推导 HTTP 状态码（取前 3 位，0 返回 200）
func (e *RichError) HTTPStatus() int {
	if e == nil || e.Code == 0 {
		return 200
	}
	httpCode := e.Code / 1000
	if httpCode >= 100 && httpCode <= 599 {
		return httpCode
	}
	return 500
}

// ==================== 快捷构造函数 ====================

// RichBadRequest 创建参数错误
func RichBadRequest(msg string) *RichError {
	return newRichWithSkip(RichCodeBadRequest, msg, 4)
}

// RichUnauthorized 创建未认证错误
func RichUnauthorized() *RichError {
	return newRichWithSkip(RichCodeUnauthorized, RichMsgUnauthorized, 4)
}

// RichForbidden 创建无权限错误
func RichForbidden() *RichError {
	return newRichWithSkip(RichCodeForbidden, RichMsgForbidden, 4)
}

// RichNotFound 创建资源不存在错误
func RichNotFound(resource string) *RichError {
	msg := RichMsgNotFound
	if resource != "" {
		msg = resource + "不存在"
	}
	return newRichWithSkip(RichCodeNotFound, msg, 4)
}

// RichInternal 创建系统内部错误（隐藏底层错误）
func RichInternal(err error) *RichError {
	return wrapRichWithSkip(err, RichCodeInternal, 4, RichMsgInternal)
}

// RichDBError 创建数据库错误
func RichDBError(err error) *RichError {
	return wrapRichWithSkip(err, RichCodeDBError, 4, RichMsgInternal)
}

// ==================== 判断函数 ====================

// RichErrorCode 返回业务码，非 RichError 返回默认值
func RichErrorCode(err error, defaultCode int) int {
	if err == nil {
		return 0
	}
	if e, ok := asRichError(err); ok {
		return e.Code
	}
	return defaultCode
}

// IsRichErrorCode 判断是否是指定业务码
func IsRichErrorCode(err error, code int) bool {
	if err == nil {
		return false
	}
	if e, ok := asRichError(err); ok {
		return e.Code == code
	}
	return false
}

// IsClientError 判断是否是客户端错误 (4xx)
func IsClientError(err error) bool {
	if e, ok := asRichError(err); ok {
		return e.Code >= 400000 && e.Code < 500000
	}
	return false
}

// IsServerError 判断是否是服务端错误 (5xx)
func IsServerError(err error) bool {
	if e, ok := asRichError(err); ok {
		return e.Code >= 500000
	}
	return false
}

// ==================== 链式方法 ====================

// WithCode 修改业务码（返回新对象；nil 接收者安全）
func (e *RichError) WithCode(code int) *RichError {
	if e == nil {
		return &RichError{
			Status: Status{Code: code, Msg: RichMsgInternal},
		}
	}
	return &RichError{
		Status: Status{Code: code, Msg: e.Msg},
		cause:  e.cause,
		stack:  e.stack,
	}
}

// WithMsg 修改提示语（返回新对象；nil 接收者安全）
func (e *RichError) WithMsg(msg string) *RichError {
	if e == nil {
		return &RichError{
			Status: Status{Code: RichCodeInternal, Msg: msg},
		}
	}
	return &RichError{
		Status: Status{Code: e.Code, Msg: msg},
		cause:  e.cause,
		stack:  e.stack,
	}
}
