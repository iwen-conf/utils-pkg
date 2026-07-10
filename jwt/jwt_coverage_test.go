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

	if m["iss"] != "issuer-1" {
		t.Fatalf("Expected iss=issuer-1, got %v", m["iss"])
	}
	if m["sub"] != "sub-1" {
		t.Fatalf("Expected sub=sub-1, got %v", m["sub"])
	}

	aud, ok := m["aud"].([]interface{})
	if !ok || len(aud) != 1 || aud[0] != "aud-1" {
		t.Fatalf("Expected aud=[aud-1], got %v", m["aud"])
	}

	if _, ok := m["exp"]; !ok {
		t.Fatalf("Expected exp present")
	}
	if _, ok := m["nbf"]; !ok {
		t.Fatalf("Expected nbf present")
	}
	if _, ok := m["iat"]; !ok {
		t.Fatalf("Expected iat present")
	}

	if m["jti"] != "jti-1" {
		t.Fatalf("Expected jti=jti-1, got %v", m["jti"])
	}
	if m["type"] != string(AccessToken) {
		t.Fatalf("Expected type=%s, got %v", AccessToken, m["type"])
	}
	if m["sid"] != "sid-1" {
		t.Fatalf("Expected sid=sid-1, got %v", m["sid"])
	}
	if m["k"] != "v" {
		t.Fatalf("Expected k=v, got %v", m["k"])
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
	_, err := NewTokenManager("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	if err == nil {
		t.Fatalf("Expected error for insufficient character variety")
	}
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

	tm.EnableCache(false)
	if tm.enableCache {
		t.Fatalf("Expected enableCache=false")
	}

	tm.SetCacheSize(7)
	if tm.cacheSize != 7 {
		t.Fatalf("Expected cacheSize=7, got %d", tm.cacheSize)
	}

	tm.SetCacheTTL(2 * time.Second)
	if tm.cacheTTL != 2*time.Second {
		t.Fatalf("Expected cacheTTL=2s, got %v", tm.cacheTTL)
	}

	if tm.GetCacheSize() != 0 {
		t.Fatalf("Expected cache size 0")
	}

	if tm.isTokenFormatValid("short") {
		t.Fatalf("Expected isTokenFormatValid(short)=false")
	}
}

func TestTokenManager_GenerateToken_ErrorOnUnmarshalableCustomClaims(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
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
	tm.EnableLog(true)

	if _, err := tm.GenerateToken("u-1"); err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
}

func TestTokenManager_ValidateToken_UnexpectedSigningMethod(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
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

	orig := parseWithClaims
	defer func() { parseWithClaims = orig }()

	parseWithClaims = func(tokenString string, claims jwt.Claims, keyFunc jwt.Keyfunc, options ...jwt.ParserOption) (*jwt.Token, error) {
		return &jwt.Token{Valid: false, Claims: jwt.MapClaims{}}, nil
	}

	_, err := tm.ValidateToken("aaaaaaaaaa.bbbbbbbbbb.cccccccccc")
	if err == nil || err.Error() != "invalid token" {
		t.Fatalf("Expected invalid token error, got %v", err)
	}
}

func TestTokenManager_RefreshToken_InvalidInput(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	_, _, err := tm.RefreshToken("invalid")
	if err == nil {
		t.Fatalf("Expected RefreshToken error")
	}
}

func TestTokenManager_RefreshToken_GenerateTokenFailure(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)

	rawClaims := jwt.MapClaims{
		"type": "refresh",
		"sid":  "s-1",
		"jti":  "id-1",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, rawClaims)
	tokenStr, err := token.SignedString(tm.secretKey)
	if err != nil {
		t.Fatalf("SignedString error: %v", err)
	}

	_, _, err = tm.RefreshToken(tokenStr)
	if err == nil {
		t.Fatalf("Expected RefreshToken error")
	}
}

