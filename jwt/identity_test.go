package jwt

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestTokenManager_IssueTokenPair(t *testing.T) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()
	ctx := context.Background()

	pair, err := manager.IssueTokenPair(Identity{
		Subject:   "user-1",
		ActorType: "write",
		Roles:     []string{"author"},
		Scopes:    []string{"books:write"},
	})
	if err != nil {
		t.Fatalf("IssueTokenPair error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected both tokens, got %+v", pair)
	}
	if pair.SessionID == "" {
		t.Fatal("expected auto-generated session id")
	}

	accessClaims, err := manager.ValidateAccessToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken error: %v", err)
	}
	if accessClaims.Subject != "user-1" || accessClaims.SessionID != pair.SessionID {
		t.Fatalf("unexpected access claims: %+v", accessClaims)
	}
	identity := accessClaims.Identity()
	if identity.ActorType != "write" {
		t.Fatalf("expected actor type write, got %q", identity.ActorType)
	}
	if identity.ActorID != "user-1" {
		t.Fatalf("expected actor id to default to subject, got %q", identity.ActorID)
	}
	if identity.TenantID != "" {
		t.Fatalf("tenant id should be optional and absent, got %q", identity.TenantID)
	}
	if got := strings.Join(identity.Roles, ","); got != "author" {
		t.Fatalf("unexpected roles: %q", got)
	}

	if _, err := manager.ValidateAccessToken(ctx, pair.RefreshToken); err == nil {
		t.Fatal("ValidateAccessToken must reject refresh token")
	}
	refreshClaims, err := manager.ValidateRefreshToken(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken error: %v", err)
	}
	if refreshClaims.TokenType != RefreshToken {
		t.Fatalf("expected refresh token type, got %q", refreshClaims.TokenType)
	}
	if refreshClaims.CustomClaims[DefaultActorTypeClaim] != "write" {
		t.Fatalf("expected refresh token to carry identity claims, got %+v", refreshClaims.CustomClaims)
	}
}

func TestTokenManager_IssueTokenPair_CustomProfile(t *testing.T) {
	options := DefaultJWTOptions()
	options.TokenProfile = &TokenProfile{
		ActorTypeClaim: "kind",
		ActorIDClaim:   "principal",
		TenantIDClaim:  "tenant",
		RolesClaim:     "role_codes",
		ScopesClaim:    "permissions",
	}
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!", options)
	defer manager.Shutdown()

	pair, err := manager.IssueTokenPair(Identity{
		Subject:   "admin-1",
		ActorType: "admin",
		ActorID:   "admin-account-1",
		TenantID:  "tenant-1",
		Roles:     []string{"system_admin"},
	})
	if err != nil {
		t.Fatalf("IssueTokenPair error: %v", err)
	}

	claims, err := manager.ValidateAccessToken(context.Background(), pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken error: %v", err)
	}
	identity := claims.Identity(options.TokenProfile)
	if identity.ActorType != "admin" || identity.ActorID != "admin-account-1" || identity.TenantID != "tenant-1" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
	if _, exists := claims.CustomClaims[DefaultActorTypeClaim]; exists {
		t.Fatalf("default claim key should not be used with custom profile: %+v", claims.CustomClaims)
	}
}

func TestTokenManager_IssueTokenPair_RejectsClaimConflict(t *testing.T) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()

	_, err := manager.IssueTokenPair(Identity{
		Subject:   "user-1",
		ActorType: "user",
		CustomClaims: map[string]interface{}{
			DefaultActorTypeClaim: "admin",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "conflicts with normalized identity claim") {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestTokenManager_RefreshTokenPair_PreservesCustomClaims(t *testing.T) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()
	ctx := context.Background()

	pair, err := manager.IssueTokenPair(Identity{
		Subject:   "user-1",
		ActorType: "user",
		Scopes:    []string{"profile:read"},
	})
	if err != nil {
		t.Fatalf("IssueTokenPair error: %v", err)
	}

	rotated, err := manager.RefreshTokenPair(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshTokenPair error: %v", err)
	}

	accessClaims, err := manager.ValidateAccessToken(ctx, rotated.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken new access error: %v", err)
	}
	if accessClaims.Identity().ActorType != "user" {
		t.Fatalf("expected rotated access token to preserve identity claims, got %+v", accessClaims.CustomClaims)
	}

	refreshClaims, err := manager.ValidateRefreshToken(ctx, rotated.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken new refresh error: %v", err)
	}
	if refreshClaims.Identity().ActorType != "user" {
		t.Fatalf("expected rotated refresh token to preserve identity claims, got %+v", refreshClaims.CustomClaims)
	}
}

func TestTokenManager_GenerateToken_IssuerAudience(t *testing.T) {
	options := DefaultJWTOptions()
	options.Issuer = "utils-test"
	options.Audience = []string{"admin", "write", "user"}
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!", options)
	defer manager.Shutdown()

	token, err := manager.GenerateToken("user-1", &TokenOptions{ExpiresIn: time.Minute})
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	claims, err := manager.ValidateAccessToken(context.Background(), token)
	if err != nil {
		t.Fatalf("ValidateAccessToken error: %v", err)
	}
	if claims.Issuer != "utils-test" {
		t.Fatalf("expected issuer utils-test, got %q", claims.Issuer)
	}
	if got := strings.Join(claims.Audience, ","); got != "admin,write,user" {
		t.Fatalf("unexpected audience: %q", got)
	}
}

func TestTokenManager_IssueTokenPair_AccessCustomClaims(t *testing.T) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()

	pair, err := manager.IssueTokenPair(Identity{Subject: "user-1"}, &TokenPairOptions{
		AccessOptions: &TokenOptions{
			CustomClaims: map[string]interface{}{"entry": "admin"},
		},
	})
	if err != nil {
		t.Fatalf("IssueTokenPair error: %v", err)
	}
	claims, err := manager.ValidateAccessToken(context.Background(), pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken error: %v", err)
	}
	if claims.CustomClaims["entry"] != "admin" {
		t.Fatalf("expected access custom claim, got %+v", claims.CustomClaims)
	}
}

func TestTokenManager_RefreshTokenPair_PreservesIssuerAudience(t *testing.T) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()
	ctx := context.Background()

	refreshToken, err := manager.GenerateToken("user-1", &TokenOptions{
		TokenType: RefreshToken,
		Issuer:    "custom-issuer",
		Audience:  []string{"custom-audience"},
		SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}

	rotated, err := manager.RefreshTokenPair(ctx, refreshToken)
	if err != nil {
		t.Fatalf("RefreshTokenPair error: %v", err)
	}
	claims, err := manager.ValidateAccessToken(ctx, rotated.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken error: %v", err)
	}
	if claims.Issuer != "custom-issuer" {
		t.Fatalf("expected issuer to be preserved, got %q", claims.Issuer)
	}
	if got := strings.Join(claims.Audience, ","); got != "custom-audience" {
		t.Fatalf("expected audience to be preserved, got %q", got)
	}
}
