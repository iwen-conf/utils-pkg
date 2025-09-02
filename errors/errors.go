// Package errors 提供了一个全面的错误处理系统，支持错误码、上下文信息、错误链、验证和工具函数。
//
// 本包组织成以下几个模块：
//   - errors.go: 核心Error结构体和基本方法
//   - error_types.go: 预定义错误类型和常量
//   - error_builder.go: 错误构建和构建器模式
//   - error_utils.go: 错误处理工具函数
//   - error_validation.go: 验证错误和验证器
//
// 使用示例：
//
//	package main
//
//	import (
//		"fmt"
//		"github.com/yourorg/utils-pkg/errors"
//	)
//
//	func main() {
//		// 创建简单错误
//		err := errors.New("USER_NOT_FOUND", "用户未找到")
//
//		// 创建带详情和上下文的错误
//		err = errors.NewWithDetails("INVALID_INPUT", "无效的用户数据", "邮箱格式不正确")
//		err.WithContext("field", "email").WithContext("user_id", "12345")
//
//		// 使用预定义错误类型
//		err = errors.FromType(errors.NotFoundError)
//
//		// 使用构建器模式
//		err = errors.NewBuilder().
//			Code("VALIDATION_ERROR").
//			Message("验证失败").
//			Severity(errors.SeverityHigh).
//			Context("field", "email").
//			Build()
//
//		fmt.Println(err.Error())
//	}
package errors

import (
	"fmt"
	"time"
)

// Error 表示一个全面的错误，包含错误码、消息、详情、时间戳、上下文信息和错误链支持。
//
// Error结构体实现了标准error接口，并为应用程序中的结构化错误处理提供了额外功能。
type Error struct {
	Code      string                 `json:"code"`      // 错误码 - 用于程序化处理错误
	Message   string                 `json:"message"`   // 错误消息 - 人类可读的错误描述
	Details   string                 `json:"details"`   // 详细错误信息 - 额外的错误详情
	Timestamp time.Time              `json:"timestamp"` // 错误发生时间 - 用于日志和调试
	Context   map[string]interface{} `json:"context"`   // 上下文信息 - 相关的元数据
	Original  error                  `json:"original"`  // 原始错误 - 支持错误链
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

// WithMessage 设置错误消息
func (e *Error) WithMessage(message string) *Error {
	e.Message = message
	return e
}

// WithDetails 设置错误详情
func (e *Error) WithDetails(details string) *Error {
	e.Details = details
	return e
}

// WithCode 设置错误码
func (e *Error) WithCode(code string) *Error {
	e.Code = code
	return e
}

// WithOriginal 设置原始错误
func (e *Error) WithOriginal(err error) *Error {
	e.Original = err
	return e
}
