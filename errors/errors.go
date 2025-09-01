package errors

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// StackFrame 堆栈帧信息
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// Error 通用错误结构体
type Error struct {
	Code       string                 `json:"code"`        // 错误码
	Message    string                 `json:"message"`     // 错误消息
	Details    string                 `json:"details"`     // 详细错误信息
	Timestamp  time.Time              `json:"timestamp"`   // 错误发生时间
	Context    map[string]interface{} `json:"context"`     // 上下文信息
	Original   error                  `json:"original"`    // 原始错误
	StackTrace []StackFrame           `json:"stack_trace"` // 堆栈跟踪
}

// Error 实现 error 接口
func (e *Error) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *Error) Unwrap() error {
	return e.Original
}

// WithContext 添加上下文信息
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithStackTrace 添加堆栈跟踪
func (e *Error) WithStackTrace(skip int) *Error {
	e.StackTrace = captureStackTrace(skip + 1)
	return e
}

// GetStackTraceString 获取堆栈跟踪字符串
func (e *Error) GetStackTraceString() string {
	if len(e.StackTrace) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, frame := range e.StackTrace {
		builder.WriteString(fmt.Sprintf("#%d %s\n\t%s:%d\n", i, frame.Function, frame.File, frame.Line))
	}
	return builder.String()
}

// captureStackTrace 捕获堆栈跟踪
func captureStackTrace(skip int) []StackFrame {
	const maxDepth = 32
	pc := make([]uintptr, maxDepth)
	n := runtime.Callers(skip+2, pc)

	if n == 0 {
		return nil
	}

	pc = pc[:n]
	frames := runtime.CallersFrames(pc)

	var stackTrace []StackFrame
	for {
		frame, more := frames.Next()

		// 跳过runtime和标准库的帧
		if strings.Contains(frame.Function, "runtime.") {
			if !more {
				break
			}
			continue
		}

		stackTrace = append(stackTrace, StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})

		if !more {
			break
		}
	}

	return stackTrace
}

// NewError 创建新的错误（内部使用）
func NewError(code, message, details string, context map[string]interface{}, original error) *Error {
	if context == nil {
		context = make(map[string]interface{})
	}

	return &Error{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Context:   context,
		Original:  original,
	}
}

// New 创建新的错误
func New(code, message string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// NewWithDetails 创建带详细信息的错误
func NewWithDetails(code, message, details string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// NewWithStack 创建带堆栈跟踪的错误
func NewWithStack(code, message string) *Error {
	err := New(code, message)
	err.StackTrace = captureStackTrace(1)
	return err
}

// Wrap 包装现有错误
func Wrap(err error, code, message string) *Error {
	if err == nil {
		return nil
	}

	return &Error{
		Code:      code,
		Message:   message,
		Details:   err.Error(),
		Timestamp: time.Now(),
		Original:  err,
		Context:   make(map[string]interface{}),
	}
}

// WrapWithStack 包装现有错误并添加堆栈跟踪
func WrapWithStack(err error, code, message string) *Error {
	if err == nil {
		return nil
	}

	wrapped := Wrap(err, code, message)
	wrapped.StackTrace = captureStackTrace(1)
	return wrapped
}

// WrapWithDetails 包装现有错误并添加详细信息
func WrapWithDetails(err error, code, message, details string) *Error {
	if err == nil {
		return nil
	}

	return &Error{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Original:  err,
		Context:   make(map[string]interface{}),
	}
}

// FromCode 根据错误码创建错误
func FromCode(code string) *Error {
	message, exists := GetMessageByCode(code)
	if !exists {
		message = "Unknown error"
	}

	return &Error{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// FromCodeWithDetails 根据错误码创建带详细信息的错误
func FromCodeWithDetails(code, details string) *Error {
	message, exists := GetMessageByCode(code)
	if !exists {
		message = "Unknown error"
	}

	return &Error{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// Is 检查错误是否为特定类型
func Is(err error, target error) bool {
	if err == nil || target == nil {
		return err == target
	}

	// 检查是否为相同的Error实例
	e1, ok1 := err.(*Error)
	e2, ok2 := target.(*Error)

	if ok1 && ok2 {
		// 比较错误码
		return e1.Code == e2.Code
	}

	// 使用标准库的错误比较
	return err == target
}

// AsError 尝试将错误转换为Error类型
func AsError(err error) *Error {
	if err == nil {
		return nil
	}

	if e, ok := err.(*Error); ok {
		return e
	}

	return nil
}

// IsBusinessError 检查是否为业务错误
func IsBusinessError(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(*Error)
	return ok
}

// GetBusinessError 获取业务错误
func GetBusinessError(err error) *Error {
	if err == nil {
		return nil
	}

	if businessErr, ok := err.(*Error); ok {
		return businessErr
	}

	return nil
}

// GetErrorCode 获取错误码
func GetErrorCode(err error) string {
	if e := AsError(err); e != nil {
		return e.Code
	}
	return ""
}

// GetErrorMessage 获取错误消息
func GetErrorMessage(err error) string {
	if e := AsError(err); e != nil {
		return e.Message
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

// GetErrorDetails 获取错误详细信息
func GetErrorDetails(err error) string {
	if e := AsError(err); e != nil {
		return e.Details
	}
	return ""
}

// GetErrorContext 获取错误上下文
func GetErrorContext(err error) map[string]interface{} {
	if e := AsError(err); e != nil {
		return e.Context
	}
	return nil
}

// HasCode 检查错误是否具有特定错误码
func HasCode(err error, code string) bool {
	if e := AsError(err); e != nil {
		return e.Code == code
	}
	return false
}

// EnableStackTrace 全局开启堆栈跟踪
var EnableStackTrace = false

// SetEnableStackTrace 设置是否全局开启堆栈跟踪
func SetEnableStackTrace(enable bool) {
	EnableStackTrace = enable
}

// NewF 创建格式化错误消息
func NewF(code string, format string, args ...interface{}) *Error {
	message := fmt.Sprintf(format, args...)
	err := New(code, message)
	if EnableStackTrace {
		err.StackTrace = captureStackTrace(1)
	}
	return err
}

// WrapF 格式化包装错误
func WrapF(err error, code string, format string, args ...interface{}) *Error {
	if err == nil {
		return nil
	}
	message := fmt.Sprintf(format, args...)
	wrapped := Wrap(err, code, message)
	if EnableStackTrace {
		wrapped.StackTrace = captureStackTrace(1)
	}
	return wrapped
}
