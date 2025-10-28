package pagination

import (
	"errors"
	"strings"
	"testing"
)

type testPayload struct {
	ID int   `json:"id"`
	TS int64 `json:"ts"`
}

func TestNormalize(t *testing.T) {
	r := CursorRequest{}
	r.Normalize()
	if r.Limit != DefaultLimit {
		t.Fatalf("expected default limit %d got %d", DefaultLimit, r.Limit)
	}
	r.Limit = -5
	r.Normalize()
	if r.Limit != DefaultLimit {
		t.Fatalf("expected default limit %d got %d", DefaultLimit, r.Limit)
	}
	r.Limit = 1000
	r.Normalize()
	if r.Limit != MaxLimit {
		t.Fatalf("expected max limit %d got %d", MaxLimit, r.Limit)
	}
}

func TestBase64JSONCodec(t *testing.T) {
	c := Base64JSONCodec{}
	p := testPayload{ID: 123, TS: 456}
	s, err := c.Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	var out testPayload
	if err := c.Decode(s, &out); err != nil {
		t.Fatal(err)
	}
	if out != p {
		t.Fatalf("mismatch")
	}
}

func TestBase64JSONCodec_EmptyCursor(t *testing.T) {
	c := Base64JSONCodec{}
	var out testPayload
	err := c.Decode("", &out)
	if !errors.Is(err, ErrEmptyCursor) {
		t.Fatalf("expected ErrEmptyCursor, got %v", err)
	}
}

func TestBase64JSONCodec_InvalidBase64(t *testing.T) {
	c := Base64JSONCodec{}
	var out testPayload
	err := c.Decode("!!!invalid!!!", &out)
	if !errors.Is(err, ErrInvalidCursorFormat) {
		t.Fatalf("expected ErrInvalidCursorFormat, got %v", err)
	}
}

func TestBase64JSONCodec_InvalidJSON(t *testing.T) {
	c := Base64JSONCodec{}
	var out testPayload
	// "notjson" 的 base64
	err := c.Decode("bm90anNvbg", &out)
	if !errors.Is(err, ErrInvalidCursorFormat) {
		t.Fatalf("expected ErrInvalidCursorFormat, got %v", err)
	}
}

func TestNewHMACCodec(t *testing.T) {
	// 正常创建（>= 32 字节）
	key32 := make([]byte, 32)
	for i := range key32 {
		key32[i] = byte(i)
	}
	codec, err := NewHMACCodec(key32)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if codec == nil {
		t.Fatal("expected non-nil codec")
	}

	// 密钥过短（警告但仍可用）
	codec, err = NewHMACCodec([]byte("short"))
	if !errors.Is(err, ErrHMACKeyTooShort) {
		t.Fatalf("expected ErrHMACKeyTooShort, got %v", err)
	}
	if codec == nil {
		t.Fatal("expected non-nil codec even with warning")
	}

	// 密钥为空
	codec, err = NewHMACCodec([]byte{})
	if !errors.Is(err, ErrEmptyHMACKey) {
		t.Fatalf("expected ErrEmptyHMACKey, got %v", err)
	}
	if codec != nil {
		t.Fatal("expected nil codec for empty key")
	}

	codec, err = NewHMACCodec(nil)
	if !errors.Is(err, ErrEmptyHMACKey) {
		t.Fatalf("expected ErrEmptyHMACKey, got %v", err)
	}
	if codec != nil {
		t.Fatal("expected nil codec for nil key")
	}
}

func TestHMACCodec(t *testing.T) {
	key32 := make([]byte, 32)
	for i := range key32 {
		key32[i] = byte(i)
	}
	codec, err := NewHMACCodec(key32)
	if err != nil {
		t.Fatal(err)
	}
	p := testPayload{ID: 123, TS: 456}
	s, err := codec.Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	// 验证格式 v1.payload.sig
	if !strings.HasPrefix(s, "v1.") {
		t.Fatalf("expected v1. prefix, got %s", s)
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}

	var out testPayload
	if err := codec.Decode(s, &out); err != nil {
		t.Fatal(err)
	}
	if out != p {
		t.Fatalf("mismatch")
	}
}

func TestHMACCodec_Tampered(t *testing.T) {
	key32 := make([]byte, 32)
	for i := range key32 {
		key32[i] = byte(i)
	}
	codec, _ := NewHMACCodec(key32)
	p := testPayload{ID: 123, TS: 456}
	s, _ := codec.Encode(p)

	// 篡改签名（追加字符导致格式错误）
	tampered := s + "a"
	var out testPayload
	err := codec.Decode(tampered, &out)
	// 追加字符后仍是 3 部分，但签名不匹配
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}

	// 篡改 payload
	parts := strings.Split(s, ".")
	tampered = "v1." + parts[1] + "xxx." + parts[2]
	err = codec.Decode(tampered, &out)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}

	// 无效格式
	err = codec.Decode("invalid", &out)
	if !errors.Is(err, ErrInvalidCursorFormat) {
		t.Fatalf("expected ErrInvalidCursorFormat, got %v", err)
	}

	// 错误版本
	err = codec.Decode("v2.payload.sig", &out)
	if !errors.Is(err, ErrInvalidCursorFormat) {
		t.Fatalf("expected ErrInvalidCursorFormat, got %v", err)
	}
}

func TestCursorResponseFields(t *testing.T) {
	resp := CursorResponse{NextCursor: "x", PrevCursor: "y", HasMore: true}
	if !resp.HasMore || resp.NextCursor != "x" || resp.PrevCursor != "y" {
		t.Fatalf("unexpected response fields")
	}
}
