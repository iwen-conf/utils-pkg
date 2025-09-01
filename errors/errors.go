package errors

import (
	"fmt"
	"time"
)

// Error 通用错误结构体
type Error struct {
	Code      string                 `json:"code"`      // 错误码
	Message   string                 `json:"message"`   // 错误消息
	Details   string                 `json:"details"`   // 详细错误信息
	Timestamp time.Time              `json:"timestamp"` // 错误发生时间
	Context   map[string]interface{} `json:"context"`   // 上下文信息
	Original  error                  `json:"original"`  // 原始错误
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
