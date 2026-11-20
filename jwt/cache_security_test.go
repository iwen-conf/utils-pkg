package jwt

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateToken_CacheCannotOutliveExp(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	opts := DefaultJWTOptions()
	opts.BlacklistCleanInterval = 0
	opts.CacheTTL = time.Minute
	opts.Clock = func() time.Time { return now }

	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()

	token, err := tm.GenerateToken("u-1", &TokenOptions{ExpiresIn: 2 * time.Second})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if _, err := tm.ValidateToken(token); err != nil {
		t.Fatalf("first ValidateToken: %v", err)
	}
	if tm.GetCacheSize() != 1 {
		t.Fatalf("expected success result to be cached, got size=%d", tm.GetCacheSize())
	}

	now = now.Add(3 * time.Second)
	_, err = tm.ValidateToken(token)
	if !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("expected jwt.ErrTokenExpired after exp, got %v", err)
	}
	if _, _, found := tm.cache.get(token, now); found {
		t.Fatalf("success cache must not survive token exp")
	}
	if _, err := tm.ValidateToken(token); !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("second post-exp validate: %v", err)
	}
}

func TestValidateToken_CacheTTLExpiresBeforeToken(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	opts := DefaultJWTOptions()
	opts.BlacklistCleanInterval = 0
	opts.CacheTTL = 5 * time.Second
	opts.Clock = func() time.Time { return now }

	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()

	token, err := tm.GenerateToken("u-1", &TokenOptions{ExpiresIn: time.Minute})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if _, err := tm.ValidateToken(token); err != nil {
		t.Fatalf("first ValidateToken: %v", err)
	}

	now = now.Add(6 * time.Second)
	if _, _, found := tm.cache.get(token, now); found {
		t.Fatalf("expected cache entry to expire after CacheTTL")
	}
	if _, err := tm.ValidateToken(token); err != nil {
		t.Fatalf("token should still be valid after cache TTL: %v", err)
	}
}

func TestValidateToken_InvalidTokensDoNotPolluteOrEvictValidCache(t *testing.T) {
	opts := DefaultJWTOptions()
	opts.BlacklistCleanInterval = 0
	opts.CacheSize = 4
	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()

	validTokens := make([]string, 4)
	for i := range validTokens {
		tok, err := tm.GenerateToken(fmt.Sprintf("u-%d", i))
		if err != nil {
			t.Fatalf("GenerateToken: %v", err)
		}
		if _, err := tm.ValidateToken(tok); err != nil {
			t.Fatalf("ValidateToken: %v", err)
		}
		validTokens[i] = tok
	}

	// Flood with 100 invalid tokens -> must NOT evict any of the 4 cached valid tokens!
	for i := 0; i < 100; i++ {
		_, _ = tm.ValidateToken(fmt.Sprintf("bad-%d.invalid.signature", i))
	}
	if tm.GetCacheSize() != 4 {
		t.Fatalf("expected 4 valid cached tokens retained, got %d", tm.GetCacheSize())
	}
	for _, tok := range validTokens {
		if _, _, found := tm.cache.get(tok, tm.now()); !found {
			t.Fatalf("valid token was evicted by invalid token flood")
		}
	}
}

func TestCachePut_DoesNotStoreNilOrExpiredSuccess(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	now := tm.now()

	if err := tm.cache.put("nil-success", nil, now, time.Minute, 10); err != nil {
		t.Fatalf("cache.put nil claims: %v", err)
	}
	if tm.GetCacheSize() != 0 {
		t.Fatalf("nil success claims must not be cached")
	}

	expired := &StandardClaims{RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(-time.Second)),
	}}
	if err := tm.cache.put("expired-success", expired, now, time.Minute, 10); err != nil {
		t.Fatalf("cache.put expired: %v", err)
	}
	if tm.GetCacheSize() != 0 {
		t.Fatalf("already expired success must not be cached")
	}
}

func TestCachePut_CloneFailureDoesNotGrowCache(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	now := tm.now()

	claims := &StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		},
		CustomClaims: map[string]interface{}{
			"bad": make(chan int),
		},
	}
	before := tm.GetCacheSize()
	err := tm.cache.put("bad-clone", claims, now, time.Minute, 10)
	if !errors.Is(err, ErrUnsupportedClaimType) {
		t.Fatalf("expected ErrUnsupportedClaimType, got %v", err)
	}
	if tm.GetCacheSize() != before {
		t.Fatalf("clone failure must not grow cache")
	}
}

func TestCacheDynamicConfigUpdatesTakeImmediateEffect(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	opts := DefaultJWTOptions()
	opts.BlacklistCleanInterval = 0
	opts.CacheSize = 5
	opts.CacheTTL = 10 * time.Minute
	opts.Clock = func() time.Time { return now }

	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()

	for i := 0; i < 5; i++ {
		tok, err := tm.GenerateToken(fmt.Sprintf("u-%d", i), &TokenOptions{ExpiresIn: time.Hour})
		if err != nil {
			t.Fatalf("GenerateToken: %v", err)
		}
		if _, err := tm.ValidateToken(tok); err != nil {
			t.Fatalf("ValidateToken: %v", err)
		}
	}
	if tm.GetCacheSize() != 5 {
		t.Fatalf("expected 5 cached items, got %d", tm.GetCacheSize())
	}

	// Shrinking CacheSize immediately evicts excess LRU entries.
	if err := tm.SetCacheSize(2); err != nil {
		t.Fatalf("SetCacheSize: %v", err)
	}
	if tm.GetCacheSize() != 2 {
		t.Fatalf("expected cache resized to 2 immediately, got %d", tm.GetCacheSize())
	}

	// Shrinking CacheTTL clamps existing validUntil.
	if err := tm.SetCacheTTL(5 * time.Second); err != nil {
		t.Fatalf("SetCacheTTL: %v", err)
	}
	now = now.Add(6 * time.Second)
	if cleaned := tm.cache.cleanExpired(now); cleaned != 2 || tm.GetCacheSize() != 0 {
		t.Fatalf("expected clamped entries expired, cleaned=%d size=%d", cleaned, tm.GetCacheSize())
	}

	// EnableCache(false) clears existing cache entries immediately.
	tok, _ := tm.GenerateToken("u-clear")
	_, _ = tm.ValidateToken(tok)
	if tm.GetCacheSize() != 1 {
		t.Fatalf("expected 1 entry before disable")
	}
	tm.EnableCache(false)
	if tm.GetCacheSize() != 0 {
		t.Fatalf("expected cache cleared on EnableCache(false), got %d", tm.GetCacheSize())
	}
}
