// Package ptr 提供用于创建各种类型指针的工具函数。
// 当您需要向函数传递指针但只有值时，这非常有用。
package ptr

import (
	"errors"
	"time"
)

// ErrNilPointer 当尝试解引用空指针时返回。
var ErrNilPointer = errors.New("指针为空")

// String 返回给定字符串值的指针。
func String(s string) *string {
	return &s
}

// Int 返回给定int值的指针。
func Int(i int) *int {
	return &i
}

// Int8 返回给定int8值的指针。
func Int8(i int8) *int8 {
	return &i
}

// Int16 返回给定int16值的指针。
func Int16(i int16) *int16 {
	return &i
}

// Int32 返回给定int32值的指针。
func Int32(i int32) *int32 {
	return &i
}

// Int64 返回给定int64值的指针。
func Int64(i int64) *int64 {
	return &i
}

// Uint 返回给定uint值的指针。
func Uint(u uint) *uint {
	return &u
}

// Uint8 返回给定uint8值的指针。
func Uint8(u uint8) *uint8 {
	return &u
}

// Uint16 返回给定uint16值的指针。
func Uint16(u uint16) *uint16 {
	return &u
}

// Uint32 返回给定uint32值的指针。
func Uint32(u uint32) *uint32 {
	return &u
}

// Uint64 返回给定uint64值的指针。
func Uint64(u uint64) *uint64 {
	return &u
}

// Float32 返回给定float32值的指针。
func Float32(f float32) *float32 {
	return &f
}

// Float64 返回给定float64值的指针。
func Float64(f float64) *float64 {
	return &f
}

// Bool 返回给定bool值的指针。
func Bool(b bool) *bool {
	return &b
}

// Byte 返回给定byte值的指针。
func Byte(b byte) *byte {
	return &b
}

// Rune 返回给定rune值的指针。
func Rune(r rune) *rune {
	return &r
}

// Complex64 返回给定complex64值的指针。
func Complex64(c complex64) *complex64 {
	return &c
}

// Complex128 返回给定complex128值的指针。
func Complex128(c complex128) *complex128 {
	return &c
}

// Time 返回给定time.Time值的指针。
func Time(t time.Time) *time.Time {
	return &t
}

// Duration 返回给定time.Duration值的指针。
func Duration(d time.Duration) *time.Duration {
	return &d
}
