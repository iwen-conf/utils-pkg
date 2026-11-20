package ptr

import "time"

// SafeString 安全地解引用字符串指针，nil 时返回零值。
func SafeString(p *string) string { return Zero(p) }

// SafeInt 安全地解引用int指针，nil 时返回零值。
func SafeInt(p *int) int { return Zero(p) }

// SafeInt8 安全地解引用int8指针，nil 时返回零值。
func SafeInt8(p *int8) int8 { return Zero(p) }

// SafeInt16 安全地解引用int16指针，nil 时返回零值。
func SafeInt16(p *int16) int16 { return Zero(p) }

// SafeInt32 安全地解引用int32指针，nil 时返回零值。
func SafeInt32(p *int32) int32 { return Zero(p) }

// SafeInt64 安全地解引用int64指针，nil 时返回零值。
func SafeInt64(p *int64) int64 { return Zero(p) }

// SafeUint 安全地解引用uint指针，nil 时返回零值。
func SafeUint(p *uint) uint { return Zero(p) }

// SafeUint8 安全地解引用uint8指针，nil 时返回零值。
func SafeUint8(p *uint8) uint8 { return Zero(p) }

// SafeUint16 安全地解引用uint16指针，nil 时返回零值。
func SafeUint16(p *uint16) uint16 { return Zero(p) }

// SafeUint32 安全地解引用uint32指针，nil 时返回零值。
func SafeUint32(p *uint32) uint32 { return Zero(p) }

// SafeUint64 安全地解引用uint64指针，nil 时返回零值。
func SafeUint64(p *uint64) uint64 { return Zero(p) }

// SafeFloat32 安全地解引用float32指针，nil 时返回零值。
func SafeFloat32(p *float32) float32 { return Zero(p) }

// SafeFloat64 安全地解引用float64指针，nil 时返回零值。
func SafeFloat64(p *float64) float64 { return Zero(p) }

// SafeBool 安全地解引用bool指针，nil 时返回零值。
func SafeBool(p *bool) bool { return Zero(p) }

// SafeByte 安全地解引用byte指针，nil 时返回零值。
func SafeByte(p *byte) byte { return Zero(p) }

// SafeRune 安全地解引用rune指针，nil 时返回零值。
func SafeRune(p *rune) rune { return Zero(p) }

// SafeComplex64 安全地解引用complex64指针，nil 时返回零值。
func SafeComplex64(p *complex64) complex64 { return Zero(p) }

// SafeComplex128 安全地解引用complex128指针，nil 时返回零值。
func SafeComplex128(p *complex128) complex128 { return Zero(p) }

// SafeTime 安全地解引用time.Time指针，nil 时返回零值。
func SafeTime(p *time.Time) time.Time { return Zero(p) }

// SafeDuration 安全地解引用time.Duration指针，nil 时返回零值。
func SafeDuration(p *time.Duration) time.Duration { return Zero(p) }
