package jwt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// TokenProfile defines the custom claim keys used for normalized business identity.
type TokenProfile struct {
	ActorTypeClaim string
	ActorIDClaim   string
	TenantIDClaim  string
	RolesClaim     string
	ScopesClaim    string
}

const (
	DefaultActorTypeClaim = "actor_type"
	DefaultActorIDClaim   = "actor_id"
	DefaultTenantIDClaim  = "tenant_id"
	DefaultRolesClaim     = "roles"
	DefaultScopesClaim    = "scopes"
)

// DefaultTokenProfile returns the default normalized identity claim names.
func DefaultTokenProfile() *TokenProfile {
	return &TokenProfile{
		ActorTypeClaim: DefaultActorTypeClaim,
		ActorIDClaim:   DefaultActorIDClaim,
		TenantIDClaim:  DefaultTenantIDClaim,
		RolesClaim:     DefaultRolesClaim,
		ScopesClaim:    DefaultScopesClaim,
	}
}

func normalizeTokenProfile(profile *TokenProfile) TokenProfile {
	defaults := DefaultTokenProfile()
	if profile == nil {
		return *defaults
	}
	normalized := *profile
	if strings.TrimSpace(normalized.ActorTypeClaim) == "" {
		normalized.ActorTypeClaim = defaults.ActorTypeClaim
	}
	if strings.TrimSpace(normalized.ActorIDClaim) == "" {
		normalized.ActorIDClaim = defaults.ActorIDClaim
	}
	if strings.TrimSpace(normalized.TenantIDClaim) == "" {
		normalized.TenantIDClaim = defaults.TenantIDClaim
	}
	if strings.TrimSpace(normalized.RolesClaim) == "" {
		normalized.RolesClaim = defaults.RolesClaim
	}
	if strings.TrimSpace(normalized.ScopesClaim) == "" {
		normalized.ScopesClaim = defaults.ScopesClaim
	}
	return normalized
}

func (p TokenProfile) clone() *TokenProfile {
	cloned := p
	return &cloned
}

func (p TokenProfile) isProfileClaimKey(key string) bool {
	return key == p.ActorTypeClaim ||
		key == p.ActorIDClaim ||
		key == p.TenantIDClaim ||
		key == p.RolesClaim ||
		key == p.ScopesClaim
}

// Identity is the normalized business identity used to issue and authenticate tokens across entry points.
type Identity struct {
	// Subject becomes the JWT sub claim and must be stable for the authenticated account.
	Subject string
	// ActorType distinguishes entry points such as admin, write, or user.
	ActorType string
	// ActorID defaults to Subject when empty.
	ActorID string
	// TenantID is optional context. It is not required to issue or validate a token.
	TenantID string
	// SessionID is optional for a single token and auto-generated for token pairs when empty.
	SessionID string
	Roles     []string
	Scopes    []string
	// CustomClaims carries project-specific data. It must not collide with reserved or normalized keys.
	CustomClaims map[string]interface{}
}

// TokenPairOptions controls access/refresh pair generation.
type TokenPairOptions struct {
	SessionID      string
	AccessOptions  *TokenOptions
	RefreshOptions *TokenOptions
	CustomClaims   map[string]interface{}
}

// TokenPair contains an issued or rotated access/refresh token pair.
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	TokenType        string
	SessionID        string
	AccessExpiresIn  int64
	RefreshExpiresIn int64
}

// IssueTokenPair 是推荐的主入口：按统一业务身份签发共享会话 ID 的 Access/Refresh 令牌对。
func (m *TokenManager) IssueTokenPair(identity Identity, options ...*TokenPairOptions) (*TokenPair, error) {
	subject := strings.TrimSpace(identity.Subject)
	if subject == "" {
		return nil, fmt.Errorf("%w: identity subject cannot be empty", ErrEmptySubject)
	}

	opts := defaultTokenPairOptions(options...)
	if strings.TrimSpace(opts.SessionID) == "" {
		opts.SessionID = strings.TrimSpace(identity.SessionID)
	}

	commonClaims, err := m.identityClaims(identity, opts.CustomClaims)
	if err != nil {
		return nil, err
	}

	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		generatedSessionID, err := generateRandomHex(16)
		if err != nil {
			return nil, fmt.Errorf("failed to generate session id: %w", err)
		}
		sessionID = generatedSessionID
	}

	accessOptions, err := pairTokenOptions(opts.AccessOptions, AccessToken, sessionID, commonClaims)
	if err != nil {
		return nil, err
	}
	refreshOptions, err := pairTokenOptions(opts.RefreshOptions, RefreshToken, sessionID, commonClaims)
	if err != nil {
		return nil, err
	}

	accessToken, err := m.GenerateToken(subject, accessOptions)
	if err != nil {
		return nil, err
	}
	refreshToken, err := m.GenerateToken(subject, refreshOptions)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		SessionID:        sessionID,
		AccessExpiresIn:  int64(m.tokenExpiry(AccessToken, accessOptions).Seconds()),
		RefreshExpiresIn: int64(m.tokenExpiry(RefreshToken, refreshOptions).Seconds()),
	}, nil
}