func TestTokenManager_checkCache_ExpiredItemIsRemoved(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	tm.SetCacheTTL(time.Millisecond)

	tm.cacheLock.Lock()
	tm.cache["k"] = cacheItem{timestamp: time.Now().Add(-time.Second)}
	tm.cacheLock.Unlock()

	_, _, found := tm.checkCache("k")
	if found {
		t.Fatalf("Expected cache miss")
	}

	tm.cacheLock.RLock()
	_, ok := tm.cache["k"]
	tm.cacheLock.RUnlock()
	if ok {
		t.Fatalf("Expected expired cache entry removed")
	}
}

func TestTokenManager_checkCache_LogPathShortToken(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	tm.EnableLog(true)

	c := &StandardClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u-1"}}
	tm.cacheLock.Lock()
	tm.cache["short"] = cacheItem{claims: c, timestamp: time.Now()}
	tm.cacheLock.Unlock()

	claims, err, found := tm.checkCache("short")
	if !found || err != nil || claims == nil {
		t.Fatalf("Expected cache hit")
	}
	if claims.Subject != "u-1" {
		t.Fatalf("Expected subject u-1, got %q", claims.Subject)
	}
}

func TestTokenManager_cacheResult_EvictsOldest(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	tm.SetCacheSize(5)

	tm.cacheLock.Lock()
	tm.cache["old"] = cacheItem{timestamp: time.Now().Add(-time.Hour)}
	tm.cache["new"] = cacheItem{timestamp: time.Now()}
	tm.cache["mid"] = cacheItem{timestamp: time.Now().Add(-time.Minute)}
	tm.cache["x"] = cacheItem{timestamp: time.Now().Add(-2 * time.Minute)}
	tm.cache["y"] = cacheItem{timestamp: time.Now().Add(-3 * time.Minute)}
	tm.cacheLock.Unlock()

	tm.cacheResult("z", nil, nil)

	tm.cacheLock.RLock()
	_, ok := tm.cache["old"]
	tm.cacheLock.RUnlock()
	if ok {
		t.Fatalf("Expected oldest cache entry evicted")
	}
}

func TestTokenManager_evictOldestCache_EarlyReturn(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)

	tm.cacheLock.Lock()
	tm.cache["k"] = cacheItem{timestamp: time.Now()}
	tm.cacheLock.Unlock()

	tm.cacheLock.Lock()
	tm.evictOldestCache(0)
	tm.cacheLock.Unlock()
}

func TestTokenManager_evictOldestCache_CountOverLen(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)

	tm.cacheLock.Lock()
	tm.cache["a"] = cacheItem{timestamp: time.Now().Add(-time.Minute)}
	tm.cache["b"] = cacheItem{timestamp: time.Now()}
	tm.evictOldestCache(10)
	_, okA := tm.cache["a"]
	_, okB := tm.cache["b"]
	tm.cacheLock.Unlock()

	if okA || okB {
		t.Fatalf("Expected all entries evicted")
	}
}

func TestTokenManager_cleanCache_RemovesExpired(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	tm.EnableLog(true)
	tm.SetCacheTTL(time.Millisecond)

	tm.cacheLock.Lock()
	tm.cache["old"] = cacheItem{timestamp: time.Now().Add(-time.Second)}
	tm.cache["new"] = cacheItem{timestamp: time.Now()}
	tm.cacheLock.Unlock()

	tm.cleanCache()

	tm.cacheLock.RLock()
	_, okOld := tm.cache["old"]
	_, okNew := tm.cache["new"]
	tm.cacheLock.RUnlock()
	if okOld {
		t.Fatalf("Expected old cache entry removed")
	}
	if !okNew {
		t.Fatalf("Expected new cache entry kept")
	}
}

func TestTokenManager_RevokeToken_EmptyToken(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	if err := tm.RevokeToken(""); err == nil {
		t.Fatalf("Expected error")
	}
}

func TestTokenManager_RevokeToken_InvalidToken(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	if err := tm.RevokeToken("invalid"); err == nil {
		t.Fatalf("Expected error")
	}
}

