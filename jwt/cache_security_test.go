package jwt

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateToken_CacheCannotOutliveExp(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	if err := tm.SetCacheTTL(time.Minute); err != nil {
		t.Fatalf("SetCacheTTL: %v", err)
	}

	token, err := tm.GenerateToken("u-1", &TokenOptions{ExpiresIn: time.Second})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	claims, err := tm.ValidateToken(token)
	if err != nil {
		t.Fatalf("first ValidateToken: %v", err)
	}
	if tm.GetCacheSize() == 0 {
		t.Fatalf("expected success result to be cached")
	}

	wait := time.Until(claims.ExpiresAt.Time) + 20*time.Millisecond
	if wait < 20*time.Millisecond {
		wait = 20 * time.Millisecond
	}
	time.Sleep(wait)

	_, err = tm.ValidateToken(token)
	if !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("expected jwt.ErrTokenExpired after exp, got %v", err)
	}
	_, cachedErr, found := tm.checkCache(token)
	if found && cachedErr == nil {
		t.Fatalf("success cache must not survive token exp")
	}
	if _, err := tm.ValidateToken(token); !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("second post-exp validate: %v", err)
	}
}

func TestValidateToken_CacheTTLExpiresBeforeToken(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	if err := tm.SetCacheTTL(40 * time.Millisecond); err != nil {
		t.Fatalf("SetCacheTTL: %v", err)
	}

	token, err := tm.GenerateToken("u-1", &TokenOptions{ExpiresIn: time.Minute})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if _, err := tm.ValidateToken(token); err != nil {
		t.Fatalf("first ValidateToken: %v", err)
	}
	time.Sleep(60 * time.Millisecond)
	if _, err := tm.ValidateToken(token); err != nil {
		t.Fatalf("token should still be valid after cache TTL: %v", err)
	}
}

func TestCacheResult_NegativeCacheUsesTTL(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	if err := tm.SetCacheTTL(40 * time.Millisecond); err != nil {
		t.Fatalf("SetCacheTTL: %v", err)
	}

	cached := errors.New("cached-fail")
	if err := tm.cacheResult("bad-token", nil, cached); err != nil {
		t.Fatalf("cacheResult: %v", err)
	}
	_, got, found := tm.checkCache("bad-token")
	if !found || !errors.Is(got, cached) {
		t.Fatalf("expected negative cache hit, found=%v err=%v", found, got)
	}
	time.Sleep(60 * time.Millisecond)
	_, _, found = tm.checkCache("bad-token")
	if found {
		t.Fatalf("negative cache should expire with TTL")
	}
}

func TestCacheResult_DoesNotStoreNilOrExpiredSuccess(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	if err := tm.cacheResult("nil-success", nil, nil); err != nil {
		t.Fatalf("cacheResult nil claims: %v", err)
	}
	if tm.GetCacheSize() != 0 {
		t.Fatalf("nil success claims must not be cached")
	}

	expired := &StandardClaims{RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Second)),
	}}
	if err := tm.cacheResult("expired-success", expired, nil); err != nil {
		t.Fatalf("cacheResult expired: %v", err)
	}
	if tm.GetCacheSize() != 0 {
		t.Fatalf("already expired success must not be cached")
	}
}

func TestCacheResult_CloneFailureDoesNotGrowCache(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	claims := &StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
		CustomClaims: map[string]interface{}{
			"bad": make(chan int),
		},
	}
	before := tm.GetCacheSize()
	err := tm.cacheResult("bad-clone", claims, nil)
	if !errors.Is(err, ErrUnsupportedClaimType) {
		t.Fatalf("expected ErrUnsupportedClaimType, got %v", err)
	}
	if tm.GetCacheSize() != before {
		t.Fatalf("clone failure must not grow cache")
	}
}
