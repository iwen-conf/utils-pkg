package jwt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestFullAuthLifecycleStateMachine verifies the complete business authentication state machine:
// Login -> Access Validation -> Identity Extraction -> Reject Wrong Token Purpose ->
// Rotation #1 -> Old Refresh Rejected -> Rotation #2 -> Session Logout -> All Session Tokens Rejected.
func TestFullAuthLifecycleStateMachine(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	opts := DefaultJWTOptions()
	opts.BlacklistCleanInterval = 0
	opts.Issuer = "auth.service"
	opts.Audience = []string{"api.service"}
	opts.ExpectedIssuer = "auth.service"
	opts.ExpectedAudience = []string{"api.service"}
	opts.AccessTokenExpiry = 15 * time.Minute
	opts.RefreshTokenExpiry = 7 * 24 * time.Hour
	opts.Clock = func() time.Time { return now }

	tm, err := NewTokenManager(testSecretKey, opts)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	defer tm.Shutdown()

	ctx := context.Background()

	// Step 1: Login -> IssueTokenPair
	pair1, err := tm.IssueTokenPair(Identity{
		Subject:      "user-42",
		ActorType:    "member",
		TenantID:     "org-7",
		Roles:        []string{"editor", " reviewer "},
		Scopes:       []string{"docs:read", "docs:write"},
		CustomClaims: map[string]interface{}{"plan": "pro"},
	})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}
	if pair1.SessionID == "" || pair1.AccessExpiresIn != 900 || pair1.RefreshExpiresIn != 604800 {
		t.Fatalf("unexpected pair1 metadata: %+v", pair1)
	}

	// Step 2: Validate AccessToken & extract Identity via Authenticate
	id, claims1, err := tm.Authenticate(ctx, pair1.AccessToken)
	if err != nil {
		t.Fatalf("Authenticate(pair1.AccessToken): %v", err)
	}
	if id.Subject != "user-42" || id.ActorID != "user-42" || id.ActorType != "member" || id.TenantID != "org-7" {
		t.Fatalf("unexpected identity: %+v", id)
	}
	if strings.Join(id.Roles, ",") != "editor,reviewer" || id.CustomClaims["plan"] != "pro" {
		t.Fatalf("unexpected roles/custom claims: %+v", id)
	}
	if claims1.SessionID != pair1.SessionID {
		t.Fatalf("session id mismatch: got %q, want %q", claims1.SessionID, pair1.SessionID)
	}

	// Step 3: Purpose isolation invariants
	if _, err := tm.ValidateAccessToken(ctx, pair1.RefreshToken); !errors.Is(err, ErrTokenTypeMismatch) {
		t.Fatalf("RefreshToken must be rejected by ValidateAccessToken: %v", err)
	}
	if _, err := tm.ValidateRefreshToken(ctx, pair1.AccessToken); !errors.Is(err, ErrTokenTypeMismatch) {
		t.Fatalf("AccessToken must be rejected by ValidateRefreshToken: %v", err)
	}
	if _, err := tm.RefreshTokenPair(ctx, pair1.AccessToken); !errors.Is(err, ErrTokenTypeMismatch) {
		t.Fatalf("AccessToken must be rejected by RefreshTokenPair: %v", err)
	}

	// Step 4: Advance clock 10 minutes & rotate #1
	now = now.Add(10 * time.Minute)
	pair2, err := tm.RefreshTokenPair(ctx, pair1.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshTokenPair #1: %v", err)
	}
	if pair2.SessionID != pair1.SessionID || pair2.RefreshToken == pair1.RefreshToken {
		t.Fatalf("unexpected rotated pair2: %+v", pair2)
	}

	// Old refresh token is immediately invalid on ValidateRefreshToken (without triggering reuse yet).
	if _, err := tm.ValidateRefreshToken(ctx, pair1.RefreshToken); !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("rotated pair1.RefreshToken should fail ValidateRefreshToken with ErrTokenRevoked, got %v", err)
	}

	// Step 5: Advance clock 10 more minutes (pair1.AccessToken is now expired, pair2 is active) & rotate #2
	now = now.Add(10 * time.Minute)
	if _, err := tm.ValidateAccessToken(ctx, pair1.AccessToken); !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("pair1.AccessToken should be expired at +20m, got %v", err)
	}
	id2, _, err := tm.Authenticate(ctx, pair2.AccessToken)
	if err != nil || id2.TenantID != "org-7" || id2.CustomClaims["plan"] != "pro" {
		t.Fatalf("pair2.AccessToken should preserve full identity: id=%+v err=%v", id2, err)
	}

	pair3, err := tm.RefreshTokenPair(ctx, pair2.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshTokenPair #2: %v", err)
	}

	// Step 6: Session logout via RevokeSessionByID -> ALL active tokens in session immediately fail!
	if err := tm.RevokeSessionByID(ctx, pair3.SessionID); err != nil {
		t.Fatalf("RevokeSessionByID: %v", err)
	}
	for name, tok := range map[string]string{
		"pair2.AccessToken": pair2.AccessToken,
		"pair3.AccessToken": pair3.AccessToken,
	} {
		if _, err := tm.ValidateAccessToken(ctx, tok); !errors.Is(err, ErrSessionRevoked) {
			t.Fatalf("%s must fail with ErrSessionRevoked after session logout, got %v", name, err)
		}
	}
	if _, err := tm.ValidateRefreshToken(ctx, pair3.RefreshToken); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("pair3.RefreshToken must fail with ErrSessionRevoked, got %v", err)
	}
	if _, err := tm.RefreshTokenPair(ctx, pair3.RefreshToken); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("RefreshTokenPair on revoked session must fail with ErrSessionRevoked, got %v", err)
	}
}

