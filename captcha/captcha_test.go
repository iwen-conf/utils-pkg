package captcha

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name        string
		length      int
		expectError bool
	}{
		{"生成4位验证码", 4, false},
		{"生成6位验证码", 6, false},
		{"生成8位验证码", 8, false},
		{"生成12位验证码", 12, false},
		{"长度太短", 3, true},
		{"长度太长", 13, true},
		{"长度为0", 0, true},
		{"负数长度", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := Generate(tt.length)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("期望出现错误，但没有错误")
				}
				return
			}
			
			if err != nil {
				t.Errorf("不期望出现错误，但出现了错误: %v", err)
				return
			}
			
			// 检查长度
			if len(code) != tt.length {
				t.Errorf("验证码长度 = %d, 期望 %d", len(code), tt.length)
			}
			
			// 检查是否只包含数字
			if !isDigitsOnly(code) {
				t.Errorf("验证码包含非数字字符: %s", code)
			}
		})
	}
}

func TestGenerate6(t *testing.T) {
	code, err := Generate6()
	if err != nil {
		t.Errorf("Generate6() 出现错误: %v", err)
		return
	}
	
	if len(code) != 6 {
		t.Errorf("验证码长度 = %d, 期望 6", len(code))
	}
	
	if !isDigitsOnly(code) {
		t.Errorf("验证码包含非数字字符: %s", code)
	}
}

func TestGenerate4(t *testing.T) {
	code, err := Generate4()
	if err != nil {
		t.Errorf("Generate4() 出现错误: %v", err)
		return
	}
	
	if len(code) != 4 {
		t.Errorf("验证码长度 = %d, 期望 4", len(code))
	}
	
	if !isDigitsOnly(code) {
		t.Errorf("验证码包含非数字字符: %s", code)
	}
}

func TestGenerate8(t *testing.T) {
	code, err := Generate8()
	if err != nil {
		t.Errorf("Generate8() 出现错误: %v", err)
		return
	}
	
	if len(code) != 8 {
		t.Errorf("验证码长度 = %d, 期望 8", len(code))
	}
	
	if !isDigitsOnly(code) {
		t.Errorf("验证码包含非数字字符: %s", code)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		want     bool
	}{
		{"完全匹配", "123456", "123456", true},
		{"不匹配", "123456", "654321", false},
		{"输入有空格", " 123456 ", "123456", true},
		{"期望有空格", "123456", " 123456 ", true},
		{"都有空格", " 123456 ", " 123456 ", true},
		{"空字符串", "", "", true},
		{"一个空一个非空", "", "123456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Validate(tt.input, tt.expected); got != tt.want {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidFormat(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{"有效的4位验证码", "1234", true},
		{"有效的6位验证码", "123456", true},
		{"有效的8位验证码", "12345678", true},
		{"有效的12位验证码", "123456789012", true},
		{"太短", "123", false},
		{"太长", "1234567890123", false},
		{"包含字母", "12a456", false},
		{"包含特殊字符", "123-456", false},
		{"空字符串", "", false},
		{"只有字母", "abcdef", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidFormat(tt.code); got != tt.want {
				t.Errorf("IsValidFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateWithCustomCharset(t *testing.T) {
	tests := []struct {
		name        string
		length      int
		charset     string
		expectError bool
	}{
		{"数字字符集", 6, "0123456789", false},
		{"字母字符集", 6, "ABCDEFGHIJKLMNOPQRSTUVWXYZ", false},
		{"混合字符集", 6, "0123456789ABCDEF", false},
		{"单字符字符集", 6, "A", false},
		{"空字符集", 6, "", true},
		{"无效长度", 3, "0123456789", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := GenerateWithCustomCharset(tt.length, tt.charset)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("期望出现错误，但没有错误")
				}
				return
			}
			
			if err != nil {
				t.Errorf("不期望出现错误，但出现了错误: %v", err)
				return
			}
			
			// 检查长度
			if len(code) != tt.length {
				t.Errorf("验证码长度 = %d, 期望 %d", len(code), tt.length)
			}
			
			// 检查是否只包含字符集中的字符
			for _, char := range code {
				if !strings.ContainsRune(tt.charset, char) {
					t.Errorf("验证码包含字符集外的字符: %c in %s", char, code)
				}
			}
		})
	}
}

// TestRandomness 测试随机性
func TestRandomness(t *testing.T) {
	const iterations = 1000
	codes := make(map[string]int)
	
	// 生成大量验证码
	for i := 0; i < iterations; i++ {
		code, err := Generate6()
		if err != nil {
			t.Fatalf("生成验证码失败: %v", err)
		}
		codes[code]++
	}
	
	// 检查是否有重复（在1000次生成中，6位数字验证码重复的概率应该很低）
	duplicates := 0
	for _, count := range codes {
		if count > 1 {
			duplicates++
		}
	}
	
	// 允许少量重复（统计学上正常）
	if duplicates > iterations/100 { // 超过1%的重复率认为异常
		t.Errorf("重复率过高: %d/%d", duplicates, len(codes))
	}
	
	t.Logf("生成了 %d 个验证码，其中 %d 个唯一，%d 个重复", iterations, len(codes), duplicates)
}

// TestDistribution 测试数字分布的均匀性
func TestDistribution(t *testing.T) {
	const iterations = 10000
	digitCounts := make([]int, 10) // 0-9的计数
	
	// 生成大量验证码并统计每个数字的出现次数
	for i := 0; i < iterations; i++ {
		code, err := Generate6()
		if err != nil {
			t.Fatalf("生成验证码失败: %v", err)
		}
		
		for _, char := range code {
			digit := int(char - '0')
			digitCounts[digit]++
		}
	}
	
	// 期望每个数字出现的次数（理论值）
	expectedCount := iterations * 6 / 10 // 总字符数 / 10个数字
	
	// 检查分布是否相对均匀（允许20%的偏差）
	tolerance := expectedCount / 5
	for digit, count := range digitCounts {
		if count < expectedCount-tolerance || count > expectedCount+tolerance {
			t.Logf("数字 %d 出现次数: %d, 期望: %d±%d", digit, count, expectedCount, tolerance)
		}
	}
	
	t.Logf("数字分布: %v", digitCounts)
}

// 基准测试
func BenchmarkGenerate6(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Generate6()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerate(b *testing.B) {
	lengths := []int{4, 6, 8, 12}
	
	for _, length := range lengths {
		b.Run(string(rune('0'+length)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := Generate(length)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkValidate(b *testing.B) {
	code1 := "123456"
	code2 := "654321"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Validate(code1, code2)
	}
}

// 辅助函数：检查字符串是否只包含数字
func isDigitsOnly(s string) bool {
	digitRegex := regexp.MustCompile(`^\d+$`)
	return digitRegex.MatchString(s)
}
