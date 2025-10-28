// Package pagination 提供不透明游标分页的通用请求体、响应体与编解码器。
// 支持 Base64+JSON 编码与 HMAC-SHA256 签名防篡改。
package pagination

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// 哨兵错误，便于使用 errors.Is 判断错误类型
var (
	// ErrEmptyCursor 游标为空
	ErrEmptyCursor = errors.New("pagination: cursor is empty")
	// ErrInvalidCursorFormat 游标格式无效
	ErrInvalidCursorFormat = errors.New("pagination: invalid cursor format")
	// ErrInvalidSignature 游标签名校验失败
	ErrInvalidSignature = errors.New("pagination: invalid cursor signature")
	// ErrEmptyHMACKey HMAC 密钥为空
	ErrEmptyHMACKey = errors.New("pagination: HMAC key cannot be empty")
	// ErrHMACKeyTooShort HMAC 密钥长度不足（建议 >= 32 字节）
	ErrHMACKeyTooShort = errors.New("pagination: HMAC key too short, recommend at least 32 bytes")
)

// CursorRequest 表示基于游标的分页请求参数。
// - Cursor: 上一页返回的游标（不透明字符串）
// - Limit: 本页希望返回的最大记录数
type CursorRequest struct {
	Cursor string `json:"cursor,omitempty" form:"cursor"`
	Limit  int    `json:"limit,omitempty" form:"limit"`
}

// CursorResponse 为基于游标的分页响应基础体，业务层自行返回数据列表。
// 仅包含翻页必要元信息。
type CursorResponse struct {
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// 分页限制常量
const (
	// DefaultLimit 分页查询的默认条数
	DefaultLimit = 20
	// MaxLimit 单次查询允许的最大条数，防止过载
	MaxLimit = 100
	// MinLimit 最小有效分页条数
	MinLimit = 1
	// RecommendedHMACKeySize HMAC 密钥推荐最小长度（字节）
	RecommendedHMACKeySize = 32
)

// Normalize 对请求的 Limit 进行归一化（默认值与上限钳制）
func (r *CursorRequest) Normalize() {
	if r.Limit < MinLimit {
		r.Limit = DefaultLimit
	}
	if r.Limit > MaxLimit {
		r.Limit = MaxLimit
	}
}

// CursorCodec 游标编解码器接口，Encode/Decode 用于生成与解析不透明游标。
type CursorCodec interface {
	Encode(v any) (string, error)
	Decode(s string, v any) error
}

// Base64JSONCodec 使用 JSON + Base64(URL) 实现游标编码。
type Base64JSONCodec struct{}

func (Base64JSONCodec) Encode(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (Base64JSONCodec) Decode(s string, v any) error {
	if s == "" {
		return ErrEmptyCursor
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCursorFormat, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCursorFormat, err)
	}
	return nil
}

// HMACCodec 在内部编解码基础上增加 HMAC-SHA256 签名，防止游标被篡改。
// 格式：v1.{payload}.{sig}
// 请使用 NewHMACCodec 构造，不要直接创建零值。
type HMACCodec struct {
	key   []byte        // 私有字段，防止外部修改
	inner CursorCodec
}

// NewHMACCodec 创建带 HMAC-SHA256 签名的游标编解码器。
// key 为签名密钥，建议长度 >= 32 字节。若 key 长度不足会返回警告错误（但仍可使用）。
func NewHMACCodec(key []byte) (*HMACCodec, error) {
	if len(key) == 0 {
		return nil, ErrEmptyHMACKey
	}
	codec := &HMACCodec{
		key:   key,
		inner: Base64JSONCodec{},
	}
	if len(key) < RecommendedHMACKeySize {
		return codec, ErrHMACKeyTooShort // 返回警告但不阻止创建
	}
	return codec, nil
}

func (c *HMACCodec) Encode(v any) (string, error) {
	if c.inner == nil {
		c.inner = Base64JSONCodec{}
	}
	payload, err := c.inner.Encode(v)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, c.key)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	// 使用 strings.Builder 减少内存分配
	var sb strings.Builder
	sb.Grow(len("v1.") + len(payload) + len(".") + len(sig))
	sb.WriteString("v1.")
	sb.WriteString(payload)
	sb.WriteByte('.')
	sb.WriteString(sig)
	return sb.String(), nil
}

func (c *HMACCodec) Decode(s string, v any) error {
	if c.inner == nil {
		c.inner = Base64JSONCodec{}
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return ErrInvalidCursorFormat
	}
	payload := parts[1]
	sig := parts[2]
	mac := hmac.New(sha256.New, c.key)
	mac.Write([]byte(payload))
	expSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expSig)) {
		return ErrInvalidSignature
	}
	return c.inner.Decode(payload, v)
}
