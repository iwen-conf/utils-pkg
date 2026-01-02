package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestNewRich(t *testing.T) {
	e := NewRich(404001, "资源不存在")

	// 验证 Status 嵌入
	if e.Code != 404001 {
		t.Errorf("expected Code=%d, got %d", 404001, e.Code)
	}
	if e.Msg != "资源不存在" {
		t.Errorf("expected Msg=%s, got %s", "资源不存在", e.Msg)
	}

	// 验证 Error() 接口
	errStr := e.Error()
	if !strings.Contains(errStr, "404001") {
		t.Errorf("Error() should contain code, got: %s", errStr)
	}

	// 验证无 cause
	if e.Unwrap() != nil {
		t.Error("NewRich should not have cause")
	}

	// 验证有堆栈
	if e.stack == nil {
		t.Error("NewRich should capture stack")
	}
}

func TestWrapRich(t *testing.T) {
	originalErr := errors.New("connection refused")
	e := WrapRich(originalErr, 500001, "数据库连接失败: %s", "timeout")

	// 验证 Status
	if e.Code != 500001 {
		t.Errorf("expected Code=%d, got %d", 500001, e.Code)
	}
	if e.Msg != "数据库连接失败: timeout" {
		t.Errorf("expected formatted Msg, got %s", e.Msg)
	}

	// 验证 Unwrap 返回根因
	if e.Unwrap() != originalErr {
		t.Error("Unwrap() should return original error")
	}

	// 验证 errors.Is 兼容性
	if !errors.Is(e, originalErr) {
		t.Error("errors.Is should work with RichError")
	}
}

func TestWrapRichNil(t *testing.T) {
	e := WrapRich(nil, 500001, "不应创建")
	if e != nil {
		t.Error("WrapRich(nil) should return nil")
	}
}

func TestFromRichError(t *testing.T) {
	// 测试转换 RichError
	original := NewRich(400001, "参数错误")
	converted := FromRichError(original)
	if converted != original {
		t.Error("FromRichError should return same RichError")
	}

	// 测试转换普通 error
	stdErr := errors.New("some error")
	converted = FromRichError(stdErr)
	if converted.Code != 500000 {
		t.Errorf("expected Code=500000, got %d", converted.Code)
	}
	if converted.Msg != RichMsgInternal {
		t.Errorf("expected default msg, got %s", converted.Msg)
	}
	if converted.Unwrap() != stdErr {
		t.Error("cause should be original error")
	}

	// 测试 nil
	if FromRichError(nil) != nil {
		t.Error("FromRichError(nil) should return nil")
	}
}

func TestFormatVerb(t *testing.T) {
	originalErr := errors.New("pg: no rows")
	e := WrapRich(originalErr, 404001, "用户不存在")

	// %s 只打印 Msg
	sStr := fmt.Sprintf("%s", e)
	if sStr != "用户不存在" {
		t.Errorf("%%s should print Msg only, got: %s", sStr)
	}

	// %v 只打印 Msg
	vStr := fmt.Sprintf("%v", e)
	if vStr != "用户不存在" {
		t.Errorf("%%v should print Msg only, got: %s", vStr)
	}

	// %+v 打印详细信息
	detailedStr := fmt.Sprintf("%+v", e)
	if !strings.Contains(detailedStr, "Code: 404001") {
		t.Errorf("%%+v should contain Code, got: %s", detailedStr)
	}
	if !strings.Contains(detailedStr, "Msg: 用户不存在") {
		t.Errorf("%%+v should contain Msg, got: %s", detailedStr)
	}
	if !strings.Contains(detailedStr, "Cause: pg: no rows") {
		t.Errorf("%%+v should contain Cause, got: %s", detailedStr)
	}
	if !strings.Contains(detailedStr, "Stack:") {
		t.Errorf("%%+v should contain Stack, got: %s", detailedStr)
	}
}

