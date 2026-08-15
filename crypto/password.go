package crypto

import (
	"errors"
	"fmt"
	"regexp"
	"sync/atomic"
	"unicode"
	"unicode/utf8"
)

var (
	// ErrInvalidPasswordPolicy 表示策略对象或配置非法。
	ErrInvalidPasswordPolicy = errors.New("invalid password policy")
	// ErrPasswordTooShort 密码过短
	ErrPasswordTooShort = errors.New("password is too short")
	// ErrPasswordTooLong 密码过长
	ErrPasswordTooLong = errors.New("password is too long")
	// ErrPasswordMissingUpper 缺少大写字母
	ErrPasswordMissingUpper = errors.New("password must contain uppercase letters")
	// ErrPasswordMissingLower 缺少小写字母
	ErrPasswordMissingLower = errors.New("password must contain lowercase letters")
	// ErrPasswordMissingNumber 缺少数字
	ErrPasswordMissingNumber = errors.New("password must contain numbers")
	// ErrPasswordMissingSpecial 缺少特殊字符
	ErrPasswordMissingSpecial = errors.New("password must contain special characters")
	// ErrPasswordDisallowedWord 包含禁用词
	ErrPasswordDisallowedWord = errors.New("password cannot contain common words")
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

// PasswordPolicyConfig 是密码策略的值对象配置。
type PasswordPolicyConfig struct {
	MinLength      int
	MaxLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireNumber  bool
	RequireSpecial bool
	DisallowWords  []string
}

type compiledPasswordPolicy struct {
	config  PasswordPolicyConfig
	regexes []*regexp.Regexp
	words   []string
}

// PasswordPolicy 保存不可变策略快照。
type PasswordPolicy struct {
	snapshot atomic.Pointer[compiledPasswordPolicy]
}

// DefaultPasswordPolicyConfig 返回默认策略配置。
func DefaultPasswordPolicyConfig() PasswordPolicyConfig {
	return PasswordPolicyConfig{
		MinLength:      8,
		MaxLength:      32,
		RequireUpper:   true,
		RequireLower:   true,
		RequireNumber:  true,
		RequireSpecial: true,
		DisallowWords:  []string{"password", "123456", "qwerty"},
	}
}

// NewDefaultPasswordPolicy 创建默认密码策略。
func NewDefaultPasswordPolicy() *PasswordPolicy {
	policy, err := NewPasswordPolicy(DefaultPasswordPolicyConfig())
	if err != nil {
		panic(err)
	}
	return policy
}

// NewPasswordPolicy 根据配置创建策略。
func NewPasswordPolicy(config PasswordPolicyConfig) (*PasswordPolicy, error) {
	compiled, err := compilePasswordPolicy(config)
	if err != nil {
		return nil, err
	}
	policy := &PasswordPolicy{}
	policy.snapshot.Store(compiled)
	return policy, nil
}

// Config 返回配置克隆。
func (p *PasswordPolicy) Config() (PasswordPolicyConfig, error) {
	compiled, err := p.load()
	if err != nil {
		return PasswordPolicyConfig{}, err
	}
	return clonePasswordPolicyConfig(compiled.config), nil
}

// SetConfig 原子替换完整策略快照。
func (p *PasswordPolicy) SetConfig(config PasswordPolicyConfig) error {
	if p == nil {
		return ErrInvalidPasswordPolicy
	}
	compiled, err := compilePasswordPolicy(config)
	if err != nil {
		return err
	}
	p.snapshot.Store(compiled)
	return nil
}

// SetDisallowWords 基于当前配置替换禁用词。
func (p *PasswordPolicy) SetDisallowWords(words []string) error {
	cfg, err := p.Config()
	if err != nil {
		return err
	}
	cfg.DisallowWords = append([]string(nil), words...)
	return p.SetConfig(cfg)
}

// ValidatePassword 基于一次快照验证密码。
func (p *PasswordPolicy) ValidatePassword(password string) error {
	compiled, err := p.load()
	if err != nil {
		return err
	}
	cfg := compiled.config
	length := utf8.RuneCountInString(password)
	if length < cfg.MinLength {
		return fmt.Errorf("%w: must be at least %d runes", ErrPasswordTooShort, cfg.MinLength)
	}
	if length > cfg.MaxLength {
		return fmt.Errorf("%w: must not exceed %d runes", ErrPasswordTooLong, cfg.MaxLength)
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	if cfg.RequireUpper && !hasUpper {
		return ErrPasswordMissingUpper
	}
	if cfg.RequireLower && !hasLower {
		return ErrPasswordMissingLower
	}
	if cfg.RequireNumber && !hasNumber {
		return ErrPasswordMissingNumber
	}
	if cfg.RequireSpecial && !hasSpecial {
		return ErrPasswordMissingSpecial
	}

	normalized := nonAlphanumericRegex.ReplaceAllString(password, "")
	for i, regex := range compiled.regexes {
		if regex.MatchString(normalized) {
			return fmt.Errorf("%w: %s", ErrPasswordDisallowedWord, compiled.words[i])
		}
	}
	return nil
}

func (p *PasswordPolicy) load() (*compiledPasswordPolicy, error) {
	if p == nil {
		return nil, ErrInvalidPasswordPolicy
	}
	compiled := p.snapshot.Load()
	if compiled == nil {
		return nil, ErrInvalidPasswordPolicy
	}
	return compiled, nil
}

func compilePasswordPolicy(config PasswordPolicyConfig) (*compiledPasswordPolicy, error) {
	if config.MinLength < 1 || config.MaxLength < config.MinLength {
		return nil, ErrInvalidPasswordPolicy
	}
	words := append([]string(nil), config.DisallowWords...)
	regexes := make([]*regexp.Regexp, 0, len(words))
	for _, word := range words {
		if word == "" {
			return nil, fmt.Errorf("%w: empty disallow word", ErrInvalidPasswordPolicy)
		}
		regexes = append(regexes, regexp.MustCompile(`(?i)`+regexp.QuoteMeta(word)))
	}
	cloned := clonePasswordPolicyConfig(config)
	cloned.DisallowWords = words
	return &compiledPasswordPolicy{config: cloned, regexes: regexes, words: words}, nil
}

func clonePasswordPolicyConfig(config PasswordPolicyConfig) PasswordPolicyConfig {
	cloned := config
	cloned.DisallowWords = append([]string(nil), config.DisallowWords...)
	return cloned
}

// PasswordHasher 密码哈希器接口
type PasswordHasher interface {
	Hash(password []byte) (string, error)
	Verify(hash, password []byte) (bool, error)
}
