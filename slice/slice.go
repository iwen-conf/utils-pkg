package slice

import (
	"runtime"
	"sort"
	"sync"

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

// containsParallel 并行检查大型切片中是否包含指定的元素
func containsParallel[T comparable](slice []T, element T) bool {
	// 对于小型切片，直接使用非并行版本
	if len(slice) < 10000 {
		return Contains(slice, element)
	}

	cpus := runtime.NumCPU()
	var wg sync.WaitGroup
	chunkSize := (len(slice) + cpus - 1) / cpus
	found := make(chan bool, 1)
	done := make(chan struct{})

	for i := 0; i < cpus; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			end := start + chunkSize
			if end > len(slice) {
				end = len(slice)
			}

			for j := start; j < end; j++ {
				if slice[j] == element {
					select {
					case found <- true:
					default:
					}
					return
				}
			}
		}(i * chunkSize)
	}

	// 等待所有goroutine完成或找到元素
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-found:
		return true
	case <-done:
		return false
	}
}

// Unique 返回切片的去重结果
func Unique[T comparable](slice []T) []T {
	if len(slice) == 0 {
		return []T{}
	}

	// 预分配合适的容量可以减少内存重分配
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

	// 使用较小的切片建立map可以减少内存使用
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
			// 确保结果中不包含重复元素
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

	// 直接使用map去重而不是先合并再去重
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

	// 预分配可能的最大容量
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// filterParallel 并行过滤大型切片
func filterParallel[T any](slice []T, predicate func(T) bool) []T {
	// 对于小型切片，直接使用非并行版本
	if len(slice) < 10000 {
		return Filter(slice, predicate)
	}

	cpus := runtime.NumCPU()
	chunkSize := (len(slice) + cpus - 1) / cpus
	var wg sync.WaitGroup
	var mu sync.Mutex
	result := make([]T, 0, len(slice)/2) // 估计结果大小

	for i := 0; i < cpus; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			localResult := make([]T, 0, chunkSize/2)

			end := start + chunkSize
			if end > len(slice) {
				end = len(slice)
			}

			for j := start; j < end; j++ {
				if predicate(slice[j]) {
					localResult = append(localResult, slice[j])
				}
			}

			// 合并结果
			mu.Lock()
			result = append(result, localResult...)
			mu.Unlock()
		}(i * chunkSize)
	}

	wg.Wait()
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

// mapParallel 并行映射大型切片
func mapParallel[T any, R any](slice []T, transform func(T) R) []R {
	// 对于小型切片，直接使用非并行版本
	if len(slice) < 10000 {
		return Map(slice, transform)
	}

	cpus := runtime.NumCPU()
	chunkSize := (len(slice) + cpus - 1) / cpus
	result := make([]R, len(slice))
	var wg sync.WaitGroup

	for i := 0; i < cpus; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			end := start + chunkSize
			if end > len(slice) {
				end = len(slice)
			}

			for j := start; j < end; j++ {
				result[j] = transform(slice[j])
			}
		}(i * chunkSize)
	}

	wg.Wait()
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

// Sort 对切片进行排序 - 使用标准库代替冒泡排序
func Sort[T constraints.Ordered](slice []T) []T {
	result := make([]T, len(slice))
	copy(result, slice)

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result
}

// LRUCache 是一个简单的LRU缓存实现，用于缓存常用操作结果
type LRUCache[K comparable, V any] struct {
	capacity int
	cache    map[K]V
	keys     []K
	mu       sync.RWMutex
}

// NewLRUCache 创建一个新的LRU缓存
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		cache:    make(map[K]V, capacity),
		keys:     make([]K, 0, capacity),
	}
}

// Get 从缓存中获取值
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	val, ok := c.cache[key]
	c.mu.RUnlock()

	if ok {
		c.mu.Lock()
		// 将key移到队列末尾（最近使用）
		c.moveToEnd(key)
		c.mu.Unlock()
	}

	return val, ok
}

// Put 将值放入缓存
func (c *LRUCache[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.cache[key]; ok {
		c.cache[key] = value
		c.moveToEnd(key)
		return
	}

	if len(c.cache) >= c.capacity {
		// 移除最不常用的元素
		delete(c.cache, c.keys[0])
		c.keys = c.keys[1:]
	}

	c.cache[key] = value
	c.keys = append(c.keys, key)
}

// moveToEnd 将key移动到keys切片的末尾
func (c *LRUCache[K, V]) moveToEnd(key K) {
	for i, k := range c.keys {
		if k == key {
			// 从当前位置删除
			c.keys = append(c.keys[:i], c.keys[i+1:]...)
			// 添加到末尾
			c.keys = append(c.keys, key)
			break
		}
	}
}
