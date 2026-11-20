package ptr

import "time"

// ValueString 安全地解引用字符串指针。
func ValueString(p *string) (string, error) { return Value(p) }

// ValueInt 安全地解引用int指针。
func ValueInt(p *int) (int, error) { return Value(p) }

// ValueInt8 安全地解引用int8指针。
func ValueInt8(p *int8) (int8, error) { return Value(p) }

// ValueInt16 安全地解引用int16指针。
func ValueInt16(p *int16) (int16, error) { return Value(p) }

// ValueInt32 安全地解引用int32指针。
func ValueInt32(p *int32) (int32, error) { return Value(p) }

// ValueInt64 安全地解引用int64指针。
func ValueInt64(p *int64) (int64, error) { return Value(p) }

// ValueUint 安全地解引用uint指针。
func ValueUint(p *uint) (uint, error) { return Value(p) }

// ValueUint8 安全地解引用uint8指针。
func ValueUint8(p *uint8) (uint8, error) { return Value(p) }

// ValueUint16 安全地解引用uint16指针。
func ValueUint16(p *uint16) (uint16, error) { return Value(p) }

// ValueUint32 安全地解引用uint32指针。
func ValueUint32(p *uint32) (uint32, error) { return Value(p) }

// ValueUint64 安全地解引用uint64指针。
func ValueUint64(p *uint64) (uint64, error) { return Value(p) }

// ValueFloat32 安全地解引用float32指针。
func ValueFloat32(p *float32) (float32, error) { return Value(p) }

// ValueFloat64 安全地解引用float64指针。
func ValueFloat64(p *float64) (float64, error) { return Value(p) }

// ValueBool 安全地解引用bool指针。
func ValueBool(p *bool) (bool, error) { return Value(p) }

// ValueByte 安全地解引用byte指针。
func ValueByte(p *byte) (byte, error) { return Value(p) }

// ValueRune 安全地解引用rune指针。
func ValueRune(p *rune) (rune, error) { return Value(p) }

// ValueComplex64 安全地解引用complex64指针。
func ValueComplex64(p *complex64) (complex64, error) { return Value(p) }

// ValueComplex128 安全地解引用complex128指针。
func ValueComplex128(p *complex128) (complex128, error) { return Value(p) }

// ValueTime 安全地解引用time.Time指针。
func ValueTime(p *time.Time) (time.Time, error) { return Value(p) }

// ValueDuration 安全地解引用time.Duration指针。
func ValueDuration(p *time.Duration) (time.Duration, error) { return Value(p) }
