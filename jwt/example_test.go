package jwt_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/iwen-conf/utils-pkg/jwt"
)

func ExampleTokenManager_authLifecycle() {
	tm, err := jwt.NewTokenManager("0123456789abcdef0123456789abcdef")
	if err != nil {
		panic(err)
	}
	defer tm.Shutdown()

	ctx := context.Background()

	// 1. Issue Access + Refresh token pair from normalized business Identity.
	pair, err := tm.IssueTokenPair(jwt.Identity{
		Subject:   "user-1001",
		ActorType: "user",
		TenantID:  "tenant-acme",
		Roles:     []string{"member"},
		Scopes:    []string{"orders:read"},
	})
	if err != nil {
		panic(err)
	}

	// 2. Authenticate Access Token in middleware and get normalized Identity directly.
	identity, _, err := tm.Authenticate(ctx, pair.AccessToken)
	if err != nil {
		panic(err)
	}
	fmt.Printf("authenticated: sub=%s actor=%s tenant=%s\n", identity.Subject, identity.ActorType, identity.TenantID)

	// 3. Refresh Token cannot be misused as an Access Token.
	_, _, err = tm.Authenticate(ctx, pair.RefreshToken)
	fmt.Printf("refresh-as-access rejected: %v\n", errors.Is(err, jwt.ErrTokenTypeMismatch))

	// 4. Rotate token pair using RefreshTokenPair.
	rotated, err := tm.RefreshTokenPair(ctx, pair.RefreshToken)
	if err != nil {
		panic(err)
	}
	fmt.Printf("rotated same session: %v\n", rotated.SessionID == pair.SessionID)

	// 5. Logout / revoke session by ID (invalidates all tokens in that session).
	if err := tm.RevokeSessionByID(ctx, rotated.SessionID); err != nil {
		panic(err)
	}
	_, _, err = tm.Authenticate(ctx, rotated.AccessToken)
	fmt.Printf("after logout rejected: %v\n", errors.Is(err, jwt.ErrSessionRevoked))

	// Output:
	// authenticated: sub=user-1001 actor=user tenant=tenant-acme
	// refresh-as-access rejected: true
	// rotated same session: true
	// after logout rejected: true
}
