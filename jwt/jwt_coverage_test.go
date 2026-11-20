package jwt

import (
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

func TestTokenManager_SettersAndHelpers(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	tm.EnableCache(false)
	if tm.enableCache {
		t.Fatalf("Expected enableCache=false")
	}
	if err := tm.SetCacheSize(7); err != nil {
		t.Fatalf("SetCacheSize: %v", err)
	}
	if tm.cacheSize != 7 {
		t.Fatalf("Expected cacheSize=7, got %d", tm.cacheSize)
	}
	if err := tm.SetCacheTTL(2 * time.Second); err != nil {
		t.Fatalf("SetCacheTTL: %v", err)
	}
	if tm.cacheTTL != 2*time.Second {
		t.Fatalf("Expected cacheTTL=2s, got %v", tm.cacheTTL)
	}
	if tm.GetCacheSize() != 0 {
		t.Fatalf("Expected cache size 0")
	}
	if tm.isTokenFormatValid("short") || tm.isTokenFormatValid("a..bbbbbbbbb") || tm.isTokenFormatValid(".aaaa.bbbbbb") || tm.isTokenFormatValid("aaaa.bbbbbb.") {
		t.Fatalf("Expected invalid token formats to be rejected")
	}
}

func TestTokenManager_GenerateToken_ErrorOnUnmarshalableCustomClaims(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	tm.EnableLog(true)

	_, err := tm.GenerateToken("u-1", &TokenOptions{
		CustomClaims: map[string]interface{}{
			"bad": make(chan int),
		},
	})
	if err == nil {
		t.Fatalf("Expected GenerateToken error")
	}
}

func TestTokenManager_GenerateToken_LogOnSuccess(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	tm.EnableLog(true)

	if _, err := tm.GenerateToken("u-1"); err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
}

func TestTokenManager_ValidateToken_UnexpectedSigningMethod(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "u-1"})
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("SignedString error: %v", err)
	}
	if _, err := tm.ValidateToken(tokenStr); err == nil {
		t.Fatalf("Expected ValidateToken error")
	}
}

func TestTokenManager_ValidateToken_InvalidTokenNoError(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	orig := parseWithClaims
	defer func() { parseWithClaims = orig }()

	parseWithClaims = func(tokenString string, claims jwt.Claims, keyFunc jwt.Keyfunc, options ...jwt.ParserOption) (*jwt.Token, error) {
		return &jwt.Token{Valid: false, Claims: jwt.MapClaims{}}, nil
	}

	_, err := tm.ValidateToken("aaaaaaaaaa.bbbbbbbbbb.cccccccccc")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestTokenManager_RefreshToken_InvalidInput(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	_, _, err := tm.RefreshToken("invalid")
	if err == nil {
		t.Fatalf("Expected RefreshToken error")
	}
}

func TestTokenManager_RefreshToken_GenerateTokenFailureDoesNotConsumeJTI(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

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

	if _, _, err = tm.RefreshToken(tokenStr); err == nil {
		t.Fatalf("Expected RefreshToken error for empty subject")
	}
	consumed, err := tm.refreshStore.IsConsumed(nil, "id-empty-sub")
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
	if !errors.Is(tm.RevokeToken(""), ErrEmptyToken) {
		t.Fatalf("Expected ErrEmptyToken")
	}
	if !errors.Is(tm.RevokeToken("invalid"), ErrInvalidTokenFormat) {
		t.Fatalf("Expected ErrInvalidTokenFormat")
	}
}

func TestTokenManager_RevokeToken_NoExpUsesDefaultExpire(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	tm.EnableLog(true)

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
	if err := tm.RevokeToken(tokenStr); err != nil {
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

func TestTokenManager_RevokeToken_RemovesCacheEntry(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	tokenStr, err := tm.GenerateToken("u-1")
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	if _, err := tm.ValidateToken(tokenStr); err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if tm.GetCacheSize() != 1 {
		t.Fatalf("Expected token cached")
	}
	if err := tm.RevokeToken(tokenStr); err != nil {
		t.Fatalf("RevokeToken error: %v", err)
	}
	if tm.GetCacheSize() != 0 {
		t.Fatalf("Expected cache entry removed")
	}
}

func TestTokenManager_IsBlacklisted_ExpiredAndActive(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	tm.EnableLog(true)

	expiredTok := "aaaaaaaaaa.bbbbbbbbbb.cccccccccc"
	activeTok := "bbbbbbbbbb.cccccccccc.dddddddddd"
	store := tm.revocations.(*MemoryRevocationStore)
	store.mu.Lock()
	store.entries[tokenRevocationKey(expiredTok)] = time.Now().Add(-time.Second)
	store.entries[tokenRevocationKey(activeTok)] = time.Now().Add(time.Hour)
	store.mu.Unlock()

	if tm.IsBlacklisted(expiredTok) {
		t.Fatalf("Expected expired token not blacklisted")
	}
	if !tm.IsBlacklisted(activeTok) {
		t.Fatalf("Expected active token blacklisted")
	}
}

func TestTokenManager_CleanBlacklist_LogsAndCleans(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	tm.EnableLog(true)
	tokenStr := "cccccccccc.dddddddddd.eeeeeeeeee"

	store := tm.revocations.(*MemoryRevocationStore)
	store.mu.Lock()
	store.entries[tokenRevocationKey(tokenStr)] = time.Now().Add(-time.Second)
	store.mu.Unlock()

	tm.CleanBlacklist()

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
	tm.EnableLog(true)

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
		if tm.GetCacheSize() == 0 && tm.GetBlacklistSize() == 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if tm.GetCacheSize() != 0 || tm.GetBlacklistSize() != 0 {
		t.Fatalf("Expected cache and blacklist cleaned, cache=%d blacklist=%d", tm.GetCacheSize(), tm.GetBlacklistSize())
	}
	tm.Shutdown()
}
