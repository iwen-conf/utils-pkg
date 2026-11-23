package jwt

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func parseHeaderOnly(t *testing.T, tokenStr string) map[string]interface{} {
	t.Helper()
	tok, _, err := jwt.NewParser().ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("ParseUnverified: %v", err)
	}
	return tok.Header
}

func TestKeyRotation_SingleKeyBackwardCompatibility(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	ctx := context.Background()

	pair, err := tm.IssueTokenPair(Identity{Subject: "u-single", ActorType: "user"})
	if err != nil {
		t.Fatalf("IssueTokenPair: %v", err)
	}
	if _, hasKID := parseHeaderOnly(t, pair.AccessToken)["kid"]; hasKID {
		t.Fatalf("single-key manager without SigningKeyID should not write kid header")
	}
	if _, _, err := tm.Authenticate(ctx, pair.AccessToken); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
}

func TestKeyRotation_SigningKeyIDStampedAndRotatedOnRefresh(t *testing.T) {
	key1 := "key-v1-secret-0123456789abcdef012345"
	key2 := "key-v2-secret-fedcba9876543210fedcba"
	ctx := context.Background()

	opts1 := DefaultJWTOptions()
	opts1.BlacklistCleanInterval = 0
	opts1.SigningKeyID = "k1"
	tm1 := MustNewTokenManager(key1, opts1)
	defer tm1.Shutdown()

	pair1, err := tm1.IssueTokenPair(Identity{Subject: "u-rot", ActorType: "member"})
	if err != nil {
		t.Fatalf("tm1.IssueTokenPair: %v", err)
	}
	if got := parseHeaderOnly(t, pair1.AccessToken)["kid"]; got != "k1" {
		t.Fatalf("access token kid = %v, want k1", got)
	}
	if got := parseHeaderOnly(t, pair1.RefreshToken)["kid"]; got != "k1" {
		t.Fatalf("refresh token kid = %v, want k1", got)
	}

	singleAccess, err := tm1.IssueAccessToken(Identity{Subject: "u-rot", SessionID: "sess-k1"})
	if err != nil {
		t.Fatalf("tm1.IssueAccessToken: %v", err)
	}
	if got := parseHeaderOnly(t, singleAccess)["kid"]; got != "k1" {
		t.Fatalf("single access token kid = %v, want k1", got)
	}

	// Rotate to key2 while keeping k1 in VerificationKeys
	opts2 := DefaultJWTOptions()
	opts2.BlacklistCleanInterval = 0
	opts2.SigningKeyID = "k2"
	opts2.VerificationKeys = map[string][]byte{
		"k1": []byte(key1),
	}
	tm2 := MustNewTokenManager(key2, opts2)
	defer tm2.Shutdown()

	// Historical k1 access token still authenticates on tm2
	id1, _, err := tm2.Authenticate(ctx, pair1.AccessToken)
	if err != nil || id1.Subject != "u-rot" {
		t.Fatalf("tm2 should verify historical k1 access token: id=%+v err=%v", id1, err)
	}

	// Refreshing k1 refresh token rotates into k2-signed pair
	pair2, err := tm2.RefreshTokenPair(ctx, pair1.RefreshToken)
	if err != nil {
		t.Fatalf("tm2.RefreshTokenPair(k1): %v", err)
	}
	if got := parseHeaderOnly(t, pair2.AccessToken)["kid"]; got != "k2" {
		t.Fatalf("rotated access token kid = %v, want k2", got)
	}
	if got := parseHeaderOnly(t, pair2.RefreshToken)["kid"]; got != "k2" {
		t.Fatalf("rotated refresh token kid = %v, want k2", got)
	}
	if _, _, err := tm2.Authenticate(ctx, pair2.AccessToken); err != nil {
		t.Fatalf("tm2.Authenticate(k2): %v", err)
	}

	// Historical k1 tokens can also be revoked on tm2
	if err := tm2.RevokeToken(ctx, pair1.AccessToken); err != nil {
		t.Fatalf("RevokeToken(k1): %v", err)
	}
	if _, err := tm2.ValidateAccessToken(ctx, pair1.AccessToken); !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("expected ErrTokenRevoked for revoked k1 token, got %v", err)
	}
	if err := tm2.RevokeSession(ctx, singleAccess); err != nil {
		t.Fatalf("RevokeSession(k1): %v", err)
	}
	if _, err := tm2.ValidateAccessToken(ctx, singleAccess); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("expected ErrSessionRevoked after k1 session revoke, got %v", err)
	}
}

