package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecretKey = "this-is-a-very-secure-jwt-secret-key-32bytes!"

func TestStandardClaims_MarshalJSON_StandardFields(t *testing.T) {
	now := time.Now()
	c := StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "issuer-1",
			Subject:   "sub-1",
			Audience:  jwt.ClaimStrings{"aud-1"},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		TokenType: AccessToken,
		SessionID: "sid-1",
		TokenID:   "jti-1",
		CustomClaims: map[string]interface{}{
			"k": "v",
		},
	}

	b, err := c.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if m["iss"] != "issuer-1" || m["sub"] != "sub-1" {
		t.Fatalf("unexpected iss/sub: %+v", m)
	}
	aud, ok := m["aud"].([]interface{})
	if !ok || len(aud) != 1 || aud[0] != "aud-1" {
		t.Fatalf("Expected aud=[aud-1], got %v", m["aud"])
	}
	for _, field := range []string{"exp", "nbf", "iat"} {
		if _, ok := m[field]; !ok {
			t.Fatalf("Expected %s present", field)
		}
	}
	if m["jti"] != "jti-1" || m["type"] != string(AccessToken) || m["sid"] != "sid-1" || m["k"] != "v" {
		t.Fatalf("unexpected claims payload: %+v", m)
	}
}

func TestStandardClaims_MarshalJSON_JTIFromRegisteredID(t *testing.T) {
	c := StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{ID: "id-1"},
		CustomClaims:     map[string]interface{}{},
	}
	b, err := c.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if m["jti"] != "id-1" {
		t.Fatalf("Expected jti=id-1, got %v", m["jti"])
	}
}

func TestStandardClaims_UnmarshalJSON_InvalidJSON(t *testing.T) {
	var c StandardClaims
	if err := c.UnmarshalJSON([]byte("{")); err == nil {
		t.Fatalf("Expected UnmarshalJSON error")
	}
}

func TestNewTokenManager_KeyHasUpper(t *testing.T) {
	tm, err := NewTokenManager("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	if err != nil {
		t.Fatalf("32-byte uniform secret should be accepted: %v", err)
	}
	tm.Shutdown()
}

func TestMustNewTokenManager_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("Expected panic")
		}
	}()
	_ = MustNewTokenManager("short")
}

func TestTokenManager_FormatValidationAndCacheInitialState(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	if tm.GetCacheSize() != 0 {
		t.Fatalf("Expected cache size 0")
	}
	if isTokenFormatValid("short") || isTokenFormatValid("a..bbbbbbbbb") || isTokenFormatValid(".aaaa.bbbbbb") || isTokenFormatValid("aaaa.bbbbbb.") {
		t.Fatalf("Expected invalid token formats to be rejected")
	}
}

func TestTokenManager_GenerateToken_ErrorOnUnmarshalableCustomClaims(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	_, err := tm.GenerateToken("u-1", &TokenOptions{
		CustomClaims: map[string]interface{}{
			"bad": make(chan int),
		},
	})
	if err == nil {
		t.Fatalf("Expected GenerateToken error")
	}
}

func TestTokenManager_ValidateAccessToken_UnexpectedSigningMethod(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "u-1"})
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("SignedString error: %v", err)
	}
	if _, err := tm.ValidateAccessToken(context.Background(), tokenStr); err == nil {
		t.Fatalf("Expected ValidateAccessToken error")
	}
}

