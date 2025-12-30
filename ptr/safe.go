package ptr

import "time"

// SafeString 安全地解引用字符串指针。
// 如果指针为nil，返回空字符串""。
// 如果指针不为nil，返回解引用后的值。
func SafeString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// SafeInt 安全地解引用int指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeInt(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeInt8 安全地解引用int8指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeInt8(ptr *int8) int8 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeInt16 安全地解引用int16指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeInt16(ptr *int16) int16 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeInt32 安全地解引用int32指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeInt32(ptr *int32) int32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeInt64 安全地解引用int64指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeUint 安全地解引用uint指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeUint(ptr *uint) uint {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeUint8 安全地解引用uint8指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeUint8(ptr *uint8) uint8 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeUint16 安全地解引用uint16指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeUint16(ptr *uint16) uint16 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeUint32 安全地解引用uint32指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeUint32(ptr *uint32) uint32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeUint64 安全地解引用uint64指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeUint64(ptr *uint64) uint64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeFloat32 安全地解引用float32指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeFloat32(ptr *float32) float32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeFloat64 安全地解引用float64指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeFloat64(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeBool 安全地解引用bool指针。
// 如果指针为nil，返回false。
// 如果指针不为nil，返回解引用后的值。
func SafeBool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

// SafeByte 安全地解引用byte指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeByte(ptr *byte) byte {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeRune 安全地解引用rune指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeRune(ptr *rune) rune {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeComplex64 安全地解引用complex64指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeComplex64(ptr *complex64) complex64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeComplex128 安全地解引用complex128指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeComplex128(ptr *complex128) complex128 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// SafeTime 安全地解引用time.Time指针。
// 如果指针为nil，返回零时间(time.Time{})。
// 如果指针不为nil，返回解引用后的值。
func SafeTime(ptr *time.Time) time.Time {
	if ptr == nil {
		return time.Time{}
	}
	return *ptr
}

// SafeDuration 安全地解引用time.Duration指针。
// 如果指针为nil，返回0。
// 如果指针不为nil，返回解引用后的值。
func SafeDuration(ptr *time.Duration) time.Duration {
	if ptr == nil {
		return 0
	}
	return *ptr
}