// TestRefreshReplayQuarantinesActiveSession verifies that replaying an already-rotated
// refresh token revokes the entire session and invalidates the newly rotated tokens.
func TestRefreshReplayQuarantinesActiveSession(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	ctx := context.Background()

	pair1, err := tm.IssueTokenPair(Identity{Subject: "u-replay", ActorType: "user"})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}

	// Legitimate client rotates -> obtains pair2.
	pair2, err := tm.RefreshTokenPair(ctx, pair1.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshTokenPair: %v", err)
	}
	// Warm pair2.AccessToken into validation cache.
	if _, err := tm.ValidateAccessToken(ctx, pair2.AccessToken); err != nil {
		t.Fatalf("ValidateAccessToken(pair2): %v", err)
	}

	// Attacker replays stolen pair1.RefreshToken -> detected and quarantines session!
	if _, err := tm.RefreshTokenPair(ctx, pair1.RefreshToken); !errors.Is(err, ErrRefreshReuseDetected) {
		t.Fatalf("expected ErrRefreshReuseDetected on replay, got %v", err)
	}

	// Cached pair2.AccessToken and pair2.RefreshToken are both immediately rejected!
	if _, err := tm.ValidateAccessToken(ctx, pair2.AccessToken); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("cached pair2.AccessToken must be rejected with ErrSessionRevoked, got %v", err)
	}
	if _, err := tm.RefreshTokenPair(ctx, pair2.RefreshToken); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("pair2.RefreshToken must be rejected with ErrSessionRevoked, got %v", err)
	}
}

// TestExplicitRevokeRefreshTokenDoesNotFalselyTriggerReuse verifies that revoking a
// refresh token via RevokeToken rejects it with ErrTokenRevoked without falsely nuking the session.
func TestExplicitRevokeRefreshTokenDoesNotFalselyTriggerReuse(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	ctx := context.Background()

	pair, err := tm.IssueTokenPair(Identity{Subject: "u-1", ActorType: "user"})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}
	if err := tm.RevokeToken(ctx, pair.RefreshToken); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	_, err = tm.RefreshTokenPair(ctx, pair.RefreshToken)
	if !errors.Is(err, ErrTokenRevoked) || errors.Is(err, ErrRefreshReuseDetected) {
		t.Fatalf("expected ErrTokenRevoked (not reuse), got %v", err)
	}
	// Access token in the same session remains valid because only the single token was revoked.
	if _, err := tm.ValidateAccessToken(ctx, pair.AccessToken); err != nil {
		t.Fatalf("AccessToken should remain valid when only RefreshToken was revoked: %v", err)
	}
}

// TestMultiInstanceConcurrentRefreshSingleWinner verifies that across 4 TokenManager instances
// sharing atomic stores, 32 concurrent refresh calls produce at most 1 successful rotation.
func TestMultiInstanceConcurrentRefreshSingleWinner(t *testing.T) {
	sharedRefresh := NewMemoryRefreshTokenUseStore()
	sharedRevoke := NewMemoryRevocationStore()

	instances := make([]*TokenManager, 4)
	for i := range instances {
		opts := DefaultJWTOptions()
		opts.BlacklistCleanInterval = 0
		opts.RefreshTokenUseStore = sharedRefresh
		opts.RevocationStore = sharedRevoke
		instances[i] = MustNewTokenManager(testSecretKey, opts)
		defer instances[i].Shutdown()
	}

	pair, err := instances[0].IssueTokenPair(Identity{Subject: "u-race", ActorType: "user"})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}

	const workers = 32
	start := make(chan struct{})
	var winners atomic.Int32
	var reuseDetections atomic.Int32
	var sessionRevocations atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		inst := instances[i%len(instances)]
		wg.Add(1)
		go func(m *TokenManager) {
			defer wg.Done()
			<-start
			_, err := m.RefreshTokenPair(context.Background(), pair.RefreshToken)
			switch {
			case err == nil:
				winners.Add(1)
			case errors.Is(err, ErrRefreshReuseDetected):
				reuseDetections.Add(1)
			case errors.Is(err, ErrSessionRevoked):
				sessionRevocations.Add(1)
			default:
				t.Errorf("unexpected concurrent refresh error: %v", err)
			}
		}(inst)
	}
	close(start)
	wg.Wait()

	if winners.Load() != 1 {
		t.Fatalf("expected exactly 1 winning refresh across instances, got %d", winners.Load())
	}
	if reuseDetections.Load()+sessionRevocations.Load() != workers-1 {
		t.Fatalf("expected %d rejected concurrent replays, got reuse=%d sessionRevoked=%d",
			workers-1, reuseDetections.Load(), sessionRevocations.Load())
	}
}

