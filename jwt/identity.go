package jwt

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	if normalized.ActorTypeClaim == "" {
		normalized.ActorTypeClaim = defaults.ActorTypeClaim
	}
	if normalized.ActorIDClaim == "" {
		normalized.ActorIDClaim = defaults.ActorIDClaim
	}
	if normalized.TenantIDClaim == "" {
		normalized.TenantIDClaim = defaults.TenantIDClaim
	}
	if normalized.RolesClaim == "" {
		normalized.RolesClaim = defaults.RolesClaim
	}
	if normalized.ScopesClaim == "" {
		normalized.ScopesClaim = defaults.ScopesClaim
	}
	return normalized
}

func (p TokenProfile) clone() *TokenProfile {
	cloned := p
	return &cloned
}

// Identity is the normalized business identity used to issue tokens across entry points.
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
	// CustomClaims carries project-specific data. It must not collide with normalized keys.
	CustomClaims map[string]interface{}
}

// TokenPairOptions controls access/refresh pair generation.
type TokenPairOptions struct {
	SessionID      string
	AccessOptions  *TokenOptions
	RefreshOptions *TokenOptions
	CustomClaims   map[string]interface{}
}

// TokenPair contains a rotated access/refresh token pair.
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	TokenType        string
	SessionID        string
	AccessExpiresIn  int64
	RefreshExpiresIn int64
}

// GenerateIdentityToken signs a single token using normalized identity claims.
func (m *TokenManager) GenerateIdentityToken(identity Identity, options ...*TokenOptions) (string, error) {
	tokenOptions := DefaultTokenOptions()
	if len(options) > 0 && options[0] != nil {
		tokenOptions = options[0]
	}

	if tokenOptions.SessionID == "" {
		tokenOptions = cloneTokenOptions(tokenOptions)
		tokenOptions.SessionID = strings.TrimSpace(identity.SessionID)
	}

	claims, err := m.identityClaims(identity, tokenOptions.CustomClaims)
	if err != nil {
		return "", err
	}
	tokenOptions = cloneTokenOptions(tokenOptions)
	tokenOptions.CustomClaims = claims

	return m.GenerateToken(strings.TrimSpace(identity.Subject), tokenOptions)
}

// GenerateTokenPair signs access and refresh tokens with one shared session id.
func (m *TokenManager) GenerateTokenPair(subject string, options ...*TokenPairOptions) (*TokenPair, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil, errors.New("subject cannot be empty")
	}

	opts := defaultTokenPairOptions(options...)
	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		generatedSessionID, err := generateRandomHex(16)
		if err != nil {
			return nil, fmt.Errorf("failed to generate session id: %w", err)
		}
		sessionID = generatedSessionID
	}

	accessOptions, err := pairTokenOptions(opts.AccessOptions, AccessToken, sessionID, opts.CustomClaims)
	if err != nil {
		return nil, err
	}
	refreshOptions, err := pairTokenOptions(opts.RefreshOptions, RefreshToken, sessionID, opts.CustomClaims)
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

// GenerateIdentityTokenPair signs access and refresh tokens using normalized identity claims.
func (m *TokenManager) GenerateIdentityTokenPair(identity Identity, options ...*TokenPairOptions) (*TokenPair, error) {
	opts := defaultTokenPairOptions(options...)
	if opts.SessionID == "" {
		opts.SessionID = strings.TrimSpace(identity.SessionID)
	}

	commonClaims, err := m.identityClaims(identity, opts.CustomClaims)
	if err != nil {
		return nil, err
	}
	opts.CustomClaims = commonClaims

	return m.GenerateTokenPair(strings.TrimSpace(identity.Subject), opts)
}

// Identity returns normalized identity fields from claims using the default token profile.
func (c *StandardClaims) Identity(profile ...*TokenProfile) Identity {
	if c == nil {
		return Identity{}
	}
	p := normalizeTokenProfile(firstProfile(profile...))
	return Identity{
		Subject:      c.Subject,
		ActorType:    stringFromClaim(c.CustomClaims[p.ActorTypeClaim]),
		ActorID:      stringFromClaim(c.CustomClaims[p.ActorIDClaim]),
		TenantID:     stringFromClaim(c.CustomClaims[p.TenantIDClaim]),
		SessionID:    c.SessionID,
		Roles:        stringSliceFromClaim(c.CustomClaims[p.RolesClaim]),
		Scopes:       stringSliceFromClaim(c.CustomClaims[p.ScopesClaim]),
		CustomClaims: cloneMapClaims(c.CustomClaims),
	}
}

func (m *TokenManager) identityClaims(identity Identity, extra map[string]interface{}) (map[string]interface{}, error) {
	identity.Subject = strings.TrimSpace(identity.Subject)
	if identity.Subject == "" {
		return nil, errors.New("identity subject cannot be empty")
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
	opts.TokenType = tokenType
	opts.SessionID = sessionID

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
	snap := m.snapshotConfig()
	if tokenType == RefreshToken {
		return snap.refreshTokenExpiry
	}
	return snap.accessTokenExpiry
}

func putClaim(claims map[string]interface{}, key string, value interface{}) error {
	if key == "" {
		return nil
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

	if _, exists := claims[key]; exists {
		return fmt.Errorf("custom claim %q conflicts with normalized identity claim", key)
	}
	claims[key] = value
	return nil
}

func mergeMapClaims(dst map[string]interface{}, src map[string]interface{}) error {
	for k, v := range src {
		if _, exists := dst[k]; exists {
			return fmt.Errorf("custom claim %q is defined more than once", k)
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
		return "", errors.New("byte length must be positive")
	}
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
