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

func TestValueString(t *testing.T) {
	s := "hello"
	val, err := ValueString(&s)
	if err != nil {
		t.Errorf("ValueString() 返回了意外错误: %v", err)
	}
	if val != s {
		t.Errorf("ValueString() = %v, 期望 %v", val, s)
	}

	var nilPtr *string
	val, err = ValueString(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueString() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != "" {
		t.Errorf("ValueString() = %v, 期望空字符串", val)
	}
}

func TestValueInt(t *testing.T) {
	i := 42
	val, err := ValueInt(&i)
	if err != nil {
		t.Errorf("ValueInt() 返回了意外错误: %v", err)
	}
	if val != i {
		t.Errorf("ValueInt() = %v, 期望 %v", val, i)
	}

	var nilPtr *int
	val, err = ValueInt(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueInt() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueInt() = %v, 期望 0", val)
	}
}

func TestValueInt8(t *testing.T) {
	i := int8(42)
	val, err := ValueInt8(&i)
	if err != nil {
		t.Errorf("ValueInt8() 返回了意外错误: %v", err)
	}
	if val != i {
		t.Errorf("ValueInt8() = %v, 期望 %v", val, i)
	}

	var nilPtr *int8
	val, err = ValueInt8(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueInt8() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueInt8() = %v, 期望 0", val)
	}
}

func TestValueInt16(t *testing.T) {
	i := int16(42)
	val, err := ValueInt16(&i)
	if err != nil {
		t.Errorf("ValueInt16() 返回了意外错误: %v", err)
	}
	if val != i {
		t.Errorf("ValueInt16() = %v, 期望 %v", val, i)
	}

	var nilPtr *int16
	val, err = ValueInt16(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueInt16() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueInt16() = %v, 期望 0", val)
	}
}

func TestValueInt32(t *testing.T) {
	i := int32(42)
	val, err := ValueInt32(&i)
	if err != nil {
		t.Errorf("ValueInt32() 返回了意外错误: %v", err)
	}
	if val != i {
		t.Errorf("ValueInt32() = %v, 期望 %v", val, i)
	}

	var nilPtr *int32
	val, err = ValueInt32(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueInt32() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueInt32() = %v, 期望 0", val)
	}
}

func TestValueInt64(t *testing.T) {
	i := int64(42)
	val, err := ValueInt64(&i)
	if err != nil {
		t.Errorf("ValueInt64() 返回了意外错误: %v", err)
	}
	if val != i {
		t.Errorf("ValueInt64() = %v, 期望 %v", val, i)
	}

	var nilPtr *int64
	val, err = ValueInt64(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueInt64() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueInt64() = %v, 期望 0", val)
	}
}

func TestValueUint(t *testing.T) {
	u := uint(42)
	val, err := ValueUint(&u)
	if err != nil {
		t.Errorf("ValueUint() 返回了意外错误: %v", err)
	}
	if val != u {
		t.Errorf("ValueUint() = %v, 期望 %v", val, u)
	}

	var nilPtr *uint
	val, err = ValueUint(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueUint() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueUint() = %v, 期望 0", val)
	}
}

func TestValueUint8(t *testing.T) {
	u := uint8(42)
	val, err := ValueUint8(&u)
	if err != nil {
		t.Errorf("ValueUint8() 返回了意外错误: %v", err)
	}
	if val != u {
		t.Errorf("ValueUint8() = %v, 期望 %v", val, u)
	}

	var nilPtr *uint8
	val, err = ValueUint8(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueUint8() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueUint8() = %v, 期望 0", val)
	}
}

func TestValueUint16(t *testing.T) {
	u := uint16(42)
	val, err := ValueUint16(&u)
	if err != nil {
		t.Errorf("ValueUint16() 返回了意外错误: %v", err)
	}
	if val != u {
		t.Errorf("ValueUint16() = %v, 期望 %v", val, u)
	}

	var nilPtr *uint16
	val, err = ValueUint16(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueUint16() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueUint16() = %v, 期望 0", val)
	}
}

func TestValueUint32(t *testing.T) {
	u := uint32(42)
	val, err := ValueUint32(&u)
	if err != nil {
		t.Errorf("ValueUint32() 返回了意外错误: %v", err)
	}
	if val != u {
		t.Errorf("ValueUint32() = %v, 期望 %v", val, u)
	}

	var nilPtr *uint32
	val, err = ValueUint32(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueUint32() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueUint32() = %v, 期望 0", val)
	}
}

func TestValueUint64(t *testing.T) {
	u := uint64(42)
	val, err := ValueUint64(&u)
	if err != nil {
		t.Errorf("ValueUint64() 返回了意外错误: %v", err)
	}
	if val != u {
		t.Errorf("ValueUint64() = %v, 期望 %v", val, u)
	}

	var nilPtr *uint64
	val, err = ValueUint64(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueUint64() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueUint64() = %v, 期望 0", val)
	}
}

func TestValueFloat32(t *testing.T) {
	f := float32(3.14)
	val, err := ValueFloat32(&f)
	if err != nil {
		t.Errorf("ValueFloat32() 返回了意外错误: %v", err)
	}
	if val != f {
		t.Errorf("ValueFloat32() = %v, 期望 %v", val, f)
	}

	var nilPtr *float32
	val, err = ValueFloat32(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueFloat32() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueFloat32() = %v, 期望 0", val)
	}
}

func TestValueFloat64(t *testing.T) {
	f := float64(3.14)
	val, err := ValueFloat64(&f)
	if err != nil {
		t.Errorf("ValueFloat64() 返回了意外错误: %v", err)
	}
	if val != f {
		t.Errorf("ValueFloat64() = %v, 期望 %v", val, f)
	}

	var nilPtr *float64
	val, err = ValueFloat64(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueFloat64() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueFloat64() = %v, 期望 0", val)
	}
}

func TestValueBool(t *testing.T) {
	b := true
	val, err := ValueBool(&b)
	if err != nil {
		t.Errorf("ValueBool() 返回了意外错误: %v", err)
	}
	if val != b {
		t.Errorf("ValueBool() = %v, 期望 %v", val, b)
	}

	var nilPtr *bool
	val, err = ValueBool(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueBool() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != false {
		t.Errorf("ValueBool() = %v, 期望 false", val)
	}
}

func TestValueByte(t *testing.T) {
	b := byte(42)
	val, err := ValueByte(&b)
	if err != nil {
		t.Errorf("ValueByte() 返回了意外错误: %v", err)
	}
	if val != b {
		t.Errorf("ValueByte() = %v, 期望 %v", val, b)
	}

	var nilPtr *byte
	val, err = ValueByte(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueByte() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueByte() = %v, 期望 0", val)
	}
}

func TestValueRune(t *testing.T) {
	r := rune('A')
	val, err := ValueRune(&r)
	if err != nil {
		t.Errorf("ValueRune() 返回了意外错误: %v", err)
	}
	if val != r {
		t.Errorf("ValueRune() = %v, 期望 %v", val, r)
	}

	var nilPtr *rune
	val, err = ValueRune(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueRune() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueRune() = %v, 期望 0", val)
	}
}

func TestValueComplex64(t *testing.T) {
	c := complex64(3 + 4i)
	val, err := ValueComplex64(&c)
	if err != nil {
		t.Errorf("ValueComplex64() 返回了意外错误: %v", err)
	}
	if val != c {
		t.Errorf("ValueComplex64() = %v, 期望 %v", val, c)
	}

	var nilPtr *complex64
	val, err = ValueComplex64(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueComplex64() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueComplex64() = %v, 期望 0", val)
	}
}

func TestValueComplex128(t *testing.T) {
	c := complex128(3 + 4i)
	val, err := ValueComplex128(&c)
	if err != nil {
		t.Errorf("ValueComplex128() 返回了意外错误: %v", err)
	}
	if val != c {
		t.Errorf("ValueComplex128() = %v, 期望 %v", val, c)
	}

	var nilPtr *complex128
	val, err = ValueComplex128(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueComplex128() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueComplex128() = %v, 期望 0", val)
	}
}

func TestValueTime(t *testing.T) {
	now := time.Now()
	val, err := ValueTime(&now)
	if err != nil {
		t.Errorf("ValueTime() 返回了意外错误: %v", err)
	}
	if val != now {
		t.Errorf("ValueTime() = %v, 期望 %v", val, now)
	}

	var nilPtr *time.Time
	val, err = ValueTime(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueTime() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if !val.IsZero() {
		t.Errorf("ValueTime() = %v, 期望零时间", val)
	}
}

func TestValueDuration(t *testing.T) {
	d := time.Hour
	val, err := ValueDuration(&d)
	if err != nil {
		t.Errorf("ValueDuration() 返回了意外错误: %v", err)
	}
	if val != d {
		t.Errorf("ValueDuration() = %v, 期望 %v", val, d)
	}

	var nilPtr *time.Duration
	val, err = ValueDuration(nilPtr)
	if err != ErrNilPointer {
		t.Errorf("ValueDuration() 错误 = %v, 期望 %v", err, ErrNilPointer)
	}
	if val != 0 {
		t.Errorf("ValueDuration() = %v, 期望 0", val)
	}
}
