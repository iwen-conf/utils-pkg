package errors

import (
	"fmt"
	"runtime"
)

const depth = 32

// stack 记录函数调用堆栈
type stack []uintptr

// callers 捕获当前调用堆栈
func callers() *stack {
	var pcs [depth]uintptr
	// Skip 3: runtime.Callers, callers, New/Wrap
	n := runtime.Callers(3, pcs[:])
	var st stack = pcs[0:n]
	return &st
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
				// 输出格式: file.go:123
				fmt.Fprintf(st, "\n\t%s:%d", file, line)
			}
		}
	}
}
