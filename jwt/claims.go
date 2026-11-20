package jwt

import (
	"encoding/json"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var reservedClaimKeys = map[string]struct{}{
	"iss":  {},
	"sub":  {},
	"aud":  {},
	"exp":  {},
	"nbf":  {},
	"iat":  {},
	"jti":  {},
	"type": {},
	"sid":  {},
}

func isReservedClaimKey(key string) bool {
	_, reserved := reservedClaimKeys[key]
	return reserved
}

func validateCustomClaimsKeys(claims map[string]interface{}) error {
	for k := range claims {
		if isReservedClaimKey(k) {
			return fmt.Errorf("%w: %q", ErrReservedClaimKey, k)
		}
	}
	return nil
}

// StandardClaims 标准 JWT 声明结构体。
type StandardClaims struct {
	jwt.RegisteredClaims
	TokenType TokenType `json:"type,omitempty"`
	SessionID string    `json:"sid,omitempty"`

	TokenID      string                 `json:"jti,omitempty"`
	CustomClaims map[string]interface{} `json:"-"`

	// profile 保存签发或验证该令牌的管理器身份映射配置，使 claims.Identity() 无需重复传参。
	profile *TokenProfile
}

func (c *StandardClaims) effectiveJTI() string {
	if c == nil {
		return ""
	}
	if c.TokenID != "" {
		return c.TokenID
	}
	return c.ID
}

// MarshalJSON 将标准声明与自定义声明序列化为扁平 JWT JSON 对象，并拒绝覆盖保留声明。
func (c *StandardClaims) MarshalJSON() ([]byte, error) {
	if err := validateCustomClaimsKeys(c.CustomClaims); err != nil {
		return nil, err
	}

	m := jwt.MapClaims{}
	for k, v := range c.CustomClaims {
		m[k] = v
	}

	if c.Issuer != "" {
		m["iss"] = c.Issuer
	}
	if c.Subject != "" {
		m["sub"] = c.Subject
	}
	if len(c.Audience) > 0 {
		m["aud"] = c.Audience
	}
	if c.ExpiresAt != nil {
		m["exp"] = c.ExpiresAt.Unix()
	}
	if c.NotBefore != nil {
		m["nbf"] = c.NotBefore.Unix()
	}
	if c.IssuedAt != nil {
		m["iat"] = c.IssuedAt.Unix()
	}
	if jti := c.effectiveJTI(); jti != "" {
		m["jti"] = jti
	}

	if c.TokenType != "" {
		m["type"] = c.TokenType
	}
	if c.SessionID != "" {
		m["sid"] = c.SessionID
	}

	return json.Marshal(m)
}

// UnmarshalJSON 反序列化扁平 JWT JSON 声明并校验所有注册字段类型。
func (c *StandardClaims) UnmarshalJSON(data []byte) error {
	var m jwt.MapClaims
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	custom := make(map[string]interface{}, len(m))
	for k, v := range m {
		custom[k] = v
	}

	subject, err := m.GetSubject()
	if err != nil {
		return fmt.Errorf("%w: sub: %w", ErrInvalidClaimType, err)
	}
	issuer, err := m.GetIssuer()
	if err != nil {
		return fmt.Errorf("%w: iss: %w", ErrInvalidClaimType, err)
	}
	if err := validateAudienceClaim(m["aud"]); err != nil {
		return fmt.Errorf("%w: aud: %w", ErrInvalidClaimType, err)
	}
	aud, err := m.GetAudience()
	if err != nil {
		return fmt.Errorf("%w: aud: %w", ErrInvalidClaimType, err)
	}
	issuedAt, err := m.GetIssuedAt()
	if err != nil {
		return fmt.Errorf("%w: iat: %w", ErrInvalidClaimType, err)
	}
	expiresAt, err := m.GetExpirationTime()
	if err != nil {
		return fmt.Errorf("%w: exp: %w", ErrInvalidClaimType, err)
	}
	notBefore, err := m.GetNotBefore()
	if err != nil {
		return fmt.Errorf("%w: nbf: %w", ErrInvalidClaimType, err)
	}

	jti, err := registeredStringClaim(m, "jti")
	if err != nil {
		return err
	}

	c.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   subject,
		Audience:  aud,
		ExpiresAt: expiresAt,
		NotBefore: notBefore,
		IssuedAt:  issuedAt,
		ID:        jti,
	}
	c.TokenID = jti
	c.TokenType = ""
	c.SessionID = ""

	if _, ok := m["type"]; ok {
		tokenType, err := registeredStringClaim(m, "type")
		if err != nil {
			return err
		}
		c.TokenType = TokenType(tokenType)
		delete(custom, "type")
	}
	if _, ok := m["sid"]; ok {
		sessionID, err := registeredStringClaim(m, "sid")
		if err != nil {
			return err
		}
		c.SessionID = sessionID
		delete(custom, "sid")
	}

	for _, k := range []string{"iss", "sub", "aud", "exp", "nbf", "iat", "jti"} {
		delete(custom, k)
	}

	c.CustomClaims = custom
	return nil
}

func validateAudienceClaim(v any) error {
	switch aud := v.(type) {
	case nil, string, []string:
		return nil
	case []interface{}:
		for _, item := range aud {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("aud contains non-string value")
			}
		}
		return nil
	default:
		return fmt.Errorf("aud is not string or string array")
	}
}

func registeredStringClaim(m jwt.MapClaims, name string) (string, error) {
	v, ok := m[name]
	if !ok || v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s is not string", ErrInvalidClaimType, name)
	}
	return s, nil
}
