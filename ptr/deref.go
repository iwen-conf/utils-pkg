package ptr

import "time"

// ValueString 安全地解引用字符串指针。
// 如果指针为nil，返回空字符串和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueString(ptr *string) (string, error) {
	if ptr == nil {
		return "", ErrNilPointer
	}
	return *ptr, nil
}

// ValueInt 安全地解引用int指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueInt(ptr *int) (int, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueInt8 安全地解引用int8指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueInt8(ptr *int8) (int8, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueInt16 安全地解引用int16指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueInt16(ptr *int16) (int16, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueInt32 安全地解引用int32指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueInt32(ptr *int32) (int32, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueInt64 安全地解引用int64指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueInt64(ptr *int64) (int64, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueUint 安全地解引用uint指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueUint(ptr *uint) (uint, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueUint8 安全地解引用uint8指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueUint8(ptr *uint8) (uint8, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueUint16 安全地解引用uint16指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueUint16(ptr *uint16) (uint16, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueUint32 安全地解引用uint32指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueUint32(ptr *uint32) (uint32, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueUint64 安全地解引用uint64指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueUint64(ptr *uint64) (uint64, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueFloat32 安全地解引用float32指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueFloat32(ptr *float32) (float32, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueFloat64 安全地解引用float64指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueFloat64(ptr *float64) (float64, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueBool 安全地解引用bool指针。
// 如果指针为nil，返回false和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueBool(ptr *bool) (bool, error) {
	if ptr == nil {
		return false, ErrNilPointer
	}
	return *ptr, nil
}

// ValueByte 安全地解引用byte指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueByte(ptr *byte) (byte, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueRune 安全地解引用rune指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueRune(ptr *rune) (rune, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueComplex64 安全地解引用complex64指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueComplex64(ptr *complex64) (complex64, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueComplex128 安全地解引用complex128指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueComplex128(ptr *complex128) (complex128, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}

// ValueTime 安全地解引用time.Time指针。
// 如果指针为nil，返回零时间和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueTime(ptr *time.Time) (time.Time, error) {
	if ptr == nil {
		return time.Time{}, ErrNilPointer
	}
	return *ptr, nil
}

// ValueDuration 安全地解引用time.Duration指针。
// 如果指针为nil，返回0和ErrNilPointer错误。
// 如果指针不为nil，返回解引用后的值和nil错误。
func ValueDuration(ptr *time.Duration) (time.Duration, error) {
	if ptr == nil {
		return 0, ErrNilPointer
	}
	return *ptr, nil
}
