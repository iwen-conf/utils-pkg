# T05 通用工具收敛

## 目标

删除无调用者或与标准库重复的能力，修复自研缓存边界，并明确 User-Agent 结果只能用于非安全用途。

## 已知范围

1. `slice/slice.go`、`slice/slice_test.go`、`slice/使用说明.md`
2. `useragent/useragent.go`、`useragent/useragent_test.go`、`useragent/使用说明.md`
3. `go.mod`、`go.sum`

## 子任务索引

| 状态 | 子任务 | Finding |
| --- | --- | --- |
| `[ ]` | [T05-01 删除失效并行切片能力](T05-01-删除失效并行切片能力.md) | GOV-001 |
| `[ ]` | [T05-02 移除未使用 slice LRU](T05-02-移除未使用sliceLRU.md) | F-UTIL-001 |
| `[ ]` | [T05-03 重写 User-Agent LRU 边界](T05-03-重写UserAgentLRU边界.md) | F-UTIL-001 |
| `[ ]` | [T05-04 限定 User-Agent 能力与解析测试](T05-04-限定UserAgent能力与解析测试.md) | GOV-001 |

## 验收标准

1. 文档不再宣称不存在的自动并行能力。
2. `golang.org/x/exp` 不再作为直接依赖。
3. 无效 LRU 容量不会在 Put 时才 panic。
4. User-Agent 缓存并发正确，识别结果不被描述为安全判定。
