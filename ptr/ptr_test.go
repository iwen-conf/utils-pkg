package ptr

import (
	"testing"
	"time"
)

func TestString(t *testing.T) {
	s := "test"
	ptr := String(s)
	if ptr == nil {
		t.Error("String() 返回了空指针")
	}
	if *ptr != s {
		t.Errorf("String() = %v, 期望 %v", *ptr, s)
	}
}

func TestInt(t *testing.T) {
	i := 42
	ptr := Int(i)
	if ptr == nil {
		t.Error("Int() 返回了空指针")
	}
	if *ptr != i {
		t.Errorf("Int() = %v, 期望 %v", *ptr, i)
	}
}

func TestInt8(t *testing.T) {
	i := int8(42)
	ptr := Int8(i)
	if ptr == nil {
		t.Error("Int8() 返回了空指针")
	}
	if *ptr != i {
		t.Errorf("Int8() = %v, 期望 %v", *ptr, i)
	}
}

func TestInt16(t *testing.T) {
	i := int16(42)
	ptr := Int16(i)
	if ptr == nil {
		t.Error("Int16() 返回了空指针")
	}
	if *ptr != i {
		t.Errorf("Int16() = %v, 期望 %v", *ptr, i)
	}
}

func TestInt32(t *testing.T) {
	i := int32(42)
	ptr := Int32(i)
	if ptr == nil {
		t.Error("Int32() 返回了空指针")
	}
	if *ptr != i {
		t.Errorf("Int32() = %v, 期望 %v", *ptr, i)
	}
}

func TestInt64(t *testing.T) {
	i := int64(42)
	ptr := Int64(i)
	if ptr == nil {
		t.Error("Int64() 返回了空指针")
	}
	if *ptr != i {
		t.Errorf("Int64() = %v, 期望 %v", *ptr, i)
	}
}

func TestUint(t *testing.T) {
	u := uint(42)
	ptr := Uint(u)
	if ptr == nil {
		t.Error("Uint() 返回了空指针")
	}
	if *ptr != u {
		t.Errorf("Uint() = %v, 期望 %v", *ptr, u)
	}
}

func TestUint8(t *testing.T) {
	u := uint8(42)
	ptr := Uint8(u)
	if ptr == nil {
		t.Error("Uint8() 返回了空指针")
	}
	if *ptr != u {
		t.Errorf("Uint8() = %v, 期望 %v", *ptr, u)
	}
}

func TestUint16(t *testing.T) {
	u := uint16(42)
	ptr := Uint16(u)
	if ptr == nil {
		t.Error("Uint16() 返回了空指针")
	}
	if *ptr != u {
		t.Errorf("Uint16() = %v, 期望 %v", *ptr, u)
	}
}

func TestUint32(t *testing.T) {
	u := uint32(42)
	ptr := Uint32(u)
	if ptr == nil {
		t.Error("Uint32() 返回了空指针")
	}
	if *ptr != u {
		t.Errorf("Uint32() = %v, 期望 %v", *ptr, u)
	}
}

func TestUint64(t *testing.T) {
	u := uint64(42)
	ptr := Uint64(u)
	if ptr == nil {
		t.Error("Uint64() 返回了空指针")
	}
	if *ptr != u {
		t.Errorf("Uint64() = %v, 期望 %v", *ptr, u)
	}
}

func TestFloat32(t *testing.T) {
	f := float32(3.14)
	ptr := Float32(f)
	if ptr == nil {
		t.Error("Float32() 返回了空指针")
	}
	if *ptr != f {
		t.Errorf("Float32() = %v, 期望 %v", *ptr, f)
	}
}

func TestFloat64(t *testing.T) {
	f := float64(3.14)
	ptr := Float64(f)
	if ptr == nil {
		t.Error("Float64() 返回了空指针")
	}
	if *ptr != f {
		t.Errorf("Float64() = %v, 期望 %v", *ptr, f)
	}
}

func TestBool(t *testing.T) {
	b := true
	ptr := Bool(b)
	if ptr == nil {
		t.Error("Bool() 返回了空指针")
	}
	if *ptr != b {
		t.Errorf("Bool() = %v, 期望 %v", *ptr, b)
	}
}

func TestByte(t *testing.T) {
	b := byte(42)
	ptr := Byte(b)
	if ptr == nil {
		t.Error("Byte() 返回了空指针")
	}
	if *ptr != b {
		t.Errorf("Byte() = %v, 期望 %v", *ptr, b)
	}
}

func TestRune(t *testing.T) {
	r := rune('A')
	ptr := Rune(r)
	if ptr == nil {
		t.Error("Rune() 返回了空指针")
	}
	if *ptr != r {
		t.Errorf("Rune() = %v, 期望 %v", *ptr, r)
	}
}

func TestComplex64(t *testing.T) {
	c := complex64(3 + 4i)
	ptr := Complex64(c)
	if ptr == nil {
		t.Error("Complex64() 返回了空指针")
	}
	if *ptr != c {
		t.Errorf("Complex64() = %v, 期望 %v", *ptr, c)
	}
}

func TestComplex128(t *testing.T) {
	c := complex128(3 + 4i)
	ptr := Complex128(c)
	if ptr == nil {
		t.Error("Complex128() 返回了空指针")
	}
	if *ptr != c {
		t.Errorf("Complex128() = %v, 期望 %v", *ptr, c)
	}
}

func TestTime(t *testing.T) {
	now := time.Now()
	ptr := Time(now)
	if ptr == nil {
		t.Error("Time() 返回了空指针")
	}
	if *ptr != now {
		t.Errorf("Time() = %v, 期望 %v", *ptr, now)
	}
}

func TestDuration(t *testing.T) {
	d := time.Hour
	ptr := Duration(d)
	if ptr == nil {
		t.Error("Duration() 返回了空指针")
	}
	if *ptr != d {
		t.Errorf("Duration() = %v, 期望 %v", *ptr, d)
	}
}
