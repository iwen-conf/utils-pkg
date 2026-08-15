package slice

import (
	"cmp"
	"sort"
)

// Contains 检查切片中是否包含指定的元素
func Contains[T comparable](slice []T, element T) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}

// Unique 返回切片的去重结果
func Unique[T comparable](slice []T) []T {
	if len(slice) == 0 {
		return []T{}
	}

	result := make([]T, 0, len(slice))
	existed := make(map[T]bool, len(slice))
	for _, v := range slice {
		if !existed[v] {
			result = append(result, v)
			existed[v] = true
		}
	}
	return result
}

// Intersection 返回两个切片的交集
func Intersection[T comparable](slice1, slice2 []T) []T {
	if len(slice1) == 0 || len(slice2) == 0 {
		return []T{}
	}

	var smallerSlice, largerSlice []T
	if len(slice1) < len(slice2) {
		smallerSlice, largerSlice = slice1, slice2
	} else {
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

// Union 返回两个切片的并集
func Union[T comparable](slice1, slice2 []T) []T {
	if len(slice1) == 0 {
		return slice2
	}
	if len(slice2) == 0 {
		return slice1
	}

	totalLen := len(slice1) + len(slice2)
	result := make([]T, 0, totalLen)
	set := make(map[T]bool, totalLen)
	for _, v := range slice1 {
		if !set[v] {
			result = append(result, v)
			set[v] = true
		}
	}
	for _, v := range slice2 {
		if !set[v] {
			result = append(result, v)
			set[v] = true
		}
	}
	return result
}

// Difference 返回两个切片的差集（在slice1中但不在slice2中的元素）
func Difference[T comparable](slice1, slice2 []T) []T {
	if len(slice1) == 0 {
		return []T{}
	}
	if len(slice2) == 0 {
		return slice1
	}

	result := make([]T, 0, len(slice1))
	set := make(map[T]bool, len(slice2))
	for _, v := range slice2 {
		set[v] = true
	}
	for _, v := range slice1 {
		if !set[v] {
			result = append(result, v)
		}
	}
	return result
}

// Filter 根据条件过滤切片元素
func Filter[T any](slice []T, predicate func(T) bool) []T {
	if len(slice) == 0 {
		return []T{}
	}
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// Map 对切片中的每个元素应用转换函数
func Map[T any, R any](slice []T, transform func(T) R) []R {
	if len(slice) == 0 {
		return []R{}
	}
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i] = transform(v)
	}
	return result
}

// Reduce 对切片中的元素进行归约操作
func Reduce[T any, R any](slice []T, initial R, reducer func(R, T) R) R {
	result := initial
	for _, v := range slice {
		result = reducer(result, v)
	}
	return result
}

// Sort 返回排序后的副本。
func Sort[T cmp.Ordered](slice []T) []T {
	result := make([]T, len(slice))
	copy(result, slice)
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}
