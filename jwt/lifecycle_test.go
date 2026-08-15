package jwt

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewTokenManager_InvalidOptions(t *testing.T) {
	opts := DefaultJWTOptions()
	opts.AccessTokenExpiry = 0
	if _, err := NewTokenManager(testSecretKey, opts); !errors.Is(err, ErrInvalidAccessTokenExpiry) {
		t.Fatalf("zero access expiry: %v", err)
	}

	opts = DefaultJWTOptions()
	opts.RefreshTokenExpiry = 0
	if _, err := NewTokenManager(testSecretKey, opts); !errors.Is(err, ErrInvalidRefreshTokenExpiry) {
		t.Fatalf("zero refresh expiry: %v", err)
	}

	opts = DefaultJWTOptions()
	opts.EnableCache = true
	opts.CacheSize = 0
	if _, err := NewTokenManager(testSecretKey, opts); !errors.Is(err, ErrInvalidCacheSize) {
		t.Fatalf("zero cache size: %v", err)
	}

	opts = DefaultJWTOptions()
	opts.CacheTTL = 0
	if _, err := NewTokenManager(testSecretKey, opts); !errors.Is(err, ErrInvalidCacheTTL) {
		t.Fatalf("zero cache ttl: %v", err)
	}

	opts = DefaultJWTOptions()
	opts.BlacklistCleanInterval = -time.Second
	if _, err := NewTokenManager(testSecretKey, opts); !errors.Is(err, ErrInvalidCleanupInterval) {
		t.Fatalf("negative cleanup: %v", err)
	}

	opts = DefaultJWTOptions()
	opts.RefreshTokenReuseAction = RefreshTokenReuseAction(99)
	if _, err := NewTokenManager(testSecretKey, opts); !errors.Is(err, ErrUnknownRefreshReuseAction) {
		t.Fatalf("unknown reuse action: %v", err)
	}
}

func TestTokenManager_SettersRejectInvalidValues(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	if err := tm.SetCacheTTL(0); !errors.Is(err, ErrInvalidCacheTTL) {
		t.Fatalf("SetCacheTTL(0): %v", err)
	}
	if err := tm.SetCacheSize(-1); !errors.Is(err, ErrInvalidCacheSize) {
		t.Fatalf("SetCacheSize(-1): %v", err)
	}
	if err := tm.SetTokenExpiry(AccessToken, 0); !errors.Is(err, ErrInvalidTokenExpiry) {
		t.Fatalf("SetTokenExpiry(0): %v", err)
	}
	if err := tm.SetTokenExpiry(TokenType("nope"), time.Minute); !errors.Is(err, ErrUnknownTokenType) {
		t.Fatalf("unknown type: %v", err)
	}
}

func TestTokenManager_ShutdownIdempotentAndConcurrent(t *testing.T) {
	opts := DefaultJWTOptions()
	opts.BlacklistCleanInterval = 20 * time.Millisecond
	tm := MustNewTokenManager(testSecretKey, opts)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tm.Shutdown()
		}()
	}
	wg.Wait()
	tm.Shutdown()
}

func TestGetBlacklistSizeContext_UnsupportedStore(t *testing.T) {
	opts := DefaultJWTOptions()
	opts.RevocationStore = unsupportedSizeStore{}
	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()

	_, err := tm.GetBlacklistSizeContext(context.Background())
	if !errors.Is(err, ErrBlacklistSizeUnsupported) {
		t.Fatalf("expected ErrBlacklistSizeUnsupported, got %v", err)
	}
}

func TestErrorReporter_ReceivesStoreErrorsWithoutSecrets(t *testing.T) {
	var mu sync.Mutex
	var seen []error
	storeErr := errors.New("backend unavailable")
	opts := DefaultJWTOptions()
	opts.RevocationStore = &reportingRevocationStore{err: storeErr}
	opts.ErrorReporter = func(err error) {
		mu.Lock()
		seen = append(seen, err)
		mu.Unlock()
	}
	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()

	_ = tm.IsBlacklisted("aaaaaaaaaa.bbbbbbbbbb.cccccccccc")
	tm.CleanBlacklist()

	mu.Lock()
	defer mu.Unlock()
	if len(seen) == 0 {
		t.Fatalf("expected reporter to receive store errors")
	}
	for _, err := range seen {
		msg := err.Error()
		if strings.Contains(msg, testSecretKey) || strings.Contains(msg, "aaaaaaaaaa.bbbbbbbbbb.cccccccccc") {
			t.Fatalf("reporter error leaked secret or token: %s", msg)
		}
		if !errors.Is(err, storeErr) {
			t.Fatalf("reporter error should wrap store error: %v", err)
		}
	}
}

func TestValidateGenerateConcurrentWithSetters(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				token, err := tm.GenerateToken("u-1")
				if err != nil {
					t.Errorf("GenerateToken: %v", err)
					return
				}
				if _, err := tm.ValidateToken(token); err != nil {
					t.Errorf("ValidateToken: %v", err)
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			_ = tm.SetCacheTTL(time.Minute)
			_ = tm.SetCacheSize(100)
			_ = tm.SetTokenExpiry(AccessToken, 10*time.Minute)
			tm.SetIssuer("iss")
			tm.SetAudience("aud")
			tm.EnableLog(i%2 == 0)
		}
	}()
	wg.Wait()
}

type unsupportedSizeStore struct{}

func (unsupportedSizeStore) Revoke(ctx context.Context, key string, expiresAt time.Time) error {
	return nil
}

func (unsupportedSizeStore) IsRevoked(ctx context.Context, key string) (bool, error) {
	return false, nil
}

type reportingRevocationStore struct {
	err error
}

func (s *reportingRevocationStore) Revoke(ctx context.Context, key string, expiresAt time.Time) error {
	return s.err
}

func (s *reportingRevocationStore) IsRevoked(ctx context.Context, key string) (bool, error) {
	return false, s.err
}

func (s *reportingRevocationStore) CleanExpired(ctx context.Context, now time.Time) (int, error) {
	return 0, s.err
}
