package signedurl

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	ErrInvalidBaseURL              = errors.New("invalid base URL")
	ErrMissingTimestamp            = errors.New("missing timestamp parameter")
	ErrMissingSignature            = errors.New("missing signature parameter")
	ErrInvalidTimestamp            = errors.New("invalid timestamp format")
	ErrExpiredURL                  = errors.New("URL has expired")
	ErrFutureTimestamp             = errors.New("timestamp is in the future")
	ErrInvalidSignature            = errors.New("invalid signature")
	ErrEmptySecretKey              = errors.New("secret key cannot be empty")
	ErrSecretKeyTooShort           = errors.New("secret key must be at least 32 bytes")
	ErrInvalidMethod               = errors.New("invalid HTTP method")
	ErrMissingSignatureVersion     = errors.New("missing signature version")
	ErrUnsupportedSignatureVersion = errors.New("unsupported signature version")
	ErrMissingExpiration           = errors.New("missing expiration")
	ErrInvalidExpiration           = errors.New("invalid expiration")
	ErrAmbiguousSignatureParams    = errors.New("ambiguous reserved signature parameters")
	ErrReservedQueryParamConflict  = errors.New("reserved signature parameter conflict")
)

const (
	allowedTimeDrift   = 300
	minSecretKeyBytes  = 32
	signatureVersionV2 = "2"
	reservedSigVersion = "_sigv"
	reservedTimestamp  = "_ts"
	reservedExpiration = "_exp"
	reservedSignature  = "_sign"
)

var reservedQueryKeys = []string{reservedSigVersion, reservedTimestamp, reservedExpiration, reservedSignature}

// URLBuilder URL 签名构建器。
type URLBuilder struct {
	baseURL      string
	params       url.Values
	fragment     string
	sortParams   bool
	secretKey    string
	timestamp    int64
	timestampSet bool
	expiration   int64
	method       string
	nowFunc      func() time.Time
}

// NewURLBuilder 创建新的 URL 构建器，默认方法为 GET，默认有效期 3600 秒。
// 未显式调用 SetTimestamp 时，每次 Build 会使用当前实时时间戳。
func NewURLBuilder(baseURL string, secretKey string) *URLBuilder {
	return &URLBuilder{
		baseURL:    baseURL,
		params:     make(url.Values),
		sortParams: true,
		secretKey:  secretKey,
		expiration: 3600,
		method:     "GET",
		nowFunc:    time.Now,
	}
}

func (b *URLBuilder) now() time.Time {
	if b != nil && b.nowFunc != nil {
		return b.nowFunc()
	}
	return time.Now()
}

func (b *URLBuilder) effectiveTimestamp() int64 {
	if b.timestampSet {
		return b.timestamp
	}
	return b.now().Unix()
}

// SetMethod 设置签名绑定的 HTTP 方法。
func (b *URLBuilder) SetMethod(method string) *URLBuilder {
	b.method = method
	return b
}

// Validate 验证构建器的基本参数。
func (b *URLBuilder) Validate() error {
	if b == nil {
		return ErrInvalidBaseURL
	}
	if _, err := normalizeSecret(b.secretKey); err != nil {
		return err
	}
	if _, err := normalizeMethod(b.method); err != nil {
		return err
	}
	parsed, err := parseAbsoluteHTTPURL(b.baseURL)
	if err != nil {
		return err
	}
	if hasReservedKeys(parsed.Query()) || hasReservedKeys(b.params) {
		return ErrReservedQueryParamConflict
	}
	if b.effectiveTimestamp() <= 0 {
		return ErrInvalidTimestamp
	}
	if b.expiration <= 0 {
		return ErrInvalidExpiration
	}
	return nil
}

// AddParam 添加查询参数。
func (b *URLBuilder) AddParam(key string, value string) *URLBuilder {
	b.params.Add(key, value)
	return b
}

// AddParams 批量添加查询参数。
func (b *URLBuilder) AddParams(params map[string]string) *URLBuilder {
	for k, v := range params {
		b.AddParam(k, v)
	}
	return b
}

