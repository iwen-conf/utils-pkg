package crypto

import (
	"regexp"
	"sync"
)

var (
	// 预编译正则表达式
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

	// 对象池，用于加密和解密操作的缓冲区
	bufferPool = sync.Pool{
		New: func() interface{} {
			// 默认分配 1KB 缓冲区
			buf := make([]byte, 0, 1024)
			return &buf
		},
	}
)