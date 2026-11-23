package pagination

import (
	"encoding/base64"
	"encoding/json"
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

	codec, err = NewHMACCodec([]byte("short"))
	if !errors.Is(err, ErrHMACKeyTooShort) {
		t.Fatalf("expected ErrHMACKeyTooShort, got %v", err)
	}
	if codec != nil {
		t.Fatal("expected nil codec for short key")
	}

	codec, err = NewHMACCodec(make([]byte, 31))
	if !errors.Is(err, ErrHMACKeyTooShort) || codec != nil {
		t.Fatalf("31-byte key: codec=%v err=%v", codec, err)
	}

	codec, err = NewHMACCodec([]byte{1})
	if !errors.Is(err, ErrHMACKeyTooShort) || codec != nil {
		t.Fatalf("1-byte key: codec=%v err=%v", codec, err)
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

func TestHMACCodec_KeyCopyAndUninitialized(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	codec, err := NewHMACCodec(key)
	if err != nil {
		t.Fatal(err)
	}
	s, err := codec.Encode(testPayload{ID: 1})
	if err != nil {
		t.Fatal(err)
	}
	for i := range key {
		key[i] = 0xff
	}
	var out testPayload
	if err := codec.Decode(s, &out); err != nil {
		t.Fatalf("mutating constructor key must not change codec: %v", err)
	}

	var nilCodec *HMACCodec
	if _, err := nilCodec.Encode(out); !errors.Is(err, ErrUninitializedHMACCodec) {
		t.Fatalf("nil Encode: %v", err)
	}
	if err := nilCodec.Decode(s, &out); !errors.Is(err, ErrUninitializedHMACCodec) {
		t.Fatalf("nil Decode: %v", err)
	}

	zero := HMACCodec{}
	if _, err := zero.Encode(out); !errors.Is(err, ErrUninitializedHMACCodec) {
		t.Fatalf("zero Encode: %v", err)
	}
	cleared := &HMACCodec{inner: Base64JSONCodec{}}
	if _, err := cleared.Encode(out); !errors.Is(err, ErrUninitializedHMACCodec) {
		t.Fatalf("empty key Encode: %v", err)
	}

	for _, size := range []int{32, 48, 64} {
		k := make([]byte, size)
		c, err := NewHMACCodec(k)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		s, err := c.Encode(testPayload{ID: size})
		if err != nil {
			t.Fatal(err)
		}
		var got testPayload
		if err := c.Decode(s, &got); err != nil || got.ID != size {
			t.Fatalf("round trip size %d: %+v %v", size, got, err)
		}
	}
}

func TestCursorResponseFields(t *testing.T) {
	resp := CursorResponse{NextCursor: "x", PrevCursor: "y", HasMore: true}
	if !resp.HasMore || resp.NextCursor != "x" || resp.PrevCursor != "y" {
		t.Fatalf("unexpected response fields")
	}
}

func TestGenericEncodeDecodeCursor(t *testing.T) {
	p := testPayload{ID: 77, TS: 1700000123}

	// nil codec
	if _, err := EncodeCursor(nil, p); !errors.Is(err, ErrNilCodec) {
		t.Fatalf("EncodeCursor(nil): %v", err)
	}
	if _, err := DecodeCursor[testPayload](nil, "abc"); !errors.Is(err, ErrNilCodec) {
		t.Fatalf("DecodeCursor(nil): %v", err)
	}

	// Base64JSONCodec
	base := Base64JSONCodec{}
	token, err := EncodeCursor(base, p)
	if err != nil {
		t.Fatalf("EncodeCursor(base): %v", err)
	}
	got, err := DecodeCursor[testPayload](base, token)
	if err != nil || got != p {
		t.Fatalf("DecodeCursor(base) = (%+v, %v), want %+v", got, err, p)
	}

	// HMACCodec
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	hmacCodec, err := NewHMACCodec(key)
	if err != nil {
		t.Fatalf("NewHMACCodec: %v", err)
	}
	signed, err := EncodeCursor(hmacCodec, p)
	if err != nil {
		t.Fatalf("EncodeCursor(hmac): %v", err)
	}
	got, err = DecodeCursor[testPayload](hmacCodec, signed)
	if err != nil || got != p {
		t.Fatalf("DecodeCursor(hmac) = (%+v, %v), want %+v", got, err, p)
	}
	if _, err := DecodeCursor[testPayload](hmacCodec, "v1.onlytwo"); !errors.Is(err, ErrInvalidCursorFormat) {
		t.Fatalf("expected ErrInvalidCursorFormat for 2 parts, got %v", err)
	}
	if _, err := DecodeCursor[testPayload](hmacCodec, "v1.a.b.c"); !errors.Is(err, ErrInvalidCursorFormat) {
		t.Fatalf("expected ErrInvalidCursorFormat for 4 parts, got %v", err)
	}
}

func TestBase64JSONCodec_ErrorChainPreserved(t *testing.T) {
	c := Base64JSONCodec{}
	var out testPayload

	err := c.Decode("!!!invalid!!!", &out)
	if !errors.Is(err, ErrInvalidCursorFormat) {
		t.Fatalf("expected ErrInvalidCursorFormat, got %v", err)
	}
	var b64Err base64.CorruptInputError
	if !errors.As(err, &b64Err) {
		t.Fatalf("expected wrapped base64.CorruptInputError, got %v", err)
	}

	err = c.Decode("bm90anNvbg", &out)
	if !errors.Is(err, ErrInvalidCursorFormat) {
		t.Fatalf("expected ErrInvalidCursorFormat, got %v", err)
	}
	var jsonErr *json.SyntaxError
	if !errors.As(err, &jsonErr) {
		t.Fatalf("expected wrapped *json.SyntaxError, got %v", err)
	}
}
