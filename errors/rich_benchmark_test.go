package errors

import (
	"errors"
	"fmt"
	"runtime"
	"testing"
)

// ==================== 性能基准测试 ====================

// BenchmarkNewRich 测试创建 RichError 的性能
func BenchmarkNewRich(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewRich(400001, "参数错误")
	}
}

// BenchmarkWrapRich 测试包装错误的性能
func BenchmarkWrapRich(b *testing.B) {
	b.ReportAllocs()
	originalErr := errors.New("original error")
	for i := 0; i < b.N; i++ {
		_ = WrapRich(originalErr, 500001, "系统错误")
	}
}

// BenchmarkWrapRichWithFormat 测试带格式化的包装性能
func BenchmarkWrapRichWithFormat(b *testing.B) {
	b.ReportAllocs()
	originalErr := errors.New("original error")
	for i := 0; i < b.N; i++ {
		_ = WrapRich(originalErr, 500001, "系统错误: %s", "timeout")
	}
}

// BenchmarkFromRichError 测试转换性能
func BenchmarkFromRichError(b *testing.B) {
	b.ReportAllocs()
	richErr := NewRich(400001, "参数错误")
	for i := 0; i < b.N; i++ {
		_ = FromRichError(richErr)
	}
}

// BenchmarkFromRichError_StdError 测试转换标准错误的性能
func BenchmarkFromRichError_StdError(b *testing.B) {
	b.ReportAllocs()
	stdErr := errors.New("standard error")
	for i := 0; i < b.N; i++ {
		_ = FromRichError(stdErr)
	}
}

// BenchmarkRichError_Error 测试 Error() 方法性能
func BenchmarkRichError_Error(b *testing.B) {
	b.ReportAllocs()
	e := NewRich(400001, "参数错误")
	for i := 0; i < b.N; i++ {
		_ = e.Error()
	}
}

// BenchmarkRichError_FormatV 测试 %v 格式化性能
func BenchmarkRichError_FormatV(b *testing.B) {
	b.ReportAllocs()
	e := NewRich(400001, "参数错误")
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%v", e)
	}
}

// BenchmarkRichError_FormatPlusV 测试 %+v 详细格式化性能
func BenchmarkRichError_FormatPlusV(b *testing.B) {
	b.ReportAllocs()
	e := WrapRich(errors.New("cause"), 400001, "参数错误")
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%+v", e)
	}
}

// BenchmarkCallers 测试堆栈捕获性能
func BenchmarkCallers(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = callers()
	}
}

// BenchmarkUnwrap 测试 Unwrap 性能
func BenchmarkUnwrap(b *testing.B) {
	b.ReportAllocs()
	e := WrapRich(errors.New("cause"), 400001, "参数错误")
	for i := 0; i < b.N; i++ {
		_ = e.Unwrap()
	}
}

// BenchmarkErrorsIs 测试 errors.Is 兼容性性能
func BenchmarkErrorsIs(b *testing.B) {
	b.ReportAllocs()
	originalErr := errors.New("cause")
	e := WrapRich(originalErr, 400001, "参数错误")
	for i := 0; i < b.N; i++ {
		_ = errors.Is(e, originalErr)
	}
}

// ==================== 对比测试 ====================

// BenchmarkCompare_StdError 标准库错误创建作为对比基准
func BenchmarkCompare_StdError(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errors.New("standard error")
	}
}

// BenchmarkCompare_FmtErrorf 标准库格式化错误作为对比
func BenchmarkCompare_FmtErrorf(b *testing.B) {
	b.ReportAllocs()
	originalErr := errors.New("original")
	for i := 0; i < b.N; i++ {
		_ = fmt.Errorf("wrapped: %w", originalErr)
	}
}

// ==================== 内存泄漏检测测试 ====================

// TestMemoryLeak_NewRich 检测 NewRich 内存泄漏
func TestMemoryLeak_NewRich(t *testing.T) {
	// 强制 GC 清理
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 创建大量错误
	const iterations = 100000
	for i := 0; i < iterations; i++ {
		e := NewRich(400001, "参数错误")
		_ = e.Error() // 使用防止被优化掉
	}

	// 强制 GC
	runtime.GC()
	runtime.GC() // 多次确保清理
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// 检测堆内存增长
	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	t.Logf("创建 %d 个 RichError 后堆内存变化: %d bytes", iterations, heapGrowth)

	// 如果堆增长超过 1MB 可能有泄漏
	if heapGrowth > 1024*1024 {
		t.Errorf("可能存在内存泄漏，堆增长: %d bytes", heapGrowth)
	}
}

// TestMemoryLeak_WrapRich 检测 WrapRich 内存泄漏
func TestMemoryLeak_WrapRich(t *testing.T) {
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	originalErr := errors.New("cause")
	const iterations = 100000
	for i := 0; i < iterations; i++ {
		e := WrapRich(originalErr, 500001, "系统错误")
		_ = e.Error()
	}

	runtime.GC()
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	t.Logf("创建 %d 个 WrapRich 后堆内存变化: %d bytes", iterations, heapGrowth)

	if heapGrowth > 1024*1024 {
		t.Errorf("可能存在内存泄漏，堆增长: %d bytes", heapGrowth)
	}
}

// TestMemoryLeak_StackCapture 检测堆栈捕获是否有泄漏
func TestMemoryLeak_StackCapture(t *testing.T) {
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	const iterations = 100000
	for i := 0; i < iterations; i++ {
		s := callers()
		_ = fmt.Sprintf("%+v", s) // 使用堆栈
	}

	runtime.GC()
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	t.Logf("创建 %d 次堆栈捕获后堆内存变化: %d bytes", iterations, heapGrowth)

	if heapGrowth > 1024*1024 {
		t.Errorf("堆栈捕获可能存在内存泄漏，堆增长: %d bytes", heapGrowth)
	}
}

// TestNoCircularReference 检测循环引用
func TestNoCircularReference(t *testing.T) {
	// 创建链式错误
	e1 := NewRich(400001, "第一个错误")
	e2 := WrapRich(e1, 400002, "第二个错误")
	e3 := WrapRich(e2, 400003, "第三个错误")

	// 遍历错误链确保不会死循环
	visited := make(map[*RichError]bool)
	current := e3
	depth := 0
	maxDepth := 100 // 防止无限循环

	for current != nil && depth < maxDepth {
		if visited[current] {
			t.Error("检测到循环引用！")
			return
		}
		visited[current] = true
		depth++

		if unwrapped := current.Unwrap(); unwrapped != nil {
			if re, ok := unwrapped.(*RichError); ok {
				current = re
			} else {
				break
			}
		} else {
			break
		}
	}

	if depth >= maxDepth {
		t.Error("错误链过深，可能存在循环引用")
	}

	t.Logf("错误链深度: %d，无循环引用", depth)
}

// TestStackNotRetained 确保堆栈不会持有过多引用
func TestStackNotRetained(t *testing.T) {
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 在深层调用中创建错误
	var lastErr *RichError
	var deepCall func(depth int)
	deepCall = func(depth int) {
		if depth > 0 {
			deepCall(depth - 1)
		} else {
			lastErr = NewRich(400001, "深层错误")
		}
	}

	// 多次深层调用
	for i := 0; i < 1000; i++ {
		deepCall(20)
		_ = lastErr.Error()
	}

	runtime.GC()
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	t.Logf("深层调用后堆内存变化: %d bytes", heapGrowth)

	// 深层调用不应导致大量内存保留
	if heapGrowth > 512*1024 {
		t.Errorf("深层堆栈可能导致内存问题，堆增长: %d bytes", heapGrowth)
	}
}