// SetFragment 设置 URL 片段（不参与服务端 HTTP 请求签名）。
func (b *URLBuilder) SetFragment(fragment string) *URLBuilder {
	b.fragment = fragment
	return b
}

// SetSortParams 设置是否对参数按键排序。
func (b *URLBuilder) SetSortParams(sort bool) *URLBuilder {
	b.sortParams = sort
	return b
}

// SetTimestamp 显式固定签名时间戳（Unix 秒）。
func (b *URLBuilder) SetTimestamp(timestamp int64) *URLBuilder {
	b.timestamp = timestamp
	b.timestampSet = true
	return b
}

// SetExpiration 设置 URL 过期时间（秒）。
func (b *URLBuilder) SetExpiration(seconds int64) *URLBuilder {
	b.expiration = seconds
	return b
}

// Build 构建完整的 v2 签名 URL。
func (b *URLBuilder) Build() (string, error) {
	if err := b.Validate(); err != nil {
		return "", err
	}

	method, err := normalizeMethod(b.method)
	if err != nil {
		return "", err
	}
	baseURL, err := parseAbsoluteHTTPURL(b.baseURL)
	if err != nil {
		return "", err
	}

	query := baseURL.Query()
	if b.sortParams {
		keys := make([]string, 0, len(b.params))
		for k := range b.params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			for _, v := range b.params[k] {
				query.Add(k, v)
			}
		}
	} else {
		for k, values := range b.params {
			for _, v := range values {
				query.Add(k, v)
			}
		}
	}

	ts := b.effectiveTimestamp()
	query.Del(reservedSignature)
	query.Set(reservedSigVersion, signatureVersionV2)
	query.Set(reservedTimestamp, strconv.FormatInt(ts, 10))
	query.Set(reservedExpiration, strconv.FormatInt(b.expiration, 10))
	baseURL.RawQuery = query.Encode()

	canonical, err := canonicalRequestV2(baseURL, method)
	if err != nil {
		return "", err
	}
	signature := signCanonical(b.secretKey, canonical)
	query.Set(reservedSignature, signature)
	baseURL.RawQuery = query.Encode()
	baseURL.Fragment = b.fragment
	return baseURL.String(), nil
}

