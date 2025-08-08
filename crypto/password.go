package crypto

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
	"unicode"
)

// PasswordPolicy 密码策略结构体
type PasswordPolicy struct {
	MinLength      int
	MaxLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireNumber  bool
	RequireSpecial bool
	DisallowWords  []string
	// 添加预编译正则表达式缓存
	disallowRegexes []*regexp.Regexp
	regexMutex      sync.RWMutex
}

// NewDefaultPasswordPolicy 创建默认密码策略
func NewDefaultPasswordPolicy() *PasswordPolicy {
	policy := &PasswordPolicy{
		MinLength:      8,
		MaxLength:      32,
		RequireUpper:   true,
		RequireLower:   true,
		RequireNumber:  true,
		RequireSpecial: true,
		DisallowWords:  []string{"password", "123456", "qwerty"},
	}

	// 预编译禁用词正则表达式
	policy.compileRegexes()

	return policy
}

// compileRegexes 预编译禁用词正则表达式
func (p *PasswordPolicy) compileRegexes() {
	p.regexMutex.Lock()
	defer p.regexMutex.Unlock()

	p.disallowRegexes = make([]*regexp.Regexp, len(p.DisallowWords))
	for i, word := range p.DisallowWords {
		p.disallowRegexes[i] = regexp.MustCompile(`(?i)` + regexp.QuoteMeta(word))
	}
}

// SetDisallowWords 设置禁用词并重新编译正则表达式
func (p *PasswordPolicy) SetDisallowWords(words []string) {
	p.regexMutex.Lock()
	p.DisallowWords = words
	p.regexMutex.Unlock()

	p.compileRegexes()
}

// ValidatePassword 验证密码是否符合策略
func (p *PasswordPolicy) ValidatePassword(password string) error {
	// 检查长度
	if len(password) < p.MinLength {
		return fmt.Errorf("密码长度不能小于 %d 个字符", p.MinLength)
	}
	if len(password) > p.MaxLength {
		return fmt.Errorf("密码长度不能大于 %d 个字符", p.MaxLength)
	}

	// 使用一次遍历检查所有字符类型
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

		// 如果所有必需条件都满足，提前退出循环
		if (!p.RequireUpper || hasUpper) &&
			(!p.RequireLower || hasLower) &&
			(!p.RequireNumber || hasNumber) &&
			(!p.RequireSpecial || hasSpecial) {
			break
		}
	}

	// 检查字符类型
	if p.RequireUpper && !hasUpper {
		return errors.New("密码必须包含大写字母")
	}
	if p.RequireLower && !hasLower {
		return errors.New("密码必须包含小写字母")
	}
	if p.RequireNumber && !hasNumber {
		return errors.New("密码必须包含数字")
	}
	if p.RequireSpecial && !hasSpecial {
		return errors.New("密码必须包含特殊字符")
	}

	// 检查禁用词
	p.regexMutex.RLock()
	regexes := p.disallowRegexes
	p.regexMutex.RUnlock()

	// 去除所有非字母数字字符
	passwordLower := nonAlphanumericRegex.ReplaceAllString(password, "")

	// 使用预编译的正则表达式
	for i, regex := range regexes {
		if regex.MatchString(passwordLower) {
			return fmt.Errorf("密码不能包含常见词汇: %s", p.DisallowWords[i])
		}
	}

	return nil
}

// PasswordHasher 密码哈希器接口
type PasswordHasher interface {
	Hash(password []byte) (string, error)
	Verify(hash, password []byte) (bool, error)
}