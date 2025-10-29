package ptr

import (
	"testing"
	"time"
)

// 这个文件包含所有 Value* 函数的测试
// 每个函数都测试 nil 和非 nil 两种情况

func BenchmarkValueString(b *testing.B) {
	s := "hello"
	for i := 0; i < b.N; i++ {
		_, _ = ValueString(&s)
	}
}

func BenchmarkValueInt(b *testing.B) {
	i := 42
	for n := 0; n < b.N; n++ {
		_, _ = ValueInt(&i)
	}
}

func BenchmarkValueBool(b *testing.B) {
	v := true
	for i := 0; i < b.N; i++ {
		_, _ = ValueBool(&v)
	}
}

func BenchmarkValueFloat64(b *testing.B) {
	f := 3.14
	for i := 0; i < b.N; i++ {
		_, _ = ValueFloat64(&f)
	}
}

func BenchmarkValueTime(b *testing.B) {
	t := time.Now()
	for i := 0; i < b.N; i++ {
		_, _ = ValueTime(&t)
	}
}

// 表格驱动测试示例
func TestValueIntWithTable(t *testing.T) {
	tests := []struct {
		name    string
		input   *int
		want    int
		wantErr bool
	}{
		{
			name:    "非nil指针",
			input:   Int(42),
			want:    42,
			wantErr: false,
		},
		{
			name:    "零值指针",
			input:   Int(0),
			want:    0,
			wantErr: false,
		},
		{
			name:    "负值指针",
			input:   Int(-10),
			want:    -10,
			wantErr: false,
		},
		{
			name:    "nil指针",
			input:   nil,
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValueInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValueInt() 错误 = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValueInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
