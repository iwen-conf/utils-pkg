package jwt

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestRepro_ValidateAccessTokenRejectsRefreshToken verifies that a RefreshToken
// cannot pass the general access-token validation entry point.
func TestRepro_ValidateAccessTokenRejectsRefreshToken(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	ctx := context.Background()

	pair, err := tm.IssueTokenPair(Identity{
		Subject:   "user-1",
		ActorType: "user",
	})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}

	if _, err := tm.ValidateAccessToken(ctx, pair.RefreshToken); !errors.Is(err, ErrTokenTypeMismatch) {
		t.Fatalf("expected ErrTokenTypeMismatch when validating RefreshToken as AccessToken, got %v", err)
	}
	if _, _, err := tm.Authenticate(ctx, pair.RefreshToken); !errors.Is(err, ErrTokenTypeMismatch) {
		t.Fatalf("expected ErrTokenTypeMismatch in Authenticate for RefreshToken, got %v", err)
	}
}

// TestRepro_NotBeforeTokenNotPoisonedByCache verifies that a token validated
// before its nbf (e.g. clock skew) is never cached as invalid for CacheTTL.
func TestRepro_NotBeforeTokenNotPoisonedByCache(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	opts := DefaultJWTOptions()
	opts.BlacklistCleanInterval = 0
	opts.CacheTTL = 5 * time.Minute
	opts.Clock = func() time.Time { return now }

	tm, err := NewTokenManager(testSecretKey, opts)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	defer tm.Shutdown()
	ctx := context.Background()

	claims := &StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-nbf",
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(10 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
			ID:        "jti-nbf-1",
		},
		TokenType: AccessToken,
		TokenID:   "jti-nbf-1",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := tok.SignedString([]byte(testSecretKey))
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	if _, err := tm.ValidateAccessToken(ctx, tokenStr); !errors.Is(err, jwt.ErrTokenNotValidYet) {
		t.Fatalf("expected ErrTokenNotValidYet before nbf, got %v", err)
	}
	if tm.GetCacheSize() != 0 {
		t.Fatalf("failed validation must not pollute cache, size=%d", tm.GetCacheSize())
	}

	now = now.Add(11 * time.Second)
	if _, err := tm.ValidateAccessToken(ctx, tokenStr); err != nil {
		t.Fatalf("valid token after nbf was rejected: %v", err)
	}
}

// TestRepro_SmallCacheSizeDoesNotGrowUnbounded verifies that CacheSize < 5
// strictly bounds cache size (even CacheSize=1 or 3).
func TestRepro_SmallCacheSizeDoesNotGrowUnbounded(t *testing.T) {
	ctx := context.Background()
	for _, capSize := range []int{1, 2, 3, 4} {
		opts := DefaultJWTOptions()
		opts.BlacklistCleanInterval = 0
		opts.CacheSize = capSize
		tm, err := NewTokenManager(testSecretKey, opts)
		if err != nil {
			t.Fatalf("NewTokenManager(cap=%d): %v", capSize, err)
		}

		for i := 0; i < 20; i++ {
			tok, err := tm.GenerateToken(fmt.Sprintf("user-%d", i))
			if err != nil {
				t.Fatalf("GenerateToken: %v", err)
			}
			if _, err := tm.ValidateAccessToken(ctx, tok); err != nil {
				t.Fatalf("ValidateAccessToken: %v", err)
			}
			if got := tm.GetCacheSize(); got > capSize {
				t.Fatalf("CacheSize=%d grew to %d entries", capSize, got)
			}
		}
		tm.Shutdown()
	}
}

// TestRepro_CustomClaimsCannotOverwriteRegisteredClaims verifies that CustomClaims
// cannot inject or overwrite registered/reserved JWT claims like iss, aud, sid, type.
func TestRepro_CustomClaimsCannotOverwriteRegisteredClaims(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	for _, reserved := range []string{"iss", "sub", "aud", "exp", "nbf", "iat", "jti", "type", "sid"} {
		_, err := tm.GenerateToken("user-1", &TokenOptions{
			CustomClaims: map[string]interface{}{
				reserved: "spoofed",
			},
		})
		if !errors.Is(err, ErrReservedClaimKey) {
			t.Fatalf("expected ErrReservedClaimKey for %q, got %v", reserved, err)
		}
	}
}

type transientFailOnceRevokeStore struct {
	*MemoryRevocationStore
	failNextRevoke bool
}

func (s *transientFailOnceRevokeStore) Revoke(ctx context.Context, key string, expiresAt time.Time) error {
	if s.failNextRevoke {
		s.failNextRevoke = false
		return errors.New("transient redis write timeout")
	}
	return s.MemoryRevocationStore.Revoke(ctx, key, expiresAt)
}