func TestStatusGetters(t *testing.T) {
	e := NewRich(200001, "操作成功")

	status := e.GetStatus()
	if status.Code != 200001 {
		t.Errorf("expected Code=200001, got %d", status.Code)
	}
	if status.Msg != "操作成功" {
		t.Errorf("expected Msg=操作成功, got %s", status.Msg)
	}
}

func TestRichErrorCode(t *testing.T) {
	e := NewRich(400001, "参数错误")

	code := RichErrorCode(e, 999)
	if code != 400001 {
		t.Errorf("expected 400001, got %d", code)
	}

	stdErr := errors.New("std error")
	code = RichErrorCode(stdErr, 999)
	if code != 999 {
		t.Errorf("expected default 999, got %d", code)
	}

	code = RichErrorCode(nil, 999)
	if code != 0 {
		t.Errorf("expected 0 for nil, got %d", code)
	}
}

func TestIsRichErrorCode(t *testing.T) {
	e := NewRich(400001, "参数错误")

	if !IsRichErrorCode(e, 400001) {
		t.Error("should match code 400001")
	}
	if IsRichErrorCode(e, 500001) {
		t.Error("should not match code 500001")
	}
	if IsRichErrorCode(nil, 400001) {
		t.Error("should return false for nil")
	}
	if IsRichErrorCode(errors.New("std"), 400001) {
		t.Error("should return false for non-RichError")
	}
}

// ==================== 新功能测试 ====================

func TestNewRichNoStack(t *testing.T) {
	e := NewRichNoStack(400001, "无堆栈错误")

	if e.Code != 400001 {
		t.Errorf("expected Code=400001, got %d", e.Code)
	}
	if e.stack != nil {
		t.Error("NewRichNoStack should not have stack")
	}
	if e.Stack() != "" {
		t.Error("Stack() should return empty string")
	}
}

func TestWrapRichNoStack(t *testing.T) {
	originalErr := errors.New("original")
	e := WrapRichNoStack(originalErr, 500001, "包装错误")

	if e.Code != 500001 {
		t.Errorf("expected Code=500001, got %d", e.Code)
	}
	if e.stack != nil {
		t.Error("WrapRichNoStack should not have stack")
	}
	if e.Unwrap() != originalErr {
		t.Error("Unwrap should return original error")
	}
}

func TestMarshalJSON(t *testing.T) {
	e := WrapRich(errors.New("db error"), 500001, "数据库错误")

	data, err := e.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"code":500001`) {
		t.Errorf("JSON should contain code, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"msg":"数据库错误"`) {
		t.Errorf("JSON should contain msg, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"cause":"db error"`) {
		t.Errorf("JSON should contain cause, got: %s", jsonStr)
	}
}

func TestNilSafe(t *testing.T) {
	var e *RichError = nil

	// 所有方法都应该 nil 安全
	if e.Error() != "" {
		t.Error("nil.Error() should return empty string")
	}
	if e.Unwrap() != nil {
		t.Error("nil.Unwrap() should return nil")
	}
	if e.Cause() != nil {
		t.Error("nil.Cause() should return nil")
	}
	if e.Stack() != "" {
		t.Error("nil.Stack() should return empty string")
	}
	if e.HTTPStatus() != 200 {
		t.Errorf("nil.HTTPStatus() should return 200, got %d", e.HTTPStatus())
	}

	status := e.GetStatus()
	if status.Code != RichCodeInternal {
		t.Errorf("nil.GetStatus().Code should be %d, got %d", RichCodeInternal, status.Code)
	}

	// Format nil
	s := fmt.Sprintf("%v", e)
	if s != "<nil>" {
		t.Errorf("nil Format should be <nil>, got: %s", s)
	}

	// MarshalJSON nil
	data, err := e.MarshalJSON()
	if err != nil {
		t.Fatalf("nil.MarshalJSON() failed: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("nil.MarshalJSON() should be null, got: %s", string(data))
	}
}
