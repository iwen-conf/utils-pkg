package jwt

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestTokenManagerSharedRevocationStoreOverridesValidationCache(t *testing.T) {
	store := NewMemoryRevocationStore()
	issuer := newManagerWithRevocationStore(t, store)
	validator := newManagerWithRevocationStore(t, store)

	token, err := issuer.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if _, err := validator.ValidateAccessToken(context.Background(), token); err != nil {
		t.Fatalf("ValidateAccessToken(before revoke) error = %v", err)
	}
	if validator.GetCacheSize() == 0 {
		t.Fatal("validation cache is empty before shared revoke")
	}

	if err := issuer.RevokeToken(context.Background(), token); err != nil {
		t.Fatalf("RevokeToken() error = %v", err)
	}
	if _, err := validator.ValidateAccessToken(context.Background(), token); !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("ValidateAccessToken(after revoke) error = %v, want ErrTokenRevoked", err)
	}
}

func TestTokenManagerRevokeTokenIsIdempotentAndDoesNotStoreRawToken(t *testing.T) {
	store := NewMemoryRevocationStore()
	manager := newManagerWithRevocationStore(t, store)
	token, err := manager.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if err := manager.RevokeToken(context.Background(), token); err != nil {
		t.Fatalf("RevokeToken(first) error = %v", err)
	}
	if err := manager.RevokeToken(context.Background(), token); err != nil {
		t.Fatalf("RevokeToken(second) error = %v", err)
	}

	store.mu.RLock()
	defer store.mu.RUnlock()
	if len(store.entries) != 1 {
		t.Fatalf("revocation entries = %d, want 1", len(store.entries))
	}
	for key := range store.entries {
		if strings.Contains(key, token) {
			t.Fatalf("revocation key contains raw token: %q", key)
		}
	}
}

func TestTokenManagerRevokeSessionRejectsCachedSessionTokens(t *testing.T) {
	store := NewMemoryRevocationStore()
	issuer := newManagerWithRevocationStore(t, store)
	validator := newManagerWithRevocationStore(t, store)
	pair, err := issuer.IssueTokenPair(Identity{Subject: "admin-1"})
	if err != nil {
		t.Fatalf("IssueTokenPair() error = %v", err)
	}

	if _, err := validator.ValidateAccessToken(context.Background(), pair.AccessToken); err != nil {
		t.Fatalf("ValidateAccessToken(access before revoke) error = %v", err)
	}
	if _, err := validator.ValidateRefreshToken(context.Background(), pair.RefreshToken); err != nil {
		t.Fatalf("ValidateRefreshToken(refresh before revoke) error = %v", err)
	}
	if err := issuer.RevokeSession(context.Background(), pair.AccessToken); err != nil {
		t.Fatalf("RevokeSession(first) error = %v", err)
	}
	if err := issuer.RevokeSession(context.Background(), pair.RefreshToken); err != nil {
		t.Fatalf("RevokeSession(second) error = %v", err)
	}

	if _, err := validator.ValidateAccessToken(context.Background(), pair.AccessToken); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("ValidateAccessToken(access after revoke) error = %v, want ErrSessionRevoked", err)
	}
	if _, err := validator.ValidateRefreshToken(context.Background(), pair.RefreshToken); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("ValidateRefreshToken(refresh after revoke) error = %v, want ErrSessionRevoked", err)
	}
}

func TestTokenManagerRevokeSessionRejectsTokenWithoutSession(t *testing.T) {
	manager := newManagerWithRevocationStore(t, NewMemoryRevocationStore())
	token, err := manager.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if err := manager.RevokeSession(context.Background(), token); !errors.Is(err, ErrSessionIDMissing) {
		t.Fatalf("RevokeSession() error = %v, want ErrSessionIDMissing", err)
	}
}

func TestTokenManagerRevocationStoreErrorsFailClosed(t *testing.T) {
	storeErr := errors.New("revocation store unavailable")
	options := DefaultJWTOptions()
	options.BlacklistCleanInterval = 0
	options.RevocationStore = failingRevocationStore{err: storeErr}
	manager, err := NewTokenManager(testSecretKey, options)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	token, err := manager.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if _, err := manager.ValidateAccessToken(context.Background(), token); !errors.Is(err, storeErr) {
		t.Fatalf("ValidateAccessToken() error = %v, want wrapped store error", err)
	}
	if err := manager.RevokeToken(context.Background(), token); !errors.Is(err, storeErr) {
		t.Fatalf("RevokeToken() error = %v, want wrapped store error", err)
	}
}

func TestTokenManagerContextCancellationStopsValidation(t *testing.T) {
	manager := newManagerWithRevocationStore(t, NewMemoryRevocationStore())
	token, err := manager.GenerateToken("user-1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := manager.ValidateAccessToken(ctx, token); !errors.Is(err, context.Canceled) {
		t.Fatalf("ValidateAccessToken() error = %v, want context.Canceled", err)
	}
}

func TestMemoryRevocationStoreExpiresEntries(t *testing.T) {
	store := NewMemoryRevocationStore()
	ctx := context.Background()
	if err := store.Revoke(ctx, "token:test", time.Now().Add(time.Millisecond)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	revoked, err := store.IsRevoked(ctx, "token:test")
	if err != nil {
		t.Fatalf("IsRevoked() error = %v", err)
	}
	if revoked {
		t.Fatal("expired entry remains revoked")
	}
	size, err := store.Size(ctx)
	if err != nil {
		t.Fatalf("Size() error = %v", err)
	}
	if size != 0 {
		t.Fatalf("Size() = %d, want 0", size)
	}
}

func newManagerWithRevocationStore(t *testing.T, store RevocationStore) *TokenManager {
	t.Helper()
	options := DefaultJWTOptions()
	options.BlacklistCleanInterval = 0
	options.RevocationStore = store
	manager, err := NewTokenManager(testSecretKey, options)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	return manager
}

type failingRevocationStore struct {
	err error
}

func (s failingRevocationStore) Revoke(context.Context, string, time.Time) error {
	return s.err
}

func (s failingRevocationStore) IsRevoked(context.Context, string) (bool, error) {
	return false, s.err
}
