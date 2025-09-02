package errors

import (
	"fmt"
	"time"
)

// New 使用给定的错误码和消息创建新的错误
func New(code, message string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// NewWithDetails 使用错误码、消息和详情创建新的错误
func NewWithDetails(code, message, details string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// Wrap 包装现有错误并添加上下文
func Wrap(err error, code, message string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
		Original:  err,
	}
}

// WrapWithDetails 包装现有错误并添加错误码、消息和详情
func WrapWithDetails(err error, code, message, details string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
		Original:  err,
	}
}

// FromType 从预定义的ErrorType创建新的错误
func FromType(errorType ErrorType) *Error {
	err := &Error{
		Code:      errorType.Code,
		Message:   errorType.Message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
	
	// 将严重级别和类别添加到上下文
	err.Context["severity"] = errorType.Severity
	err.Context["category"] = errorType.Category
	
	return err
}

// FromTypeWithDetails 从预定义的ErrorType创建带额外详情的错误
func FromTypeWithDetails(errorType ErrorType, details string) *Error {
	err := FromType(errorType)
	err.Details = details
	return err
}

// WrapWithType 使用预定义的ErrorType包装现有错误
func WrapWithType(err error, errorType ErrorType) *Error {
	wrappedErr := &Error{
		Code:      errorType.Code,
		Message:   errorType.Message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
		Original:  err,
	}
	
	// 将严重级别和类别添加到上下文
	wrappedErr.Context["severity"] = errorType.Severity
	wrappedErr.Context["category"] = errorType.Category
	
	return wrappedErr
}

// Builder 提供用于构建错误的流式接口
type Builder struct {
	err *Error
}

// NewBuilder 创建新的错误构建器
func NewBuilder() *Builder {
	return &Builder{
		err: &Error{
			Timestamp: time.Now(),
			Context:   make(map[string]interface{}),
		},
	}
}

// Code 设置错误码
func (b *Builder) Code(code string) *Builder {
	b.err.Code = code
	return b
}

// Message 设置错误消息
func (b *Builder) Message(message string) *Builder {
	b.err.Message = message
	return b
}

// Details 设置错误详情
func (b *Builder) Details(details string) *Builder {
	b.err.Details = details
	return b
}

// Wrap 设置原始错误
func (b *Builder) Wrap(err error) *Builder {
	b.err.Original = err
	return b
}

// Context 添加上下文信息
func (b *Builder) Context(key string, value interface{}) *Builder {
	b.err.Context[key] = value
	return b
}

// Severity 设置错误严重级别
func (b *Builder) Severity(severity Severity) *Builder {
	b.err.Context["severity"] = severity
	return b
}

// Category 设置错误类别
func (b *Builder) Category(category Category) *Builder {
	b.err.Context["category"] = category
	return b
}

// UserID 将用户ID添加到上下文
func (b *Builder) UserID(userID string) *Builder {
	b.err.Context["user_id"] = userID
	return b
}

// RequestID 将请求ID添加到上下文
func (b *Builder) RequestID(requestID string) *Builder {
	b.err.Context["request_id"] = requestID
	return b
}

// Operation 将操作名称添加到上下文
func (b *Builder) Operation(operation string) *Builder {
	b.err.Context["operation"] = operation
	return b
}

// Build 返回构造的错误
func (b *Builder) Build() *Error {
	return b.err
}

// 常见错误类型的便捷函数

// Internal 创建一个内部服务器错误
func Internal(message string) *Error {
	return FromType(InternalError).WithMessage(message)
}

// NotFound 创建一个未找到错误
func NotFound(resource string) *Error {
	return FromType(NotFoundError).WithDetails(fmt.Sprintf("资源 '%s' 未找到", resource))
}

// Unauthorized 创建一个未授权错误
func Unauthorized(message string) *Error {
	return FromType(UnauthorizedError).WithMessage(message)
}

// Forbidden 创建一个禁止访问错误
func Forbidden(message string) *Error {
	return FromType(ForbiddenError).WithMessage(message)
}

// InvalidInput 创建一个无效输入错误
func InvalidInput(field, reason string) *Error {
	return FromType(InvalidInputError).
		WithDetails(fmt.Sprintf("字段 '%s': %s", field, reason)).
		WithContext("field", field)
}

// MissingField 创建一个缺少字段错误
func MissingField(field string) *Error {
	return FromType(MissingFieldError).
		WithDetails(fmt.Sprintf("必填字段 '%s' 缺失", field)).
		WithContext("field", field)
}

// Timeout 创建一个超时错误
func Timeout(operation string, duration time.Duration) *Error {
	return FromType(TimeoutError).
		WithDetails(fmt.Sprintf("操作 '%s' 在 %v 后超时", operation, duration)).
		WithContext("operation", operation).
		WithContext("timeout_duration", duration)
}

// Database 创建一个数据库错误
func Database(operation string, err error) *Error {
	return WrapWithType(err, DatabaseError).
		WithDetails(fmt.Sprintf("数据库操作 '%s' 失败", operation)).
		WithContext("operation", operation)
}

// Network 创建一个网络错误
func Network(operation string, err error) *Error {
	return WrapWithType(err, NetworkError).
		WithDetails(fmt.Sprintf("网络操作 '%s' 失败", operation)).
		WithContext("operation", operation)
}
