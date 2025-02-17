package slice

import (
	"golang.org/x/exp/constraints"
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
	result := make([]T, 0, len(slice))
	existed := make(map[T]bool)

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
	result := make([]T, 0)
	set := make(map[T]bool)

	for _, v := range slice1 {
		set[v] = true
	}

	for _, v := range slice2 {
		if set[v] {
			result = append(result, v)
		}
	}

	return Unique(result)
}

// Union 返回两个切片的并集
func Union[T comparable](slice1, slice2 []T) []T {
	result := make([]T, 0, len(slice1)+len(slice2))
	result = append(result, slice1...)
	result = append(result, slice2...)
	return Unique(result)
}

// Difference 返回两个切片的差集（在slice1中但不在slice2中的元素）
func Difference[T comparable](slice1, slice2 []T) []T {
	result := make([]T, 0)
	set := make(map[T]bool)

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
	result := make([]T, 0)
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// Map 对切片中的每个元素应用转换函数
func Map[T any, R any](slice []T, transform func(T) R) []R {
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

// Sort 对切片进行排序
func Sort[T constraints.Ordered](slice []T) []T {
	result := make([]T, len(slice))
	copy(result, slice)

	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			if result[j] > result[j+1] {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result
}