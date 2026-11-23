package signedurl

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const testURLSecret = "0123456789abcdef0123456789abcdef"

func TestNewURLBuilder(t *testing.T) {
	baseURL := "https://example.com"
	secretKey := testURLSecret
	builder := NewURLBuilder(baseURL, secretKey)

	if builder.baseURL != baseURL {
		t.Errorf("Expected baseURL %s, got %s", baseURL, builder.baseURL)
	}
	if builder.secretKey != secretKey {
		t.Errorf("Expected secretKey %s, got %s", secretKey, builder.secretKey)
	}
	if !builder.sortParams {
		t.Error("Expected sortParams to be true by default")
	}
}

func TestURLBuilder_AddParam(t *testing.T) {
	builder := NewURLBuilder("https://example.com", testURLSecret)
	builder.AddParam("key1", "value1")
	builder.AddParam("key1", "value2")

	if len(builder.params["key1"]) != 2 {
		t.Errorf("Expected 2 values for key1, got %d", len(builder.params["key1"]))
	}
	if builder.params["key1"][0] != "value1" || builder.params["key1"][1] != "value2" {
		t.Error("Parameter values not stored correctly")
	}
}

func TestURLBuilder_AddParams(t *testing.T) {
	builder := NewURLBuilder("https://example.com", testURLSecret)
	params := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	builder.AddParams(params)

	if len(builder.params) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(builder.params))
	}
	for k, v := range params {
		if builder.params[k][0] != v {
			t.Errorf("Expected value %s for key %s, got %s", v, k, builder.params[k][0])
		}
	}
}

func TestURLBuilder_SetFragment(t *testing.T) {
	builder := NewURLBuilder("https://example.com", testURLSecret)
	fragment := "section1"
	builder.SetFragment(fragment)

	if builder.fragment != fragment {
		t.Errorf("Expected fragment %s, got %s", fragment, builder.fragment)
	}
}

func TestURLBuilder_Build(t *testing.T) {
	builder := NewURLBuilder("https://example.com", testURLSecret)
	timestamp := time.Now().Unix()
	builder.SetTimestamp(timestamp)
	builder.AddParam("key1", "value1")
	builder.SetFragment("section1")

	url, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 验证 URL 包含必要的组件
	if url == "" {
		t.Error("Built URL is empty")
	}
	if !contains(url, "key1=value1") {
		t.Error("URL does not contain expected parameter")
	}
	if !contains(url, "_ts=") {
		t.Error("URL does not contain timestamp")
	}
	if !contains(url, "_sign=") {
		t.Error("URL does not contain signature")
	}
	if !contains(url, "#section1") {
		t.Error("URL does not contain fragment")
	}
}

func TestValidateSignature(t *testing.T) {
	builder := NewURLBuilder("https://example.com", testURLSecret)
	timestamp := time.Now().Unix()
	builder.SetTimestamp(timestamp)
	builder.AddParam("key1", "value1")

	url, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 测试有效签名
	valid, err := ValidateSignature(url, testURLSecret, 3600)
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}
	if !valid {
		t.Error("Expected signature to be valid")
	}

	// 测试无效密钥
	valid, err = ValidateSignature(url, "wrong-secret-key-32bytes-xxxxxxxx", 3600)
	if err == nil || !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("Expected invalid signature error, got: %v", err)
	}
	if valid {
		t.Error("Expected signature to be invalid with wrong secret")
	}

	// 测试过期 URL
	oldURL, err := NewURLBuilder("https://example.com", testURLSecret).
		SetTimestamp(time.Now().Unix()-7200). // 2小时前
		AddParam("key1", "value1").
		Build()

	if err != nil {
		t.Fatalf("Build expired URL failed: %v", err)
	}

	valid, err = ValidateSignature(oldURL, testURLSecret, 3600)
	if err == nil || !errors.Is(err, ErrExpiredURL) {
		t.Errorf("Expected expired URL error, got: %v", err)
	}
	if valid {
		t.Error("Expected signature to be invalid for expired URL")
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func BenchmarkURLBuilder_Build(b *testing.B) {
	builder := NewURLBuilder("https://example.com", testURLSecret)
	builder.AddParam("key1", "value1")
	builder.AddParam("key2", "value2")
	builder.SetFragment("section1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := builder.Build()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateSignature(b *testing.B) {
	builder := NewURLBuilder("https://example.com", testURLSecret)
	builder.AddParam("key1", "value1")
	url, err := builder.Build()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ValidateSignature(url, testURLSecret, 3600)
		if err != nil {
			b.Fatal(err)
		}
	}
}
