package errors

import (
	"fmt"
	"strings"
)

// ErrorBuilder 错误构建器
type ErrorBuilder struct {
	code     string
	message  string
	details  string
	context  map[string]interface{}
	original error
}

// NewBuilder 创建新的错误构建器
func NewBuilder() *ErrorBuilder {
	return &ErrorBuilder{
		context: make(map[string]interface{}),
	}
}

// Code 设置错误码
func (b *ErrorBuilder) Code(code string) *ErrorBuilder {
	b.code = code
	return b
}

// Message 设置错误消息
func (b *ErrorBuilder) Message(message string) *ErrorBuilder {
	b.message = message
	return b
}

// Details 设置详细信息
func (b *ErrorBuilder) Details(details string) *ErrorBuilder {
	b.details = details
	return b
}

// Context 添加上下文信息
func (b *ErrorBuilder) Context(key string, value interface{}) *ErrorBuilder {
	b.context[key] = value
	return b
}

// ContextMap 批量添加上下文信息
func (b *ErrorBuilder) ContextMap(context map[string]interface{}) *ErrorBuilder {
	for k, v := range context {
		b.context[k] = v
	}
	return b
}

// Original 设置原始错误
func (b *ErrorBuilder) Original(err error) *ErrorBuilder {
	b.original = err
	return b
}

// Build 构建错误
func (b *ErrorBuilder) Build() *Error {
	// 如果没有设置消息，尝试从错误码获取
	if b.message == "" && b.code != "" {
		if message, exists := GetMessageByCode(b.code); exists {
			b.message = message
		}
	}

	return NewError(b.code, b.message, b.details, b.context, b.original)
}

// ErrorFormatter 错误格式化器接口
type ErrorFormatter interface {
	Format(err *Error) string
}

// DefaultFormatter 默认格式化器
type DefaultFormatter struct{}

// Format 格式化错误
func (f *DefaultFormatter) Format(err *Error) string {
	if err == nil {
		return ""
	}

	var parts []string

	if err.Code != "" {
		parts = append(parts, fmt.Sprintf("Code: %s", err.Code))
	}

	if err.Message != "" {
		parts = append(parts, fmt.Sprintf("Message: %s", err.Message))
	}

	if err.Details != "" {
		parts = append(parts, fmt.Sprintf("Details: %s", err.Details))
	}

	if err.Original != nil {
		parts = append(parts, fmt.Sprintf("Original: %s", err.Original.Error()))
	}

	return strings.Join(parts, " | ")
}

// JSONFormatter JSON格式化器
type JSONFormatter struct{}

// Format 格式化错误为JSON风格
func (f *JSONFormatter) Format(err *Error) string {
	if err == nil {
		return "{}"
	}

	parts := []string{
		fmt.Sprintf(`"code":"%s"`, err.Code),
		fmt.Sprintf(`"message":"%s"`, err.Message),
	}

	if err.Details != "" {
		parts = append(parts, fmt.Sprintf(`"details":"%s"`, err.Details))
	}

	if err.Original != nil {
		parts = append(parts, fmt.Sprintf(`"original":"%s"`, err.Original.Error()))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

var defaultFormatter ErrorFormatter = &DefaultFormatter{}

// SetDefaultFormatter 设置默认格式化器
func SetDefaultFormatter(formatter ErrorFormatter) {
	defaultFormatter = formatter
}

// FormatError 使用默认格式化器格式化错误
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	if businessErr := AsError(err); businessErr != nil {
		return defaultFormatter.Format(businessErr)
	}

	return err.Error()
}

// CollectErrors 收集多个错误
func CollectErrors(errs ...error) []error {
	var validErrors []error
	for _, err := range errs {
		if err != nil {
			validErrors = append(validErrors, err)
		}
	}
	return validErrors
}

// HasErrors 检查是否有错误
func HasErrors(errs ...error) bool {
	for _, err := range errs {
		if err != nil {
			return true
		}
	}
	return false
}

// FirstError 获取第一个非空错误
func FirstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// WrapMultiple 包装多个错误
func WrapMultiple(errs []error, code, message string) []*Error {
	var wrappedErrors []*Error
	for _, err := range errs {
		if err != nil {
			wrappedErrors = append(wrappedErrors, Wrap(err, code, message))
		}
	}
	return wrappedErrors
}

// ErrorHandler 错误处理器接口
type ErrorHandler func(err *Error) error

// ErrorHandlerChain 错误处理器链
type ErrorHandlerChain struct {
	handlers []ErrorHandler
}

// NewHandlerChain 创建新的处理器链
func NewHandlerChain() *ErrorHandlerChain {
	return &ErrorHandlerChain{
		handlers: make([]ErrorHandler, 0),
	}
}

// Add 添加处理器
func (c *ErrorHandlerChain) Add(handler ErrorHandler) *ErrorHandlerChain {
	c.handlers = append(c.handlers, handler)
	return c
}

// Handle 处理错误
func (c *ErrorHandlerChain) Handle(err *Error) error {
	for _, handler := range c.handlers {
		if handlerErr := handler(err); handlerErr != nil {
			return handlerErr
		}
	}
	return nil
}

// RetryableChecker 可重试检查器
type RetryableChecker func(err *Error) bool

var defaultRetryableChecker RetryableChecker = func(err *Error) bool {
	if err == nil {
		return false
	}

	// 根据错误码分类判断
	category := GetCategoryByCode(err.Code)

	// 服务端错误通常可重试
	if category == "server" {
		return true
	}

	// 客户端错误通常不可重试，除了超时
	if category == "client" {
		return err.Code == CodeRequestTimeout
	}

	return false
}

// SetRetryableChecker 设置可重试检查器
func SetRetryableChecker(checker RetryableChecker) {
	defaultRetryableChecker = checker
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if businessErr := AsError(err); businessErr != nil {
		return defaultRetryableChecker(businessErr)
	}

	return false
}

// ErrorAggregator 错误聚合器
type ErrorAggregator struct {
	errors []*Error
}

// NewAggregator 创建新的错误聚合器
func NewAggregator() *ErrorAggregator {
	return &ErrorAggregator{
		errors: make([]*Error, 0),
	}
}

// Add 添加错误
func (a *ErrorAggregator) Add(err error) {
	if err == nil {
		return
	}

	if businessErr := AsError(err); businessErr != nil {
		a.errors = append(a.errors, businessErr)
	} else {
		a.errors = append(a.errors, Wrap(err, CodeInternalError, "Aggregated error"))
	}
}

// HasErrors 是否有错误
func (a *ErrorAggregator) HasErrors() bool {
	return len(a.errors) > 0
}

// Errors 获取所有错误
func (a *ErrorAggregator) Errors() []*Error {
	return a.errors
}

// Error 实现error接口
func (a *ErrorAggregator) Error() string {
	if len(a.errors) == 0 {
		return ""
	}

	messages := make([]string, 0, len(a.errors))
	for _, err := range a.errors {
		messages = append(messages, err.Error())
	}

	return fmt.Sprintf("Multiple errors occurred: %s", strings.Join(messages, "; "))
}

// Clear 清空错误
func (a *ErrorAggregator) Clear() {
	a.errors = make([]*Error, 0)
}