// TestTokenProfileValidationAndTokenSizeGuard verifies fail-fast constructor checks on TokenProfile
// and oversized token DoS protection.
func TestTokenProfileValidationAndTokenSizeGuard(t *testing.T) {
	// Reserved claim name in TokenProfile
	opts := DefaultJWTOptions()
	opts.TokenProfile = &TokenProfile{ActorTypeClaim: "sub"}
	if _, err := NewTokenManager(testSecretKey, opts); !errors.Is(err, ErrReservedClaimKey) {
		t.Fatalf("expected ErrReservedClaimKey for TokenProfile using 'sub', got %v", err)
	}

	// Duplicate claim name in TokenProfile
	opts = DefaultJWTOptions()
	opts.TokenProfile = &TokenProfile{ActorTypeClaim: "role", RolesClaim: "role"}
	if _, err := NewTokenManager(testSecretKey, opts); !errors.Is(err, ErrDuplicateClaimKey) {
		t.Fatalf("expected ErrDuplicateClaimKey for duplicate TokenProfile claim, got %v", err)
	}

	// IssueAccessToken rejects RefreshToken override
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	if _, err := tm.IssueAccessToken(Identity{Subject: "u-1"}, &TokenOptions{TokenType: RefreshToken}); !errors.Is(err, ErrTokenTypeMismatch) {
		t.Fatalf("expected ErrTokenTypeMismatch from IssueAccessToken, got %v", err)
	}

	// Oversized token rejected before touching RevocationStore
	var storeHits atomic.Int32
	countingOpts := DefaultJWTOptions()
	countingOpts.BlacklistCleanInterval = 0
	countingOpts.RevocationStore = &countingRevokeStore{hits: &storeHits}
	tmGuard := MustNewTokenManager(testSecretKey, countingOpts)
	defer tmGuard.Shutdown()

	huge := fmt.Sprintf("%s.%s.%s", strings.Repeat("a", 3000), strings.Repeat("b", 3000), strings.Repeat("c", 3000))
	if _, err := tmGuard.ValidateAccessToken(context.Background(), huge); !errors.Is(err, ErrTokenTooLarge) {
		t.Fatalf("expected ErrTokenTooLarge, got %v", err)
	}
	if storeHits.Load() != 0 {
		t.Fatalf("oversized token should be rejected before querying RevocationStore, hits=%d", storeHits.Load())
	}

	// Typed slice & struct in CustomClaims remain supported via JSON serialization
	type roleMeta struct {
		Level int `json:"level"`
	}
	structTok, err := tm.GenerateToken("u-typed", &TokenOptions{
		CustomClaims: map[string]interface{}{
			"ids":  []int{10, 20},
			"meta": roleMeta{Level: 3},
		},
	})
	if err != nil {
		t.Fatalf("GenerateToken with typed custom claims: %v", err)
	}
	parsedTyped, err := tm.ValidateAccessToken(context.Background(), structTok)
	if err != nil || parsedTyped.CustomClaims["meta"].(map[string]interface{})["level"] != float64(3) {
		t.Fatalf("ValidateAccessToken typed custom claims failed: claims=%+v err=%v", parsedTyped, err)
	}

	// Cross-issuer token cannot revoke session on a manager with ExpectedIssuer
	optsA := DefaultJWTOptions()
	optsA.BlacklistCleanInterval = 0
	optsA.Issuer = "svc-a"
	optsA.ExpectedIssuer = "svc-a"
	tmA := MustNewTokenManager(testSecretKey, optsA)
	defer tmA.Shutdown()

	optsB := DefaultJWTOptions()
	optsB.BlacklistCleanInterval = 0
	optsB.Issuer = "svc-b"
	tmB := MustNewTokenManager(testSecretKey, optsB)
	defer tmB.Shutdown()

	pairB, err := tmB.IssueTokenPair(Identity{Subject: "u-1"})
	if err != nil {
		t.Fatalf("tmB.IssueTokenPair: %v", err)
	}
	if err := tmA.RevokeSession(context.Background(), pairB.AccessToken); !errors.Is(err, jwt.ErrTokenInvalidIssuer) {
		t.Fatalf("expected cross-issuer RevokeSession to fail with ErrTokenInvalidIssuer, got %v", err)
	}
}

type countingRevokeStore struct {
	hits *atomic.Int32
}

func (s *countingRevokeStore) Revoke(context.Context, string, time.Time) error {
	s.hits.Add(1)
	return nil
}

func (s *countingRevokeStore) IsRevoked(context.Context, string) (bool, error) {
	s.hits.Add(1)
	return false, nil
}