// IssueAccessToken 按统一业务身份签发单个访问令牌（强制 TokenType 为 AccessToken）。
func (m *TokenManager) IssueAccessToken(identity Identity, options ...*TokenOptions) (string, error) {
	tokenOptions := DefaultTokenOptions()
	if len(options) > 0 && options[0] != nil {
		if options[0].TokenType != "" && options[0].TokenType != AccessToken {
			return "", fmt.Errorf("%w: IssueAccessToken only issues %q, got %q", ErrTokenTypeMismatch, AccessToken, options[0].TokenType)
		}
		tokenOptions = cloneTokenOptions(options[0])
	}
	tokenOptions.TokenType = AccessToken
	if strings.TrimSpace(tokenOptions.SessionID) == "" {
		tokenOptions.SessionID = strings.TrimSpace(identity.SessionID)
	}

	claims, err := m.identityClaims(identity, tokenOptions.CustomClaims)
	if err != nil {
		return "", err
	}
	tokenOptions.CustomClaims = claims

	return m.GenerateToken(strings.TrimSpace(identity.Subject), tokenOptions)
}

// Authenticate 是推荐的鉴权入口：验证 AccessToken 并直接返回归一化业务身份与声明。
// 若传入 RefreshToken、吊销令牌或会话已注销，均安全拒绝。
func (m *TokenManager) Authenticate(ctx context.Context, tokenStr string) (Identity, *StandardClaims, error) {
	claims, err := m.ValidateAccessToken(ctx, tokenStr)
	if err != nil {
		return Identity{}, nil, err
	}
	return m.ExtractIdentity(claims), claims, nil
}

// ExtractIdentity 使用当前管理器的 TokenProfile 从声明中提取业务身份。
func (m *TokenManager) ExtractIdentity(claims *StandardClaims) Identity {
	if claims == nil {
		return Identity{}
	}
	return claims.Identity(&m.tokenProfile)
}

// Identity 从声明中还原业务身份。
// 未显式传入 profile 时，自动优先使用验证该令牌的 TokenManager 所配置的 TokenProfile，
// 且返回的 Identity.CustomClaims 仅保留纯业务自定义字段（不混入归一化身份键），可直接再次用于签发。
func (c *StandardClaims) Identity(profile ...*TokenProfile) Identity {
	if c == nil {
		return Identity{}
	}
	selected := firstProfile(profile...)
	if selected == nil {
		selected = c.profile
	}
	p := normalizeTokenProfile(selected)

	var pureCustom map[string]interface{}
	if len(c.CustomClaims) > 0 {
		for k, v := range c.CustomClaims {
			if p.isProfileClaimKey(k) {
				continue
			}
			if pureCustom == nil {
				pureCustom = make(map[string]interface{})
			}
			clonedVal, err := cloneClaimValue(v)
			if err == nil {
				pureCustom[k] = clonedVal
			} else {
				pureCustom[k] = v
			}
		}
	}

	return Identity{
		Subject:      c.Subject,
		ActorType:    stringFromClaim(c.CustomClaims[p.ActorTypeClaim]),
		ActorID:      stringFromClaim(c.CustomClaims[p.ActorIDClaim]),
		TenantID:     stringFromClaim(c.CustomClaims[p.TenantIDClaim]),
		SessionID:    c.SessionID,
		Roles:        stringSliceFromClaim(c.CustomClaims[p.RolesClaim]),
		Scopes:       stringSliceFromClaim(c.CustomClaims[p.ScopesClaim]),
		CustomClaims: pureCustom,
	}
}

