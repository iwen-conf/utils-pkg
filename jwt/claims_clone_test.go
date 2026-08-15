package jwt

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestCloneStandardClaims_IsolatesMutation(t *testing.T) {
	src := &StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "sub-1",
			Audience:  jwt.ClaimStrings{"aud-1"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        "jti-1",
		},
		TokenID:   "jti-1",
		SessionID: "sid-1",
		CustomClaims: map[string]interface{}{
			"role": "admin",
			"meta": map[string]interface{}{"k": "v"},
			"tags": []interface{}{"a", "b"},
		},
	}

	cloned, err := cloneStandardClaims(src)
	if err != nil {
		t.Fatalf("cloneStandardClaims: %v", err)
	}
	cloned.Subject = "mutated"
	cloned.TokenID = "other"
	cloned.Audience[0] = "aud-mut"
	cloned.CustomClaims["role"] = "user"
	cloned.CustomClaims["meta"].(map[string]interface{})["k"] = "x"
	cloned.CustomClaims["tags"].([]interface{})[0] = "z"

	if src.Subject != "sub-1" || src.TokenID != "jti-1" || src.Audience[0] != "aud-1" {
		t.Fatalf("clone mutated source registered claims: %+v", src)
	}
	if src.CustomClaims["role"] != "admin" {
		t.Fatalf("clone mutated top-level custom claim")
	}
	if src.CustomClaims["meta"].(map[string]interface{})["k"] != "v" {
		t.Fatalf("clone mutated nested map")
	}
	if src.CustomClaims["tags"].([]interface{})[0] != "a" {
		t.Fatalf("clone mutated nested slice")
	}
}

func TestValidateToken_ReturnedClaimsMutationDoesNotAffectCache(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	token, err := tm.GenerateToken("u-1", &TokenOptions{
		SessionID: "sid-1",
		Audience:  []string{"svc"},
		CustomClaims: map[string]interface{}{
			"role": "admin",
			"meta": map[string]interface{}{"k": "v"},
			"tags": []interface{}{"a", "b"},
		},
	})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	first, err := tm.ValidateToken(token)
	if err != nil {
		t.Fatalf("first ValidateToken: %v", err)
	}
	first.Subject = "mutated"
	first.TokenID = "mutated-jti"
	if len(first.Audience) > 0 {
		first.Audience[0] = "mutated-aud"
	}
	first.CustomClaims["role"] = "user"
	first.CustomClaims["meta"].(map[string]interface{})["k"] = "x"
	first.CustomClaims["tags"].([]interface{})[0] = "z"

	second, err := tm.ValidateToken(token)
	if err != nil {
		t.Fatalf("second ValidateToken: %v", err)
	}
	if second.Subject != "u-1" || second.CustomClaims["role"] != "admin" {
		t.Fatalf("second validation saw mutated claims: subject=%s role=%v", second.Subject, second.CustomClaims["role"])
	}
	if second.Audience[0] != "svc" {
		t.Fatalf("audience mutated: %v", second.Audience)
	}
	if second.CustomClaims["meta"].(map[string]interface{})["k"] != "v" {
		t.Fatalf("nested map mutated: %v", second.CustomClaims["meta"])
	}
	if second.CustomClaims["tags"].([]interface{})[0] != "a" {
		t.Fatalf("nested slice mutated: %v", second.CustomClaims["tags"])
	}

	second.Subject = "again"
	second.CustomClaims["role"] = "guest"
	third, err := tm.ValidateToken(token)
	if err != nil {
		t.Fatalf("third ValidateToken: %v", err)
	}
	if third.Subject != "u-1" || third.CustomClaims["role"] != "admin" {
		t.Fatalf("third validation saw mutated claims")
	}
}

func TestValidateToken_ConcurrentMutationHasNoRace(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	token, err := tm.GenerateToken("u-1", &TokenOptions{
		Audience: []string{"svc"},
		CustomClaims: map[string]interface{}{
			"role": "admin",
			"tags": []interface{}{"a"},
		},
	})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if _, err := tm.ValidateToken(token); err != nil {
		t.Fatalf("seed ValidateToken: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claims, err := tm.ValidateToken(token)
			if err != nil {
				t.Errorf("ValidateToken: %v", err)
				return
			}
			claims.Subject = "mut"
			claims.Audience[0] = "mut"
			claims.CustomClaims["role"] = "mut"
			claims.CustomClaims["tags"].([]interface{})[0] = "mut"
		}()
	}
	wg.Wait()

	final, err := tm.ValidateToken(token)
	if err != nil {
		t.Fatalf("final ValidateToken: %v", err)
	}
	if final.Subject != "u-1" || final.Audience[0] != "svc" || final.CustomClaims["role"] != "admin" {
		t.Fatalf("concurrent mutation leaked into cache: %+v", final)
	}
}

func TestCloneClaimValue_UnsupportedType(t *testing.T) {
	_, err := cloneClaimValue(make(chan int))
	if !errors.Is(err, ErrUnsupportedClaimType) {
		t.Fatalf("expected ErrUnsupportedClaimType, got %v", err)
	}

	_, err = cloneClaimValue(json.Number("1"))
	if err != nil {
		t.Fatalf("json.Number should clone: %v", err)
	}
}

func TestCheckCache_CloneFailureReturnsErrorAndHit(t *testing.T) {
	tm := MustNewTokenManager(testSecretKey)
	defer tm.Shutdown()

	tm.cacheLock.Lock()
	tm.cache["bad"] = cacheItem{
		claims: &StandardClaims{
			CustomClaims: map[string]interface{}{"bad": make(chan int)},
		},
		timestamp:  time.Now(),
		validUntil: time.Now().Add(time.Minute),
	}
	tm.cacheLock.Unlock()

	claims, err, found := tm.checkCache("bad")
	if !found {
		t.Fatalf("expected hit")
	}
	if claims != nil {
		t.Fatalf("must not return shared object on clone failure")
	}
	if !errors.Is(err, ErrUnsupportedClaimType) {
		t.Fatalf("expected clone error, got %v", err)
	}
}
