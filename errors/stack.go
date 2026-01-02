package errors

import (
	"fmt"
	"runtime"
	"sync"
)

const depth = 32

// stack 记录函数调用堆栈
type stack []uintptr

// stackPool 使用 sync.Pool 复用堆栈数组，减少内存分配
var stackPool = sync.Pool{
	New: func() interface{} {
		s := make(stack, 0, depth)
		return &s
	},
}

// callers 捕获当前调用堆栈
func callers() *stack {
	pcs := stackPool.Get().(*stack)
	*pcs = (*pcs)[:cap(*pcs)]

	// Skip 3: runtime.Callers, callers, New/Wrap
	n := runtime.Callers(3, *pcs)
	*pcs = (*pcs)[:n]

	// 复制一份返回，原始 slice 放回 pool
	result := make(stack, n)
	copy(result, *pcs)

	*pcs = (*pcs)[:0]
	stackPool.Put(pcs)

	return &result
}

// Format 实现 fmt.Formatter 接口
func (s *stack) Format(st fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case st.Flag('+'):
			for _, pc := range *s {
				f := runtime.FuncForPC(pc)
				if f == nil {
					continue
				}
				file, line := f.FileLine(pc)
				fmt.Fprintf(st, "\n\t%s:%d", file, line)
			}
		}
	}
}