func TestTokenManager_RevokeToken_NoExpUsesDefaultExpire(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
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
	tokenStr, err := tm.GenerateToken("u-1")
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}

	if _, err := tm.ValidateToken(tokenStr); err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}

	tm.cacheLock.RLock()
	_, ok := tm.cache[tokenStr]
	tm.cacheLock.RUnlock()
	if !ok {
		t.Fatalf("Expected token cached")
	}

	if err := tm.RevokeToken(tokenStr); err != nil {
		t.Fatalf("RevokeToken error: %v", err)
	}

	tm.cacheLock.RLock()
	_, ok = tm.cache[tokenStr]
	tm.cacheLock.RUnlock()
	if ok {
		t.Fatalf("Expected cache entry removed")
	}
}

func TestTokenManager_IsBlacklisted_ExpiredEntryIsRemoved(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	tm.EnableLog(true)
	tokenStr := "aaaaaaaaaa.bbbbbbbbbb.cccccccccc"

	store := tm.revocations.(*MemoryRevocationStore)
	store.mu.Lock()
	store.entries[tokenRevocationKey(tokenStr)] = time.Now().Add(-time.Second)
	store.mu.Unlock()

	if tm.IsBlacklisted(tokenStr) {
		t.Fatalf("Expected not blacklisted")
	}

	store.mu.RLock()
	_, ok := store.entries[tokenRevocationKey(tokenStr)]
	store.mu.RUnlock()
	if ok {
		t.Fatalf("Expected expired entry removed")
	}
}

func TestTokenManager_IsBlacklisted_NonExpiredTrue(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	tm.EnableLog(true)
	tokenStr := "bbbbbbbbbb.cccccccccc.dddddddddd"

	store := tm.revocations.(*MemoryRevocationStore)
	store.mu.Lock()
	store.entries[tokenRevocationKey(tokenStr)] = time.Now().Add(time.Hour)
	store.mu.Unlock()

	if !tm.IsBlacklisted(tokenStr) {
		t.Fatalf("Expected blacklisted")
	}
}

func TestTokenManager_CleanBlacklist_LogsAndCleans(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
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
	opts.BlacklistCleanInterval = 0

	tm, err := NewTokenManager(testSecretKey, opts)
	if err != nil {
		t.Fatalf("NewTokenManager error: %v", err)
	}

	tm.EnableLog(true)
	tm.SetCacheTTL(time.Millisecond)
	tm.SetCacheSize(10)

	tm.cacheLock.Lock()
	tm.cache["old"] = cacheItem{timestamp: time.Now().Add(-time.Second)}
	tm.cacheLock.Unlock()

	tokenStr := "ffffffffff.gggggggggg.hhhhhhhhhh"
	store := tm.revocations.(*MemoryRevocationStore)
	store.mu.Lock()
	store.entries[tokenRevocationKey(tokenStr)] = time.Now().Add(-time.Second)
	store.mu.Unlock()

	tm.cleanupTicker = time.NewTicker(time.Millisecond)
	done := make(chan struct{})
	go func() {
		tm.startCleanupRoutine()
		close(done)
	}()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if tm.GetCacheSize() == 0 && tm.GetBlacklistSize() == 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if tm.GetCacheSize() != 0 {
		t.Fatalf("Expected cache cleaned")
	}
	if tm.GetBlacklistSize() != 0 {
		t.Fatalf("Expected blacklist cleaned")
	}

	tm.Shutdown()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Expected cleanup routine stopped")
	}
}

func TestTokenManager_EnableCache_AndLogf(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	tm.EnableLog(true)
	tm.EnableCache(true)

	tm.cacheLock.Lock()
	tm.cache["short"] = cacheItem{err: errors.New("cached"), timestamp: time.Now()}
	tm.cacheLock.Unlock()

	_, err := tm.ValidateToken("short")
	if err == nil || err.Error() != "cached" {
		t.Fatalf("Expected cached error, got %v", err)
	}
}