// TestRepro_RefreshFailureDoesNotFalseTriggerReuseOnRetry verifies that if revoking
// the old refresh token fails transiently during rotation, the client's retry succeeds
// instead of falsely triggering ErrRefreshReuseDetected and revoking the entire session.
func TestRepro_RefreshFailureDoesNotFalseTriggerReuseOnRetry(t *testing.T) {
	revStore := &transientFailOnceRevokeStore{
		MemoryRevocationStore: NewMemoryRevocationStore(),
		failNextRevoke:        true,
	}
	opts := DefaultJWTOptions()
	opts.BlacklistCleanInterval = 0
	opts.RevocationStore = revStore
	tm, err := NewTokenManager(testSecretKey, opts)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	defer tm.Shutdown()

	pair, err := tm.IssueTokenPair(Identity{
		Subject:   "user-retry",
		ActorType: "user",
	})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}

	if _, err := tm.RefreshTokenPair(context.Background(), pair.RefreshToken); err == nil {
		t.Fatalf("expected transient error on first refresh")
	}

	rotated, err := tm.RefreshTokenPair(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatalf("client retry after transient failure failed: %v", err)
	}
	if rotated.AccessToken == "" || rotated.RefreshToken == "" || rotated.SessionID != pair.SessionID {
		t.Fatalf("unexpected rotated pair: %+v", rotated)
	}
}

// TestRepro_RefreshTokenWithoutSessionIDStillReturnsReuseError verifies that replaying
// a consumed refresh token without sid still returns ErrRefreshReuseDetected.
func TestRepro_RefreshTokenWithoutSessionIDStillReturnsReuseError(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	ctx := context.Background()

	rt, err := tm.GenerateToken("user-nosid", &TokenOptions{TokenType: RefreshToken})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if _, err := tm.RefreshTokenPair(ctx, rt); err != nil {
		t.Fatalf("first RefreshTokenPair: %v", err)
	}
	_, err = tm.RefreshTokenPair(ctx, rt)
	if !errors.Is(err, ErrRefreshReuseDetected) {
		t.Fatalf("expected ErrRefreshReuseDetected, got %v", err)
	}
}

// TestRepro_CustomTokenProfilePreservedAndRoundTrippable verifies that claims.Identity()
// automatically uses the TokenManager's TokenProfile and can be round-tripped back into IssueTokenPair.
func TestRepro_CustomTokenProfilePreservedAndRoundTrippable(t *testing.T) {
	opts := DefaultJWTOptions()
	opts.TokenProfile = &TokenProfile{
		ActorTypeClaim: "kind",
		ActorIDClaim:   "principal",
		TenantIDClaim:  "org",
		RolesClaim:     "role_list",
		ScopesClaim:    "perm_list",
	}
	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()
	ctx := context.Background()

	pair, err := tm.IssueTokenPair(Identity{
		Subject:      "u-1",
		ActorType:    "admin",
		TenantID:     "t-9",
		Roles:        []string{"root"},
		CustomClaims: map[string]interface{}{"region": "ap-east"},
	})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}
	claims, err := tm.ValidateAccessToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	got := claims.Identity()
	if got.ActorType != "admin" || got.TenantID != "t-9" || got.CustomClaims["region"] != "ap-east" {
		t.Fatalf("claims.Identity() lost fields: %+v", got)
	}
	reissued, err := tm.IssueTokenPair(got)
	if err != nil {
		t.Fatalf("round-trip IssueTokenPair(claims.Identity()) failed: %v", err)
	}
	id2, _, err := tm.Authenticate(ctx, reissued.AccessToken)
	if err != nil || id2.ActorType != "admin" || id2.CustomClaims["region"] != "ap-east" {
		t.Fatalf("Authenticate round-trip failed: id=%+v err=%v", id2, err)
	}
}

// TestRepro_GenerateTokenRejectsInvalidOptions verifies input validation on GenerateToken.
func TestRepro_GenerateTokenRejectsInvalidOptions(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	if _, err := tm.GenerateToken("   "); !errors.Is(err, ErrEmptySubject) {
		t.Fatalf("expected ErrEmptySubject, got %v", err)
	}
	if _, err := tm.GenerateToken("u-1", &TokenOptions{TokenType: TokenType("bogus")}); !errors.Is(err, ErrUnknownTokenType) {
		t.Fatalf("expected ErrUnknownTokenType, got %v", err)
	}
	if _, err := tm.GenerateToken("u-1", &TokenOptions{ExpiresIn: -time.Minute}); !errors.Is(err, ErrInvalidTokenExpiry) {
		t.Fatalf("expected ErrInvalidTokenExpiry, got %v", err)
	}
}
