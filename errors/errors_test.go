package errors

import (
	"fmt"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	err := New("USER001", "用户不存在")

	if err.Code != "USER001" {
		t.Errorf("期望错误码 USER001，实际得到 %s", err.Code)
	}

	if err.Message != "用户不存在" {
		t.Errorf("期望错误消息 '用户不存在'，实际得到 %s", err.Message)
	}

	if err.Timestamp.IsZero() {
		t.Error("时间戳应该被设置")
	}
}

func TestNewWithDetails(t *testing.T) {
	err := NewWithDetails("DATA001", "数据验证失败", "输入参数不符合要求")

	if err.Code != "DATA001" {
		t.Errorf("期望错误码 DATA001，实际得到 %s", err.Code)
	}

	if err.Message != "数据验证失败" {
		t.Errorf("期望错误消息 '数据验证失败'，实际得到 %s", err.Message)
	}

	if err.Details != "输入参数不符合要求" {
		t.Errorf("期望详细信息 '输入参数不符合要求'，实际得到 %s", err.Details)
	}
}

func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("数据库连接失败")
	wrappedErr := Wrap(originalErr, "DB001", "数据库操作异常")

	if wrappedErr.Code != "DB001" {
		t.Errorf("期望错误码 DB001，实际得到 %s", wrappedErr.Code)
	}

	if wrappedErr.Original != originalErr {
		t.Error("原始错误应该被正确设置")
	}

	if wrappedErr.Details != originalErr.Error() {
		t.Errorf("详细信息应该包含原始错误，期望 %s，实际得到 %s", originalErr.Error(), wrappedErr.Details)
	}
}

func TestFromCode(t *testing.T) {
	err := FromCode("4000")

	if err.Code != "4000" {
		t.Errorf("期望错误码 4000，实际得到 %s", err.Code)
	}

	// 检查是否从错误码映射中获取了消息
	expectedMessage, exists := GetMessageByCode("4000")
	if exists && err.Message != expectedMessage {
		t.Errorf("期望错误消息 %s，实际得到 %s", expectedMessage, err.Message)
	}
}

func TestErrorMethods(t *testing.T) {
	err := New("USER001", "用户不存在")

	// 测试 Error() 方法
	errorString := err.Error()
	expectedString := "[USER001] 用户不存在"
	if errorString != expectedString {
		t.Errorf("期望错误字符串 %s，实际得到 %s", expectedString, errorString)
	}

	// 测试带详细信息的 Error() 方法
	err.Details = "用户ID无效"
	errorString = err.Error()
	expectedString = "[USER001] 用户不存在: 用户ID无效"
	if errorString != expectedString {
		t.Errorf("期望错误字符串 %s，实际得到 %s", expectedString, errorString)
	}
}

func TestWithContext(t *testing.T) {
	err := New("USER001", "用户不存在")

	err = err.WithContext("user_id", "user123")
	err = err.WithContext("request_id", "req_456")

	if err.Context["user_id"] != "user123" {
		t.Errorf("期望上下文 user_id 为 user123，实际得到 %v", err.Context["user_id"])
	}

	if err.Context["request_id"] != "req_456" {
		t.Errorf("期望上下文 request_id 为 req_456，实际得到 %v", err.Context["request_id"])
	}
}

func TestIsBusinessError(t *testing.T) {
	// 测试业务错误
	businessErr := New("USER001", "用户不存在")
	if !IsBusinessError(businessErr) {
		t.Error("业务错误应该被正确识别")
	}

	// 测试非业务错误
	regularErr := fmt.Errorf("系统错误")
	if IsBusinessError(regularErr) {
		t.Error("系统错误不应该被识别为业务错误")
	}

	// 测试 nil 错误
	if IsBusinessError(nil) {
		t.Error("nil 错误不应该被识别为业务错误")
	}
}

func TestGetBusinessError(t *testing.T) {
	// 测试业务错误
	businessErr := New("USER001", "用户不存在")
	retrievedErr := GetBusinessError(businessErr)

	if retrievedErr != businessErr {
		t.Error("应该返回相同的业务错误")
	}

	// 测试非业务错误
	regularErr := fmt.Errorf("系统错误")
	retrievedErr = GetBusinessError(regularErr)

	if retrievedErr != nil {
		t.Error("系统错误应该返回 nil")
	}
}

func TestGetErrorCode(t *testing.T) {
	err := New("USER001", "用户不存在")

	code := GetErrorCode(err)
	if code != "USER001" {
		t.Errorf("期望错误码 USER001，实际得到 %s", code)
	}

	// 测试非业务错误
	regularErr := fmt.Errorf("系统错误")
	code = GetErrorCode(regularErr)
	if code != "" {
		t.Errorf("系统错误应该返回空字符串，实际得到 %s", code)
	}
}

