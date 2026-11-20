package jwt

import (
	"encoding/json"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

func cloneNumericDate(src *jwt.NumericDate) *jwt.NumericDate {
	if src == nil {
		return nil
	}
	cloned := *src
	return &cloned
}

func cloneAudience(src jwt.ClaimStrings) jwt.ClaimStrings {
	if src == nil {
		return nil
	}
	dst := make(jwt.ClaimStrings, len(src))
	copy(dst, src)
	return dst
}

func cloneStandardClaims(src *StandardClaims) (*StandardClaims, error) {
	if src == nil {
		return nil, nil
	}

	cloned := *src
	cloned.Audience = cloneAudience(src.Audience)
	cloned.ExpiresAt = cloneNumericDate(src.ExpiresAt)
	cloned.NotBefore = cloneNumericDate(src.NotBefore)
	cloned.IssuedAt = cloneNumericDate(src.IssuedAt)
	if src.profile != nil {
		cloned.profile = src.profile.clone()
	}

	custom, err := cloneClaimValue(src.CustomClaims)
	if err != nil {
		return nil, err
	}
	switch typed := custom.(type) {
	case nil:
		cloned.CustomClaims = nil
	case map[string]interface{}:
		cloned.CustomClaims = typed
	default:
		return nil, fmt.Errorf("%w: custom claims became %T", ErrUnsupportedClaimType, custom)
	}
	return &cloned, nil
}

func cloneClaimValue(v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case string, bool, float32, float64,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		json.Number:
		return x, nil
	case []byte:
		if x == nil {
			return []byte(nil), nil
		}
		dst := make([]byte, len(x))
		copy(dst, x)
		return dst, nil
	case []string:
		return cloneStringSlice(x), nil
	case map[string]string:
		if x == nil {
			return map[string]string(nil), nil
		}
		dst := make(map[string]string, len(x))
		for k, val := range x {
			dst[k] = val
		}
		return dst, nil
	case []interface{}:
		if x == nil {
			return []interface{}(nil), nil
		}
		dst := make([]interface{}, len(x))
		for i, item := range x {
			cloned, err := cloneClaimValue(item)
			if err != nil {
				return nil, err
			}
			dst[i] = cloned
		}
		return dst, nil
	case map[string]interface{}:
		if x == nil {
			return map[string]interface{}(nil), nil
		}
		dst := make(map[string]interface{}, len(x))
		for k, val := range x {
			cloned, err := cloneClaimValue(val)
			if err != nil {
				return nil, err
			}
			dst[k] = cloned
		}
		return dst, nil
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedClaimType, v)
	}
}
