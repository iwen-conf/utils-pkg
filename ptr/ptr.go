package ptr

import (
	"errors"
	"time"
)

// ErrNilPointer 当尝试解引用空指针时返回。
var ErrNilPointer = errors.New("指针为空")

// String 返回给定字符串值的指针。
func String(s string) *string { return Of(s) }

// Int 返回给定int值的指针。
func Int(i int) *int { return Of(i) }

// Int8 返回给定int8值的指针。
func Int8(i int8) *int8 { return Of(i) }

// Int16 返回给定int16值的指针。
func Int16(i int16) *int16 { return Of(i) }

// Int32 返回给定int32值的指针。
func Int32(i int32) *int32 { return Of(i) }

// Int64 返回给定int64值的指针。
func Int64(i int64) *int64 { return Of(i) }

// Uint 返回给定uint值的指针。
func Uint(u uint) *uint { return Of(u) }

// Uint8 返回给定uint8值的指针。
func Uint8(u uint8) *uint8 { return Of(u) }

// Uint16 返回给定uint16值的指针。
func Uint16(u uint16) *uint16 { return Of(u) }

// Uint32 返回给定uint32值的指针。
func Uint32(u uint32) *uint32 { return Of(u) }

// Uint64 返回给定uint64值的指针。
func Uint64(u uint64) *uint64 { return Of(u) }

// Float32 返回给定float32值的指针。
func Float32(f float32) *float32 { return Of(f) }

// Float64 返回给定float64值的指针。
func Float64(f float64) *float64 { return Of(f) }

// Bool 返回给定bool值的指针。
func Bool(b bool) *bool { return Of(b) }

// Byte 返回给定byte值的指针。
func Byte(b byte) *byte { return Of(b) }

// Rune 返回给定rune值的指针。
func Rune(r rune) *rune { return Of(r) }

// Complex64 返回给定complex64值的指针。
func Complex64(c complex64) *complex64 { return Of(c) }

// Complex128 返回给定complex128值的指针。
func Complex128(c complex128) *complex128 { return Of(c) }

// Time 返回给定time.Time值的指针。
func Time(t time.Time) *time.Time { return Of(t) }

// Duration 返回给定time.Duration值的指针。
func Duration(d time.Duration) *time.Duration { return Of(d) }
