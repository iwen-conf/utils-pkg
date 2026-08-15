# T01 JWT 认证完整性

## 目标

保证 JWT 验证、缓存、refresh rotation、吊销和生命周期不会因性能优化或默认内存状态破坏认证完整性。

## 已知范围

1. `jwt/jwt.go`
2. `jwt/identity.go`
3. `jwt/revocation.go`
4. `jwt/*_test.go`
5. `jwt/使用说明.md`

## 子任务索引

| 状态 | 子任务 | Finding |
| --- | --- | --- |
| `[/]` | [T01-01 修复 JWT 缓存越过 exp](T01-01-修复JWT缓存越过exp.md) | F-AUTH-001 |
| `[ ]` | [T01-02 隔离缓存 claims 可变状态](T01-02-隔离缓存claims可变状态.md) | F-AUTH-002 |
| `[ ]` | [T01-03 严格 JWT 解析与验证策略](T01-03-严格JWT解析与验证策略.md) | F-AUTH-003 |
| `[ ]` | [T01-04 实现 refresh 原子消费存储](T01-04-实现refresh原子消费存储.md) | F-AUTH-004 |
| `[ ]` | [T01-05 修复生命周期配置与错误传播](T01-05-修复JWT生命周期配置与错误传播.md) | F-AUTH-005 |

## 验收标准

1. 缓存不能延长 token 有效期，也不能被调用方修改污染。
2. malformed/无 exp/非 HS256 token 明确失败。
3. 多实例能力通过接口契约表达，默认内存限制写清。
4. 外部存储错误失败关闭且可观测，Shutdown 可重复调用。
5. `go test -race ./jwt` 通过新增并发路径。