func TestKeyRotation_LegacyNoKIDVerificationAndUpgrade(t *testing.T) {
	legacyKey := "legacy-key-0123456789abcdef01234567"
	newKey := "new-key-fedcba9876543210fedcba987654"
	ctx := context.Background()

	legacyTM := MustNewTokenManager(legacyKey)
	defer legacyTM.Shutdown()

	legacyPair, err := legacyTM.IssueTokenPair(Identity{Subject: "u-legacy", ActorType: "user"})
	if err != nil {
		t.Fatalf("legacy IssueTokenPair: %v", err)
	}

	// Strict mode without LegacyVerificationKey rejects no-kid token
	strictOpts := DefaultJWTOptions()
	strictOpts.BlacklistCleanInterval = 0
	strictOpts.SigningKeyID = "k2"
	strictTM := MustNewTokenManager(newKey, strictOpts)
	defer strictTM.Shutdown()

	if _, err := strictTM.ValidateAccessToken(ctx, legacyPair.AccessToken); !errors.Is(err, ErrUnknownKeyID) {
		t.Fatalf("expected ErrUnknownKeyID for missing kid without LegacyVerificationKey, got %v", err)
	}

	// With LegacyVerificationKey configured, no-kid tokens verify and refresh upgrades to kid
	migratingOpts := DefaultJWTOptions()
	migratingOpts.BlacklistCleanInterval = 0
	migratingOpts.SigningKeyID = "k2"
	migratingOpts.LegacyVerificationKey = []byte(legacyKey)
	migratingTM := MustNewTokenManager(newKey, migratingOpts)
	defer migratingTM.Shutdown()

	if _, _, err := migratingTM.Authenticate(ctx, legacyPair.AccessToken); err != nil {
		t.Fatalf("expected legacy no-kid access token to pass with LegacyVerificationKey: %v", err)
	}
	upgraded, err := migratingTM.RefreshTokenPair(ctx, legacyPair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshTokenPair(legacy): %v", err)
	}
	if got := parseHeaderOnly(t, upgraded.AccessToken)["kid"]; got != "k2" {
		t.Fatalf("upgraded access token kid = %v, want k2", got)
	}
	if got := parseHeaderOnly(t, upgraded.RefreshToken)["kid"]; got != "k2" {
		t.Fatalf("upgraded refresh token kid = %v, want k2", got)
	}
}

func TestKeyRotation_RejectsMalformedOrUnknownKID(t *testing.T) {
	ctx := context.Background()
	opts := DefaultJWTOptions()
	opts.BlacklistCleanInterval = 0
	opts.SigningKeyID = "k2"
	opts.VerificationKeys = map[string][]byte{
		"k1": []byte("key-v1-secret-0123456789abcdef012345"),
	}
	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()

	signWithKID := func(kid interface{}, include bool) string {
		claims := &StandardClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "u-1",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
				ID:        "jti-kid",
			},
			TokenType: AccessToken,
			TokenID:   "jti-kid",
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		if include {
			tok.Header["kid"] = kid
		}
		str, err := tok.SignedString([]byte(testSecretKey))
		if err != nil {
			t.Fatalf("SignedString: %v", err)
		}
		return str
	}

	cases := []struct {
		name string
		tok  string
	}{
		{"unknown kid", signWithKID("k-unknown", true)},
		{"empty kid", signWithKID("", true)},
		{"whitespace kid", signWithKID("   ", true)},
		{"non-string kid", signWithKID(123, true)},
		{"missing kid", signWithKID(nil, false)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tm.ValidateAccessToken(ctx, tc.tok); !errors.Is(err, ErrUnknownKeyID) {
				t.Fatalf("%s: expected ErrUnknownKeyID, got %v", tc.name, err)
			}
		})
	}
}

