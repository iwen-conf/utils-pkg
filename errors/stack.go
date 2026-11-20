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

// callers 捕获当前调用堆栈（默认跳过 runtime.Callers + callers + 公开构造入口 3 层）。
func callers() *stack {
	return callersSkip(3)
}

func callersSkip(skip int) *stack {
	pcs := stackPool.Get().(*stack)
	*pcs = (*pcs)[:cap(*pcs)]

	n := runtime.Callers(skip, *pcs)
	*pcs = (*pcs)[:n]

	result := make(stack, n)
	copy(result, *pcs)

	*pcs = (*pcs)[:0]
	stackPool.Put(pcs)

	return &result
}

// Format 实现 fmt.Formatter 接口，使用 runtime.CallersFrames 精确还原内联帧与调用行号。
func (s *stack) Format(st fmt.State, verb rune) {
	if s == nil || len(*s) == 0 {
		return
	}
	if verb == 'v' && st.Flag('+') {
		frames := runtime.CallersFrames(*s)
		for {
			frame, more := frames.Next()
			if frame.File != "" {
				fmt.Fprintf(st, "\n\t%s:%d", frame.File, frame.Line)
			}
			if !more {
				break
			}
		}
	}
}
