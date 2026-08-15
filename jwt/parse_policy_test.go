package jwt

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewTokenManager_SecretLength(t *testing.T) {
	_, err := NewTokenManager(strings.Repeat("a", 31))
	if !errors.Is(err, ErrSecretKeyTooShort) {
		t.Fatalf("31-byte secret: %v", err)
	}

	hexKey := "0123456789abcdef0123456789abcdef"
	tm, err := NewTokenManager(hexKey)
	if err != nil {
		t.Fatalf("32-byte hex secret should be accepted: %v", err)
	}
	tm.Shutdown()
}

func TestValidateToken_RejectsHS384(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"sub": "u-1",
		"exp": time.Now().Add(time.Minute).Unix(),
	})
	tokenStr, err := token.SignedString(tm.secretKey)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	if _, err := tm.ValidateToken(tokenStr); err == nil {
		t.Fatalf("HS384 token must be rejected")
	}
}

func TestValidateToken_RequiresExpiration(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "u-1",
		"jti": "id-1",
	})
	tokenStr, err := token.SignedString(tm.secretKey)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	if _, err := tm.ValidateToken(tokenStr); err == nil {
		t.Fatalf("token without exp must fail")
	}
}

func TestStandardClaims_MalformedRegisteredClaims(t *testing.T) {
	cases := []string{
		`{"exp":"invalid"}`,
		`{"aud":123}`,
		`{"iss":true}`,
		`{"jti":1}`,
		`{"type":2}`,
		`{"sid":3}`,
	}
	for _, raw := range cases {
		var c StandardClaims
		err := c.UnmarshalJSON([]byte(raw))
		if !errors.Is(err, ErrInvalidClaimType) {
			t.Fatalf("raw %s: expected ErrInvalidClaimType, got %v", raw, err)
		}
	}
}

func TestValidateToken_ExpectedIssuerAudience(t *testing.T) {
	opts := DefaultJWTOptions()
	opts.Issuer = "issuer-default"
	opts.Audience = []string{"aud-default"}
	opts.ExpectedIssuer = "issuer-expected"
	opts.ExpectedAudience = []string{"aud-expected"}
	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()

	okToken, err := tm.GenerateToken("u-1", &TokenOptions{
		Issuer:   "issuer-expected",
		Audience: []string{"aud-expected"},
	})
	if err != nil {
		t.Fatalf("GenerateToken expected: %v", err)
	}
	if _, err := tm.ValidateToken(okToken); err != nil {
		t.Fatalf("matching expected issuer/audience should pass: %v", err)
	}

	mismatch, err := tm.GenerateToken("u-1")
	if err != nil {
		t.Fatalf("GenerateToken default issuer: %v", err)
	}
	if _, err := tm.ValidateToken(mismatch); err == nil {
		t.Fatalf("default issued token must fail expected issuer/audience")
	}
}

func TestValidateToken_IssuerIsIssueDefaultOnly(t *testing.T) {
	opts := DefaultJWTOptions()
	opts.Issuer = "issuer-default"
	opts.Audience = []string{"aud-default"}
	tm := MustNewTokenManager(testSecretKey, opts)
	defer tm.Shutdown()

	other, err := tm.GenerateToken("u-1", &TokenOptions{Issuer: "other", Audience: []string{"other-aud"}})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if _, err := tm.ValidateToken(other); err != nil {
		t.Fatalf("without Expected* issue defaults must not reject custom iss/aud: %v", err)
	}
}

func TestParseTokenClaims_ValidateFalseStillRestrictsAlgAndTypes(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	hs384 := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"sub": "u-1",
		"exp": time.Now().Add(time.Minute).Unix(),
	})
	tokenStr, err := hs384.SignedString(tm.secretKey)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	if _, err := tm.parseTokenClaims(tokenStr, false); err == nil {
		t.Fatalf("validateClaims=false must still reject HS384")
	}

	var c StandardClaims
	if err := c.UnmarshalJSON([]byte(`{"jti":1}`)); !errors.Is(err, ErrInvalidClaimType) {
		t.Fatalf("malformed jti still fails without semantic validation: %v", err)
	}
}
