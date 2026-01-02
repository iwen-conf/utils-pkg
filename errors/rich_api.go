package errors

import (
	"fmt"
)

// NewRich 创建一个纯业务错误 (Service 层使用)
// 场景：参数校验失败、逻辑不满足
func NewRich(code int, msg string) *RichError {
	return &RichError{
		Status: Status{
			Code: code,
			Msg:  msg,
		},
		cause: nil,
		stack: callers(), // 自动记录当前代码位置
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
		cause: err,       // 原始错误被保存
		stack: callers(), // 记录 Wrap 发生时的堆栈
	}
}

// FromRichError 智能转换 (Controller 层使用)
// 作用：把任意 error 还原成 *RichError。如果还原不了，就变成"未知错误"。
func FromRichError(err error) *RichError {
	if err == nil {
		return nil
	}

	// 1. 如果已经是我们定义的 *RichError，直接强转返回
	if e, ok := err.(*RichError); ok {
		return e
	}

	// 2. 如果是其他错误 (标准库 error 或第三方库 error)
	// 统一包装成 500 系统错误
	return &RichError{
		Status: Status{
			Code: 500000, // 建议引用 constants.CodeServerErr
			Msg:  "系统内部错误",
		},
		cause: err,
		stack: callers(),
	}
}

// RichErrorCode 返回 RichError 的业务码，若不是 RichError 则返回默认值
func RichErrorCode(err error, defaultCode int) int {
	if err == nil {
		return 0
	}
	if e, ok := err.(*RichError); ok {
		return e.Code
	}
	return defaultCode
}

// IsRichErrorCode 判断 error 是否是指定业务码的 RichError
func IsRichErrorCode(err error, code int) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*RichError); ok {
		return e.Code == code
	}
	return false
}