func TestKeyRotation_ConstructorValidationAndCopyIsolation(t *testing.T) {
	validKey1 := []byte("key-v1-secret-0123456789abcdef012345")
	validKey2 := []byte("key-v2-secret-fedcba9876543210fedcba")

	// Whitespace-only SigningKeyID rejected
	opts := DefaultJWTOptions()
	opts.SigningKeyID = "   "
	if _, err := NewTokenManager(string(validKey2), opts); !errors.Is(err, ErrInvalidKeyID) {
		t.Fatalf("whitespace SigningKeyID: %v", err)
	}

	// Empty/whitespace VerificationKeys key rejected
	opts = DefaultJWTOptions()
	opts.VerificationKeys = map[string][]byte{"  ": validKey1}
	if _, err := NewTokenManager(string(validKey2), opts); !errors.Is(err, ErrInvalidKeyID) {
		t.Fatalf("whitespace VerificationKeys key: %v", err)
	}

	// Duplicate trimmed VerificationKeys key rejected
	opts = DefaultJWTOptions()
	opts.VerificationKeys = map[string][]byte{
		"k1":   validKey1,
		" k1 ": validKey1,
	}
	if _, err := NewTokenManager(string(validKey2), opts); !errors.Is(err, ErrDuplicateKeyID) {
		t.Fatalf("duplicate trimmed VerificationKeys key: %v", err)
	}

	// Conflicting secret under SigningKeyID rejected
	opts = DefaultJWTOptions()
	opts.SigningKeyID = "k2"
	opts.VerificationKeys = map[string][]byte{"k2": validKey1}
	if _, err := NewTokenManager(string(validKey2), opts); !errors.Is(err, ErrDuplicateKeyID) {
		t.Fatalf("conflicting SigningKeyID in VerificationKeys: %v", err)
	}

	// Matching secret under SigningKeyID allowed
	opts = DefaultJWTOptions()
	opts.SigningKeyID = "k2"
	opts.VerificationKeys = map[string][]byte{"k2": bytes.Clone(validKey2)}
	if tm, err := NewTokenManager(string(validKey2), opts); err != nil {
		t.Fatalf("matching SigningKeyID in VerificationKeys should succeed: %v", err)
	} else {
		tm.Shutdown()
	}

	// Short VerificationKeys secret rejected
	opts = DefaultJWTOptions()
	opts.VerificationKeys = map[string][]byte{"k1": []byte("short")}
	if _, err := NewTokenManager(string(validKey2), opts); !errors.Is(err, ErrSecretKeyTooShort) {
		t.Fatalf("short VerificationKeys entry: %v", err)
	}

	// Short LegacyVerificationKey rejected
	opts = DefaultJWTOptions()
	opts.LegacyVerificationKey = []byte("short")
	if _, err := NewTokenManager(string(validKey2), opts); !errors.Is(err, ErrSecretKeyTooShort) {
		t.Fatalf("short LegacyVerificationKey: %v", err)
	}

	// Deep-copy isolation against caller slice/map mutation
	k1Mut := bytes.Clone(validKey1)
	legacyMut := bytes.Clone(validKey1)
	vMap := map[string][]byte{"k1": k1Mut}
	opts = DefaultJWTOptions()
	opts.BlacklistCleanInterval = 0
	opts.SigningKeyID = "k2"
	opts.VerificationKeys = vMap
	opts.LegacyVerificationKey = legacyMut

	tm := MustNewTokenManager(string(validKey2), opts)
	defer tm.Shutdown()

	for i := range k1Mut {
		k1Mut[i] = 0
		legacyMut[i] = 0
	}
	vMap["k1"] = []byte("other")

	k1Signer := MustNewTokenManager(string(validKey1), &JWTOptions{
		AccessTokenExpiry:  time.Minute,
		RefreshTokenExpiry: time.Hour,
		SigningKeyID:       "k1",
	})
	defer k1Signer.Shutdown()
	tok, err := k1Signer.GenerateToken("u-iso")
	if err != nil {
		t.Fatalf("k1Signer.GenerateToken: %v", err)
	}
	if _, err := tm.ValidateAccessToken(context.Background(), tok); err != nil {
		t.Fatalf("caller key mutation affected TokenManager: %v", err)
	}
}

func TestOnDemandCleanup_SkipsTickerWhenNotNeeded(t *testing.T) {
	opts := DefaultJWTOptions()
	opts.EnableCache = false
	opts.BlacklistCleanInterval = 10 * time.Millisecond
	opts.RevocationStore = unsupportedSizeStore{}
	opts.RefreshTokenUseStore = minimalRefreshStore{}

	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()

	if tm.cleanupTicker != nil {
		t.Fatalf("expected no cleanupTicker when cache disabled and stores do not implement Cleaner")
	}
}

type minimalRefreshStore struct{}

func (minimalRefreshStore) Consume(context.Context, string, time.Time) (bool, error) {
	return true, nil
}

func (minimalRefreshStore) IsConsumed(context.Context, string) (bool, error) {
	return false, nil
}