func TestTokenManager_ValidateAccessToken_InvalidTokenNoError(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	orig := parseWithClaims
	defer func() { parseWithClaims = orig }()

	parseWithClaims = func(tokenString string, claims jwt.Claims, keyFunc jwt.Keyfunc, options ...jwt.ParserOption) (*jwt.Token, error) {
		return &jwt.Token{Valid: false, Claims: jwt.MapClaims{}}, nil
	}

	_, err := tm.ValidateAccessToken(context.Background(), "aaaaaaaaaa.bbbbbbbbbb.cccccccccc")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestTokenManager_RefreshTokenPair_InvalidInput(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	_, err := tm.RefreshTokenPair(context.Background(), "invalid")
	if err == nil {
		t.Fatalf("Expected RefreshTokenPair error")
	}
}

func TestTokenManager_RefreshTokenPair_GenerateTokenFailureDoesNotConsumeJTI(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	ctx := context.Background()

	rawClaims := jwt.MapClaims{
		"sub":  "   ",
		"type": "refresh",
		"sid":  "s-1",
		"jti":  "id-empty-sub",
		"exp":  time.Now().Add(time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, rawClaims)
	tokenStr, err := token.SignedString(tm.secretKey)
	if err != nil {
		t.Fatalf("SignedString error: %v", err)
	}

	if _, err = tm.RefreshTokenPair(ctx, tokenStr); err == nil {
		t.Fatalf("Expected RefreshTokenPair error for empty subject")
	}
	consumed, err := tm.refreshStore.IsConsumed(ctx, "id-empty-sub")
	if err != nil || consumed {
		t.Fatalf("failed pre-generation must not consume jti: consumed=%v err=%v", consumed, err)
	}
}

func TestValidationCache_LRUAndExpiry(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	c := newValidationCache()

	mkClaims := func(sub string) *StandardClaims {
		return &StandardClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   sub,
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			},
			TokenType: AccessToken,
		}
	}

	for _, k := range []string{"a", "b", "c"} {
		if err := c.put(k, mkClaims(k), now, time.Minute, 2); err != nil {
			t.Fatalf("put %s: %v", k, err)
		}
	}
	if _, _, ok := c.get("a", now); ok {
		t.Fatalf("expected oldest key 'a' evicted at capacity 2")
	}
	if got, err, ok := c.get("b", now); !ok || err != nil || got.Subject != "b" {
		t.Fatalf("expected 'b' hit, got=%+v err=%v ok=%v", got, err, ok)
	}
	// Accessing 'b' promotes it, so inserting 'd' evicts 'c'.
	if err := c.put("d", mkClaims("d"), now, time.Minute, 2); err != nil {
		t.Fatalf("put d: %v", err)
	}
	if _, _, ok := c.get("c", now); ok {
		t.Fatalf("expected LRU key 'c' evicted")
	}
	if _, _, ok := c.get("b", now); !ok {
		t.Fatalf("expected promoted key 'b' retained")
	}

	// Expire entries via clock advance.
	if _, _, ok := c.get("b", now.Add(2*time.Minute)); ok {
		t.Fatalf("expected expired entry miss")
	}
	if cleaned := c.cleanExpired(now.Add(2 * time.Minute)); cleaned != 1 || c.size() != 0 {
		t.Fatalf("expected cleanExpired=1 size=0, got cleaned=%d size=%d", cleaned, c.size())
	}
}

func TestTokenManager_RevokeToken_EmptyAndInvalid(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	ctx := context.Background()
	if !errors.Is(tm.RevokeToken(ctx, ""), ErrEmptyToken) {
		t.Fatalf("Expected ErrEmptyToken")
	}
	if !errors.Is(tm.RevokeToken(ctx, "invalid"), ErrInvalidTokenFormat) {
		t.Fatalf("Expected ErrInvalidTokenFormat")
	}
}

func TestTokenManager_RevokeToken_NoExpUsesDefaultExpire(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer managerShutdown(tm)
	ctx := context.Background()

	rawClaims := jwt.MapClaims{
		"sub":  "u-1",
		"type": "access",
		"sid":  "s-1",
		"jti":  "id-1",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, rawClaims)
	tokenStr, err := token.SignedString(tm.secretKey)
	if err != nil {
		t.Fatalf("SignedString error: %v", err)
	}

	before := time.Now()
	if err := tm.RevokeToken(ctx, tokenStr); err != nil {
		t.Fatalf("RevokeToken error: %v", err)
	}

	store := tm.revocations.(*MemoryRevocationStore)
	store.mu.RLock()
	expireAt, ok := store.entries[tokenRevocationKey(tokenStr)]
	store.mu.RUnlock()
	if !ok {
		t.Fatalf("Expected token in blacklist")
	}
	if !expireAt.After(before.Add(23 * time.Hour)) {
		t.Fatalf("Expected expireAt about now+24h, got %v", expireAt)
	}
}

