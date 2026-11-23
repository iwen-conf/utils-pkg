// Package ptr 提供泛型指针创建与安全解引用辅助函数。
package ptr

import "errors"

// ErrNilPointer 当尝试解引用空指针时返回。
var ErrNilPointer = errors.New("指针为空")

// Of 返回指向给定值副本的指针，适用于任意基本类型、自定义类型或结构体。
func Of[T any](v T) *T {
	return &v
}

// Value 安全解引用任意类型指针；若 p 为 nil，返回 T 的零值与 ErrNilPointer。
func Value[T any](p *T) (T, error) {
	if p == nil {
		var zero T
		return zero, ErrNilPointer
	}
	return *p, nil
}

// Deref 安全解引用指针；若 p 为 nil，返回调用方指定的默认值 fallback。
func Deref[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

// Zero 安全解引用指针；若 p 为 nil，返回类型 T 的零值。
func Zero[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
