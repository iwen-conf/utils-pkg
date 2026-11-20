package slice

import (
	"cmp"
	"slices"
)

// Contains 检查切片中是否包含指定的元素。
func Contains[T comparable](s []T, element T) bool {
	return slices.Contains(s, element)
}

// Unique 返回保持首次出现顺序的去重副本（始终分配新切片，不与原切片共享底层数组）。
func Unique[T comparable](s []T) []T {
	if len(s) == 0 {
		return []T{}
	}

	result := make([]T, 0, len(s))
	seen := make(map[T]struct{}, len(s))
	for _, v := range s {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// Intersection 返回两个切片的去重交集。
func Intersection[T comparable](slice1, slice2 []T) []T {
	if len(slice1) == 0 || len(slice2) == 0 {
		return []T{}
	}

	smallerSlice, largerSlice := slice1, slice2
	if len(slice1) > len(slice2) {
		smallerSlice, largerSlice = slice2, slice1
	}

	result := make([]T, 0, len(smallerSlice))
	set := make(map[T]bool, len(smallerSlice))
	for _, v := range smallerSlice {
		set[v] = true
	}
	for _, v := range largerSlice {
		if set[v] {
			result = append(result, v)
			set[v] = false
		}
	}
	return result
}

// Union 返回两个切片的去重并集（始终返回独立切片并保证去重，即使某一输入为空）。
func Union[T comparable](slice1, slice2 []T) []T {
	if len(slice1) == 0 && len(slice2) == 0 {
		return []T{}
	}
	if len(slice1) == 0 {
		return Unique(slice2)
	}
	if len(slice2) == 0 {
		return Unique(slice1)
	}

	totalLen := len(slice1) + len(slice2)
	result := make([]T, 0, totalLen)
	seen := make(map[T]struct{}, totalLen)
	for _, v := range slice1 {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	for _, v := range slice2 {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// Difference 返回在 slice1 中但不在 slice2 中的元素副本（始终返回独立切片，不与输入共享底层数组）。
func Difference[T comparable](slice1, slice2 []T) []T {
	if len(slice1) == 0 {
		return []T{}
	}
	if len(slice2) == 0 {
		result := make([]T, len(slice1))
		copy(result, slice1)
		return result
	}

	result := make([]T, 0, len(slice1))
	exclude := make(map[T]struct{}, len(slice2))
	for _, v := range slice2 {
		exclude[v] = struct{}{}
	}
	for _, v := range slice1 {
		if _, found := exclude[v]; !found {
			result = append(result, v)
		}
	}
	return result
}

// Filter 根据条件过滤切片元素并保持原顺序。
func Filter[T any](s []T, predicate func(T) bool) []T {
	if len(s) == 0 {
		return []T{}
	}
	result := make([]T, 0, len(s))
	for _, v := range s {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// Map 对切片中的每个元素应用转换函数。
func Map[T any, R any](s []T, transform func(T) R) []R {
	if len(s) == 0 {
		return []R{}
	}
	result := make([]R, len(s))
	for i, v := range s {
		result[i] = transform(v)
	}
	return result
}

// Reduce 对切片中的元素进行归约操作。
func Reduce[T any, R any](s []T, initial R, reducer func(R, T) R) R {
	result := initial
	for _, v := range s {
		result = reducer(result, v)
	}
	return result
}

// Sort 返回升序排序后的独立副本（使用标准库 slices.Sort，正确处理浮点数 NaN 全序）。
func Sort[T cmp.Ordered](s []T) []T {
	if len(s) == 0 {
		return []T{}
	}
	result := make([]T, len(s))
	copy(result, s)
	slices.Sort(result)
	return result
}
