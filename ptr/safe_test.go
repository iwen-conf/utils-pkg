package ptr

import (
	"testing"
	"time"
)

// 这个文件包含所有 Safe* 函数的测试
// 每个函数都测试 nil 和非 nil 两种情况

func TestSafeString(t *testing.T) {
	tests := []struct {
		name  string
		input *string
		want  string
	}{
		{name: "非nil指针", input: String("hello"), want: "hello"},
		{name: "空字符串指针", input: String(""), want: ""},
		{name: "nil指针", input: nil, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeString(tt.input); got != tt.want {
				t.Errorf("SafeString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeInt(t *testing.T) {
	tests := []struct {
		name  string
		input *int
		want  int
	}{
		{name: "非nil指针", input: Int(42), want: 42},
		{name: "零值指针", input: Int(0), want: 0},
		{name: "负值指针", input: Int(-10), want: -10},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeInt(tt.input); got != tt.want {
				t.Errorf("SafeInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeInt8(t *testing.T) {
	tests := []struct {
		name  string
		input *int8
		want  int8
	}{
		{name: "非nil指针", input: Int8(42), want: 42},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeInt8(tt.input); got != tt.want {
				t.Errorf("SafeInt8() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeInt16(t *testing.T) {
	tests := []struct {
		name  string
		input *int16
		want  int16
	}{
		{name: "非nil指针", input: Int16(42), want: 42},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeInt16(tt.input); got != tt.want {
				t.Errorf("SafeInt16() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeInt32(t *testing.T) {
	tests := []struct {
		name  string
		input *int32
		want  int32
	}{
		{name: "非nil指针", input: Int32(42), want: 42},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeInt32(tt.input); got != tt.want {
				t.Errorf("SafeInt32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeInt64(t *testing.T) {
	tests := []struct {
		name  string
		input *int64
		want  int64
	}{
		{name: "非nil指针", input: Int64(42), want: 42},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeInt64(tt.input); got != tt.want {
				t.Errorf("SafeInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeUint(t *testing.T) {
	tests := []struct {
		name  string
		input *uint
		want  uint
	}{
		{name: "非nil指针", input: Uint(42), want: 42},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeUint(tt.input); got != tt.want {
				t.Errorf("SafeUint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeUint8(t *testing.T) {
	tests := []struct {
		name  string
		input *uint8
		want  uint8
	}{
		{name: "非nil指针", input: Uint8(42), want: 42},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeUint8(tt.input); got != tt.want {
				t.Errorf("SafeUint8() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeUint16(t *testing.T) {
	tests := []struct {
		name  string
		input *uint16
		want  uint16
	}{
		{name: "非nil指针", input: Uint16(42), want: 42},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeUint16(tt.input); got != tt.want {
				t.Errorf("SafeUint16() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeUint32(t *testing.T) {
	tests := []struct {
		name  string
		input *uint32
		want  uint32
	}{
		{name: "非nil指针", input: Uint32(42), want: 42},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeUint32(tt.input); got != tt.want {
				t.Errorf("SafeUint32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeUint64(t *testing.T) {
	tests := []struct {
		name  string
		input *uint64
		want  uint64
	}{
		{name: "非nil指针", input: Uint64(42), want: 42},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeUint64(tt.input); got != tt.want {
				t.Errorf("SafeUint64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeFloat32(t *testing.T) {
	tests := []struct {
		name  string
		input *float32
		want  float32
	}{
		{name: "非nil指针", input: Float32(3.14), want: 3.14},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeFloat32(tt.input); got != tt.want {
				t.Errorf("SafeFloat32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeFloat64(t *testing.T) {
	tests := []struct {
		name  string
		input *float64
		want  float64
	}{
		{name: "非nil指针", input: Float64(3.14159), want: 3.14159},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeFloat64(tt.input); got != tt.want {
				t.Errorf("SafeFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeBool(t *testing.T) {
	tests := []struct {
		name  string
		input *bool
		want  bool
	}{
		{name: "true指针", input: Bool(true), want: true},
		{name: "false指针", input: Bool(false), want: false},
		{name: "nil指针", input: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeBool(tt.input); got != tt.want {
				t.Errorf("SafeBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeByte(t *testing.T) {
	tests := []struct {
		name  string
		input *byte
		want  byte
	}{
		{name: "非nil指针", input: Byte('A'), want: 'A'},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeByte(tt.input); got != tt.want {
				t.Errorf("SafeByte() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeRune(t *testing.T) {
	tests := []struct {
		name  string
		input *rune
		want  rune
	}{
		{name: "非nil指针", input: Rune('中'), want: '中'},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeRune(tt.input); got != tt.want {
				t.Errorf("SafeRune() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeComplex64(t *testing.T) {
	tests := []struct {
		name  string
		input *complex64
		want  complex64
	}{
		{name: "非nil指针", input: Complex64(1 + 2i), want: 1 + 2i},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeComplex64(tt.input); got != tt.want {
				t.Errorf("SafeComplex64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeComplex128(t *testing.T) {
	tests := []struct {
		name  string
		input *complex128
		want  complex128
	}{
		{name: "非nil指针", input: Complex128(1 + 2i), want: 1 + 2i},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeComplex128(tt.input); got != tt.want {
				t.Errorf("SafeComplex128() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		input *time.Time
		want  time.Time
	}{
		{name: "非nil指针", input: Time(now), want: now},
		{name: "nil指针", input: nil, want: time.Time{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeTime(tt.input); !got.Equal(tt.want) {
				t.Errorf("SafeTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeDuration(t *testing.T) {
	tests := []struct {
		name  string
		input *time.Duration
		want  time.Duration
	}{
		{name: "非nil指针", input: Duration(time.Hour), want: time.Hour},
		{name: "nil指针", input: nil, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeDuration(tt.input); got != tt.want {
				t.Errorf("SafeDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Benchmark测试
func BenchmarkSafeString(b *testing.B) {
	s := "hello"
	for i := 0; i < b.N; i++ {
		_ = SafeString(&s)
	}
}

func BenchmarkSafeStringNil(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = SafeString(nil)
	}
}

func BenchmarkSafeInt(b *testing.B) {
	i := 42
	for n := 0; n < b.N; n++ {
		_ = SafeInt(&i)
	}
}

func BenchmarkSafeIntNil(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_ = SafeInt(nil)
	}
}

func BenchmarkSafeBool(b *testing.B) {
	v := true
	for i := 0; i < b.N; i++ {
		_ = SafeBool(&v)
	}
}

func BenchmarkSafeFloat64(b *testing.B) {
	f := 3.14
	for i := 0; i < b.N; i++ {
		_ = SafeFloat64(&f)
	}
}

func BenchmarkSafeTime(b *testing.B) {
	t := time.Now()
	for i := 0; i < b.N; i++ {
		_ = SafeTime(&t)
	}
}