func TestGetErrorMessage(t *testing.T) {
	err := New("USER001", "用户不存在")

	message := GetErrorMessage(err)
	if message != "用户不存在" {
		t.Errorf("期望错误消息 '用户不存在'，实际得到 %s", message)
	}

	// 测试非业务错误
	regularErr := fmt.Errorf("系统错误")
	message = GetErrorMessage(regularErr)
	if message != regularErr.Error() {
		t.Errorf("系统错误应该返回原始错误消息，期望 %s，实际得到 %s", regularErr.Error(), message)
	}
}

func TestGetErrorDetails(t *testing.T) {
	err := NewWithDetails("USER001", "用户不存在", "用户ID无效")

	details := GetErrorDetails(err)
	if details != "用户ID无效" {
		t.Errorf("期望错误详情 '用户ID无效'，实际得到 %s", details)
	}

	// 测试没有详细信息的错误
	err = New("USER001", "用户不存在")
	details = GetErrorDetails(err)
	if details != "" {
		t.Errorf("没有详细信息的错误应该返回空字符串，实际得到 %s", details)
	}
}

func TestErrorBuilder(t *testing.T) {
	err := NewBuilder().
		Code("USER001").
		Message("用户不存在").
		Details("用户ID无效").
		Context("user_id", "user123").
		Build()

	if err.Code != "USER001" {
		t.Errorf("期望错误码 USER001，实际得到 %s", err.Code)
	}

	if err.Message != "用户不存在" {
		t.Errorf("期望错误消息 '用户不存在'，实际得到 %s", err.Message)
	}

	if err.Details != "用户ID无效" {
		t.Errorf("期望错误详情 '用户ID无效'，实际得到 %s", err.Details)
	}

	if err.Context["user_id"] != "user123" {
		t.Errorf("期望上下文 user_id 为 user123，实际得到 %v", err.Context["user_id"])
	}
}

func TestErrorClassification(t *testing.T) {
	// 测试系统错误
	if !IsSystemError("500001") {
		t.Error("500001 应该被识别为系统错误")
	}

	// 测试客户端错误
	if !IsClientError("400001") {
		t.Error("400001 应该被识别为客户端错误")
	}

	// 测试业务错误
	if !IsBusinessErrorCode("600001") {
		t.Error("600001 应该被识别为业务错误")
	}

	// 测试错误分类
	if GetCategoryByCode("500001") != "server" {
		t.Error("500001 应该被分类为 server")
	}

	if GetCategoryByCode("400001") != "client" {
		t.Error("400001 应该被分类为 client")
	}

	if GetCategoryByCode("600001") != "business" {
		t.Error("600001 应该被分类为 business")
	}
}

func TestErrorRecovery(t *testing.T) {
	// 测试可重试错误
	retryableErr := New("5000", "内部服务器错误")
	if !IsRetryableErrorCode(retryableErr.Code) {
		t.Error("内部服务器错误应该被识别为可重试错误")
	}

	// 测试永久性错误
	permanentErr := New("4001", "未授权")
	if !IsPermanentErrorCode(permanentErr.Code) {
		t.Error("未授权应该被识别为永久性错误")
	}
}

func TestErrorChaining(t *testing.T) {
	originalErr := fmt.Errorf("数据库连接失败")
	wrappedErr := Wrap(originalErr, "DB001", "数据库操作异常")

	// 测试 Unwrap 方法
	unwrappedErr := wrappedErr.Unwrap()
	if unwrappedErr != originalErr {
		t.Error("Unwrap 应该返回原始错误")
	}
}

func TestErrorContext(t *testing.T) {
	err := New("USER001", "用户不存在")

	// 添加多个上下文信息
	err = err.WithContext("user_id", "user123").
		WithContext("request_id", "req_456").
		WithContext("timestamp", 1640995200)

	// 验证上下文信息
	expectedContext := map[string]interface{}{
		"user_id":    "user123",
		"request_id": "req_456",
		"timestamp":  1640995200,
	}

	for key, expectedValue := range expectedContext {
		if err.Context[key] != expectedValue {
			t.Errorf("期望上下文 %s 为 %v，实际得到 %v", key, expectedValue, err.Context[key])
		}
	}
}

func TestErrorTimestamp(t *testing.T) {
	before := time.Now()
	err := New("USER001", "用户不存在")
	after := time.Now()

	if err.Timestamp.Before(before) || err.Timestamp.After(after) {
		t.Errorf("时间戳应该在 %v 和 %v 之间，实际得到 %v", before, after, err.Timestamp)
	}
}

// 基准测试
func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New("USER001", "用户不存在")
	}
}

func BenchmarkNewWithDetails(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewWithDetails("USER001", "用户不存在", "用户ID无效")
	}
}

func BenchmarkWrap(b *testing.B) {
	originalErr := fmt.Errorf("数据库连接失败")
	for i := 0; i < b.N; i++ {
		Wrap(originalErr, "DB001", "数据库操作异常")
	}
}

func BenchmarkErrorBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewBuilder().
			Code("USER001").
			Message("用户不存在").
			Details("用户ID无效").
			Context("user_id", "user123").
			Build()
	}
}
