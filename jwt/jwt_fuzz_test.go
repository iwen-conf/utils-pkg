package jwt

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func FuzzStandardClaimsJSONRoundTrip(f *testing.F) {
	f.Add("u-1", "s-1", "access", "role-1")
	f.Add("", "", "refresh", "")

	f.Fuzz(func(t *testing.T, subject, sessionID, tokenType, roleID string) {
		tm := MustNewTokenManager(testSecretKey)
		defer tm.Shutdown()

		opts := &TokenOptions{
			TokenType: TokenType(tokenType),
			SessionID: sessionID,
			CustomClaims: map[string]interface{}{
				"user_current_role_id": roleID,
			},
		}

		if subject == "" {
			_, _ = tm.GenerateToken(subject, opts)
			return
		}

		tok, err := tm.GenerateToken(subject, opts)
		if err != nil {
			t.Skip()
		}

		claims, err := tm.ValidateAccessToken(context.Background(), tok)
		if err != nil {
			t.Skip()
		}

		b, err := json.Marshal(claims)
		if err != nil {
			t.Skip()
		}

		var rt StandardClaims
		if err := json.Unmarshal(b, &rt); err != nil {
			t.Skip()
		}

		if rt.Subject != claims.Subject {
			t.Fatalf("subject mismatch")
		}
	})
}

func FuzzValidateAccessToken_DoesNotPanic(f *testing.F) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	f.Add("invalid")
	f.Add("a.b.c")
	f.Add("aaaaaaaaaa.bbbbbbbbbb.cccccccccc")

	f.Fuzz(func(t *testing.T, tokenStr string) {
		_, _ = tm.ValidateAccessToken(context.Background(), tokenStr)
	})
}

func FuzzStandardClaimsUnmarshalJSON_DoesNotPanic(f *testing.F) {
	f.Add([]byte(`{"sub":"u","jti":"id","type":"access","sid":"s","x":1}`))
	f.Add([]byte(`{"jti":123}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`{`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var c StandardClaims
		_ = c.UnmarshalJSON(data)
	})
}

func FuzzValidateAccessToken_UnexpectedAlgDoesNotPanic(f *testing.F) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()
	f.Add("u-1")

	f.Fuzz(func(t *testing.T, subject string) {
		tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": subject})
		str, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			t.Skip()
		}
		_, _ = tm.ValidateAccessToken(context.Background(), str)
	})
}
