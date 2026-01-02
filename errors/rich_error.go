package errors

import (
	"encoding/json"
	"fmt"
	"io"
)

// Status 代表业务状态，可直接被 JSON 序列化
// 这是一个"纯数据"结构，Controller 层可直接嵌入到 Response 中
type Status struct {
	Code int    `json:"code"` // 业务码
	Msg  string `json:"msg"`  // 用户提示语
}

// RichError 是企业级富错误类型
// ✅ 核心设计：嵌入 Status，自然拥有 Code 和 Msg 字段
type RichError struct {
	Status        // 组合特性 (Composition)
	cause  error  // 根因 (不导出，不给前端看)
	stack  *stack // 堆栈 (不导出)
}

// Error 实现标准 error 接口
func (e *RichError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("code=%d msg=%s", e.Code, e.Msg)
}

// Unwrap 实现 Go 1.13 标准，允许 errors.Is/As 穿透
func (e *RichError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Cause 返回根因错误（用于内部调试）
func (e *RichError) Cause() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Stack 返回堆栈信息的字符串表示（用于日志输出）
func (e *RichError) Stack() string {
	if e == nil || e.stack == nil {
		return ""
	}
	return fmt.Sprintf("%+v", e.stack)
}

// Format 实现 fmt.Formatter，支持 %+v 打印详细堆栈
// 使用: logger.Errorf("%+v", err)
func (e *RichError) Format(s fmt.State, verb rune) {
	if e == nil {
		io.WriteString(s, "<nil>")
		return
	}
	switch verb {
	case 'v':
		if s.Flag('+') {
			// 详细模式：打印 Code, Msg, Cause, Stack
			fmt.Fprintf(s, "Code: %d\nMsg: %s\n", e.Code, e.Msg)
			if e.cause != nil {
				fmt.Fprintf(s, "Cause: %+v\n", e.cause)
			}
			// 打印堆栈
			if e.stack != nil {
				fmt.Fprintf(s, "Stack:%v", e.stack)
			}
			return
		}
		fallthrough
	case 's':
		// 普通字符串模式：只打印 Msg
		io.WriteString(s, e.Msg)
	case 'q':
		fmt.Fprintf(s, "%q", e.Msg)
	}
}

// GetStatus 返回 Status 结构体（便于 Controller 层使用）
func (e *RichError) GetStatus() Status {
	if e == nil {
		return Status{Code: RichCodeInternal, Msg: RichMsgInternal}
	}
	return e.Status
}

// MarshalJSON 实现 JSON 序列化，用于日志输出
// 输出格式: {"code":500001,"msg":"xxx","cause":"原始错误"}
func (e *RichError) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}

	type jsonError struct {
		Code  int    `json:"code"`
		Msg   string `json:"msg"`
		Cause string `json:"cause,omitempty"`
	}

	je := jsonError{
		Code: e.Code,
		Msg:  e.Msg,
	}
	if e.cause != nil {
		je.Cause = e.cause.Error()
	}

	return json.Marshal(je)
}