func canonicalRequestV2(parsed *url.URL, method string) (string, error) {
	method, err := normalizeMethod(method)
	if err != nil {
		return "", err
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	query := parsed.Query()
	query.Del(reservedSignature)
	return strings.Join([]string{
		signatureVersionV2,
		method,
		strings.ToLower(parsed.Scheme),
		strings.ToLower(parsed.Host),
		path,
		query.Encode(),
	}, "\n"), nil
}

func signCanonical(secretKey, canonical string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(canonical))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// ValidateSignature 按 GET 验证 v2 签名 URL。
func ValidateSignature(rawURL, secretKey string, maxAgeSeconds int64) (bool, error) {
	return ValidateSignatureWithMethod(rawURL, "GET", secretKey, maxAgeSeconds)
}

// ValidateSignatureWithMethod 按指定 HTTP 方法验证 v2 签名 URL。
func ValidateSignatureWithMethod(rawURL, method, secretKey string, maxAgeSeconds int64) (bool, error) {
	return validateSignatureAt(rawURL, method, secretKey, maxAgeSeconds, time.Now().Unix())
}

func validateSignatureAt(rawURL, method, secretKey string, maxAgeSeconds int64, now int64) (bool, error) {
	if _, err := normalizeSecret(secretKey); err != nil {
		return false, err
	}
	method, err := normalizeMethod(method)
	if err != nil {
		return false, err
	}
	parsedURL, err := parseAbsoluteHTTPURL(rawURL)
	if err != nil {
		return false, err
	}

	query := parsedURL.Query()

	sigv, err := reservedValue(query, reservedSigVersion, ErrMissingSignatureVersion)
	if err != nil {
		return false, err
	}
	if sigv != signatureVersionV2 {
		return false, ErrUnsupportedSignatureVersion
	}
	timestamp, err := reservedValue(query, reservedTimestamp, ErrMissingTimestamp)
	if err != nil {
		return false, err
	}
	expiration, err := reservedValue(query, reservedExpiration, ErrMissingExpiration)
	if err != nil {
		return false, err
	}
	signature, err := reservedValue(query, reservedSignature, ErrMissingSignature)
	if err != nil {
		return false, err
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || ts <= 0 {
		return false, ErrInvalidTimestamp
	}
	exp, err := strconv.ParseInt(expiration, 10, 64)
	if err != nil || exp <= 0 {
		return false, ErrInvalidExpiration
	}
	if ts > now+allowedTimeDrift {
		return false, ErrFutureTimestamp
	}

	deadline, ok := addUnix(ts, exp)
	if !ok {
		return false, ErrInvalidExpiration
	}
	if maxAgeSeconds > 0 {
		maxDeadline, ok := addUnix(ts, maxAgeSeconds)
		if !ok {
			return false, ErrInvalidExpiration
		}
		if maxDeadline < deadline {
			deadline = maxDeadline
		}
	}
	if now > deadline {
		return false, ErrExpiredURL
	}

	canonical, err := canonicalRequestV2(parsedURL, method)
	if err != nil {
		return false, err
	}
	expected := signCanonical(secretKey, canonical)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) != 1 {
		return false, ErrInvalidSignature
	}
	return true, nil
}

// BatchValidateSignatures 批量验证 GET URL 签名。
func BatchValidateSignatures(urls []string, secretKey string, maxAgeSeconds int64) map[string]error {
	results := make(map[string]error, len(urls))
	for _, rawURL := range urls {
		_, err := ValidateSignature(rawURL, secretKey, maxAgeSeconds)
		results[rawURL] = err
	}
	return results
}

// CreateSignedURL 快速创建带签名的 GET URL。
func CreateSignedURL(baseURL, secretKey string, params map[string]string, expireSeconds int64) (string, error) {
	builder := NewURLBuilder(baseURL, secretKey)
	builder.AddParams(params)
	if expireSeconds > 0 {
		builder.SetExpiration(expireSeconds)
	} else if expireSeconds < 0 {
		return "", ErrInvalidExpiration
	}
	return builder.Build()
}

func normalizeSecret(secretKey string) (string, error) {
	if secretKey == "" {
		return "", ErrEmptySecretKey
	}
	if len([]byte(secretKey)) < minSecretKeyBytes {
		return "", ErrSecretKeyTooShort
	}
	return secretKey, nil
}

func normalizeMethod(method string) (string, error) {
	trimmed := strings.TrimSpace(method)
	if trimmed == "" {
		return "", ErrInvalidMethod
	}
	for _, r := range method {
		if r == '\n' || r == '\r' || unicode.IsControl(r) {
			return "", ErrInvalidMethod
		}
	}
	for _, r := range trimmed {
		if r <= ' ' || unicode.IsSpace(r) {
			return "", ErrInvalidMethod
		}
	}
	return strings.ToUpper(trimmed), nil
}

func parseAbsoluteHTTPURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, ErrInvalidBaseURL
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBaseURL, err)
	}
	if !parsed.IsAbs() || parsed.Host == "" || parsed.User != nil || parsed.Opaque != "" {
		return nil, ErrInvalidBaseURL
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return nil, ErrInvalidBaseURL
	}
	return parsed, nil
}

func hasReservedKeys(values url.Values) bool {
	for _, key := range reservedQueryKeys {
		if _, ok := values[key]; ok {
			return true
		}
	}
	return false
}

func reservedValue(query url.Values, key string, missing error) (string, error) {
	vals := query[key]
	switch len(vals) {
	case 0:
		return "", missing
	case 1:
		return vals[0], nil
	default:
		return "", ErrAmbiguousSignatureParams
	}
}

func addUnix(base, delta int64) (int64, bool) {
	if delta > 0 && base > math.MaxInt64-delta {
		return 0, false
	}
	if delta < 0 && base < math.MinInt64-delta {
		return 0, false
	}
	return base + delta, true
}