func (m *TokenManager) identityClaims(identity Identity, extra map[string]interface{}) (map[string]interface{}, error) {
	identity.Subject = strings.TrimSpace(identity.Subject)
	if identity.Subject == "" {
		return nil, fmt.Errorf("%w: identity subject cannot be empty", ErrEmptySubject)
	}

	if err := validateCustomClaimsKeys(identity.CustomClaims); err != nil {
		return nil, err
	}
	if err := validateCustomClaimsKeys(extra); err != nil {
		return nil, err
	}

	claims := makeMapClaims(identity.CustomClaims)
	if err := mergeMapClaims(claims, extra); err != nil {
		return nil, err
	}

	profile := m.tokenProfile
	if err := putClaim(claims, profile.ActorTypeClaim, strings.TrimSpace(identity.ActorType)); err != nil {
		return nil, err
	}

	actorID := strings.TrimSpace(identity.ActorID)
	if actorID == "" {
		actorID = identity.Subject
	}
	if err := putClaim(claims, profile.ActorIDClaim, actorID); err != nil {
		return nil, err
	}
	if err := putClaim(claims, profile.TenantIDClaim, strings.TrimSpace(identity.TenantID)); err != nil {
		return nil, err
	}
	if err := putClaim(claims, profile.RolesClaim, normalizeStrings(identity.Roles)); err != nil {
		return nil, err
	}
	if err := putClaim(claims, profile.ScopesClaim, normalizeStrings(identity.Scopes)); err != nil {
		return nil, err
	}

	return claims, nil
}

func defaultTokenPairOptions(options ...*TokenPairOptions) *TokenPairOptions {
	opts := &TokenPairOptions{}
	if len(options) > 0 && options[0] != nil {
		source := options[0]
		opts.SessionID = source.SessionID
		opts.AccessOptions = source.AccessOptions
		opts.RefreshOptions = source.RefreshOptions
		opts.CustomClaims = cloneMapClaims(source.CustomClaims)
	}
	return opts
}

func pairTokenOptions(options *TokenOptions, tokenType TokenType, sessionID string, common map[string]interface{}) (*TokenOptions, error) {
	opts := cloneTokenOptions(options)
	if opts == nil {
		opts = DefaultTokenOptions()
	}
	if opts.TokenType != "" && options != nil && options.TokenType != "" && options.TokenType != tokenType {
		return nil, fmt.Errorf("%w: pair %s options cannot override token type to %q", ErrTokenTypeMismatch, tokenType, options.TokenType)
	}
	opts.TokenType = tokenType
	opts.SessionID = sessionID

	if err := validateCustomClaimsKeys(common); err != nil {
		return nil, err
	}
	if err := validateCustomClaimsKeys(opts.CustomClaims); err != nil {
		return nil, err
	}

	claims := makeMapClaims(common)
	if err := mergeMapClaims(claims, opts.CustomClaims); err != nil {
		return nil, err
	}
	opts.CustomClaims = claims

	return opts, nil
}

func cloneTokenOptions(options *TokenOptions) *TokenOptions {
	if options == nil {
		return nil
	}
	cloned := *options
	cloned.Audience = cloneStringSlice(options.Audience)
	cloned.CustomClaims = cloneMapClaims(options.CustomClaims)
	return &cloned
}

func (m *TokenManager) tokenExpiry(tokenType TokenType, options *TokenOptions) time.Duration {
	if options != nil && options.ExpiresIn > 0 {
		return options.ExpiresIn
	}
	if tokenType == RefreshToken {
		return m.refreshTokenExpiry
	}
	return m.accessTokenExpiry
}

func putClaim(claims map[string]interface{}, key string, value interface{}) error {
	if key == "" {
		return nil
	}
	if _, exists := claims[key]; exists {
		return fmt.Errorf("%w: custom claim %q conflicts with normalized identity claim", ErrDuplicateClaimKey, key)
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			return nil
		}
	case []string:
		if len(v) == 0 {
			return nil
		}
		value = cloneStringSlice(v)
	}

	claims[key] = value
	return nil
}

func mergeMapClaims(dst map[string]interface{}, src map[string]interface{}) error {
	for k, v := range src {
		if _, exists := dst[k]; exists {
			return fmt.Errorf("%w: custom claim %q is defined more than once", ErrDuplicateClaimKey, k)
		}
		dst[k] = v
	}
	return nil
}

func cloneMapClaims(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	return makeMapClaims(src)
}

func makeMapClaims(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneStringSlice(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func normalizeStrings(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func firstProfile(profiles ...*TokenProfile) *TokenProfile {
	if len(profiles) > 0 {
		return profiles[0]
	}
	return nil
}

func stringFromClaim(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func stringSliceFromClaim(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return cloneStringSlice(v)
	case []interface{}:
		values := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				values = append(values, s)
			}
		}
		return values
	default:
		return nil
	}
}

func generateRandomHex(byteLen int) (string, error) {
	if byteLen <= 0 {
		return "", fmt.Errorf("byte length must be positive")
	}
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
