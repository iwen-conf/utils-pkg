package url

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewURLBuilder(t *testing.T) {
	baseURL := "https://example.com"
	secretKey := "test-secret"
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
	builder := NewURLBuilder("https://example.com", "secret")
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
	builder := NewURLBuilder("https://example.com", "secret")
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
	builder := NewURLBuilder("https://example.com", "secret")
	fragment := "section1"
	builder.SetFragment(fragment)

	if builder.fragment != fragment {
		t.Errorf("Expected fragment %s, got %s", fragment, builder.fragment)
	}
}

func TestURLBuilder_Build(t *testing.T) {
	builder := NewURLBuilder("https://example.com", "secret")
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
	builder := NewURLBuilder("https://example.com", "secret")
	timestamp := time.Now().Unix()
	builder.SetTimestamp(timestamp)
	builder.AddParam("key1", "value1")

	url, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 测试有效签名
	valid, err := ValidateSignature(url, "secret", 3600)
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}
	if !valid {
		t.Error("Expected signature to be valid")
	}

	// 测试无效密钥
	valid, err = ValidateSignature(url, "wrong-secret", 3600)
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}
	if valid {
		t.Error("Expected signature to be invalid with wrong secret")
	}

	// 测试过期 URL
	valid, err = ValidateSignature(url, "secret", -1)
	if err == nil {
		t.Error("Expected error for expired URL")
	}
	if valid {
		t.Error("Expected signature to be invalid for expired URL")
	}
}

func TestParseURL(t *testing.T) {
	testURL := "https://example.com/path?key1=value1&key2=value2#fragment"
	result, err := ParseURL(testURL)
	if err != nil {
		t.Fatalf("ParseURL failed: %v", err)
	}

	expected := map[string]interface{}{
		"scheme":   "https",
		"host":     "example.com",
		"path":     "/path",
		"fragment": "fragment",
		"query": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestSerializeParams(t *testing.T) {
	params := map[string]interface{}{
		"string":     "value",
		"stringSlice": []string{"value1", "value2"},
		"object":     map[string]string{"key": "value"},
	}

	result := SerializeParams(params)

	// 验证字符串参数
	if !contains(result, "string=value") {
		t.Error("Missing string parameter")
	}

	// 验证字符串切片参数
	if !contains(result, "stringSlice=value1") || !contains(result, "stringSlice=value2") {
		t.Error("Missing string slice parameters")
	}

	// 验证对象参数（JSON 序列化）
	objectJSON, _ := json.Marshal(map[string]string{"key": "value"})
	expectedObject := "object=" + url.QueryEscape(string(objectJSON))
	if !contains(result, expectedObject) {
		t.Error("Missing or incorrect object parameter")
	}
}

func TestDeserializeParams(t *testing.T) {
	queryString := "key1=value1&key2=value2&array=item1&array=item2"
	result := DeserializeParams(queryString)

	// 测试单值参数
	if result["key1"] != "value1" || result["key2"] != "value2" {
		t.Error("Single value parameters not deserialized correctly")
	}

	// 测试多值参数
	array, ok := result["array"].([]string)
	if !ok {
		t.Error("Array parameter not deserialized to string slice")
	} else if len(array) != 2 || array[0] != "item1" || array[1] != "item2" {
		t.Error("Array parameter values not deserialized correctly")
	}

	// 测试空查询字符串
	emptyResult := DeserializeParams("")
	if len(emptyResult) != 0 {
		t.Error("Empty query string should return empty map")
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}