func managerShutdown(tm *TokenManager) {
	tm.Shutdown()
}

func TestTokenManager_RevokeToken_RemovesCacheEntry(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	ctx := context.Background()
	tokenStr, err := tm.GenerateToken("u-1")
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	if _, err := tm.ValidateAccessToken(ctx, tokenStr); err != nil {
		t.Fatalf("ValidateAccessToken error: %v", err)
	}
	if tm.GetCacheSize() != 1 {
		t.Fatalf("Expected token cached")
	}
	if err := tm.RevokeToken(ctx, tokenStr); err != nil {
		t.Fatalf("RevokeToken error: %v", err)
	}
	if tm.GetCacheSize() != 0 {
		t.Fatalf("Expected cache entry removed")
	}
}

func TestTokenManager_IsTokenRevoked_ExpiredAndActive(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	ctx := context.Background()

	expiredTok := "aaaaaaaaaa.bbbbbbbbbb.cccccccccc"
	activeTok := "bbbbbbbbbb.cccccccccc.dddddddddd"
	store := tm.revocations.(*MemoryRevocationStore)
	store.mu.Lock()
	store.entries[tokenRevocationKey(expiredTok)] = time.Now().Add(-time.Second)
	store.entries[tokenRevocationKey(activeTok)] = time.Now().Add(time.Hour)
	store.mu.Unlock()

	if revoked, err := tm.IsTokenRevoked(ctx, expiredTok); err != nil || revoked {
		t.Fatalf("Expected expired token not revoked, revoked=%v err=%v", revoked, err)
	}
	if revoked, err := tm.IsTokenRevoked(ctx, activeTok); err != nil || !revoked {
		t.Fatalf("Expected active token revoked, revoked=%v err=%v", revoked, err)
	}
}

func TestTokenManager_CleanBlacklist_Cleans(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	ctx := context.Background()
	tokenStr := "cccccccccc.dddddddddd.eeeeeeeeee"

	store := tm.revocations.(*MemoryRevocationStore)
	store.mu.Lock()
	store.entries[tokenRevocationKey(tokenStr)] = time.Now().Add(-time.Second)
	store.mu.Unlock()

	if _, err := tm.CleanBlacklist(ctx); err != nil {
		t.Fatalf("CleanBlacklist: %v", err)
	}

	store.mu.RLock()
	_, ok := store.entries[tokenRevocationKey(tokenStr)]
	store.mu.RUnlock()
	if ok {
		t.Fatalf("Expected expired blacklist entry removed")
	}
}

func TestTokenManager_startCleanupRoutine_TickAndStop(t *testing.T) {
	opts := DefaultJWTOptions()
	opts.BlacklistCleanInterval = 5 * time.Millisecond
	tm, err := NewTokenManager(testSecretKey, opts)
	if err != nil {
		t.Fatalf("NewTokenManager error: %v", err)
	}
	ctx := context.Background()

	now := tm.now()
	_ = tm.cache.put("old.tok.sig", &StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(2 * time.Millisecond))},
		TokenType:        AccessToken,
	}, now, time.Minute, 10)

	tokenStr := "ffffffffff.gggggggggg.hhhhhhhhhh"
	store := tm.revocations.(*MemoryRevocationStore)
	store.mu.Lock()
	store.entries[tokenRevocationKey(tokenStr)] = now.Add(-time.Second)
	store.mu.Unlock()

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		size, _ := tm.GetBlacklistSize(ctx)
		if tm.GetCacheSize() == 0 && size == 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	size, _ := tm.GetBlacklistSize(ctx)
	if tm.GetCacheSize() != 0 || size != 0 {
		t.Fatalf("Expected cache and blacklist cleaned, cache=%d blacklist=%d", tm.GetCacheSize(), size)
	}
	tm.Shutdown()
}
