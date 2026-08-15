# T01-04 修复 F-AUTH-004 实现 refresh 原子消费存储

状态：`[ ]`

## 来源

1. finding_id：F-AUTH-004。
2. Handoff：[Refresh 防重放默认状态仅在单进程内原子](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-auth-004--refresh-防重放默认状态仅在单进程内原子)。
3. 权限类型：user，前提是同一 refresh token 被窃取或并发使用。
4. 数据产量：AUTH capability，可产生多组有效 token。

## 目标

把 refresh JTI 一次性消费抽象成可共享、可原子实现的存储接口；两个 `TokenManager` 使用同一存储并发刷新时，只允许一次首次消费。

## 前置

1. T01-01～T01-03 已完成。

## 允许改动

1. 新增 `jwt/refresh_store.go` 和对应测试。
2. `jwt/jwt.go` 中 refresh JTI 字段、构造、refresh、cleanup。
3. `jwt/revocation.go` 仅限共享 helper，不把新接口合并进 `RevocationStore`。
4. `jwt/使用说明.md` 的单实例/多实例章节。

## 禁止改动

1. 不把一次性消费塞进现有普通 `RevocationStore` 接口。
2. 不直接依赖 Redis、SQL 或特定云 SDK。
3. 不用 Get-then-Set 冒充跨实例原子性。
4. 不保存原始 refresh token。

## 目标接口

```go
type RefreshTokenUseStore interface {
    Consume(ctx context.Context, jti string, expiresAt time.Time) (firstUse bool, err error)
    IsConsumed(ctx context.Context, jti string) (bool, error)
}
```

可选能力：

```go
type RefreshTokenUseStoreCleaner interface {
    CleanExpired(ctx context.Context, now time.Time) (int, error)
}
```

`JWTOptions` 增加 `RefreshTokenUseStore`；nil 时使用线程安全内存实现。

<Context-Queries>

```bash
arc-idx search 'f:^jwt/ usedRefreshJTIs|consumeRefreshJTI|refreshJTIUsed|cleanUsedRefreshJTIs|RefreshTokenContext' --repo . --context 5 --max 200 --format json
arc-idx search 'f:^jwt/ RevocationStore|newManagerWithRevocationStore|SharedRevocation' --repo . --context 4 --max 150 --format json
arc-idx ast --repo . --lang go 'if $FIRST := $M.consumeRefreshJTI($$$); !$FIRST { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. 已消费 refresh JTI 保存在 TokenManager 本地分片 map。
2. 两个实例即使共享 RevocationStore，也会各自把同一 JTI 当首次使用。
3. 外部存储错误与首次消费语义没有独立契约。

## 执行清单

- [ ] 新增接口、可选 cleaner 和 `MemoryRefreshTokenUseStore`；内存实现使用 mutex + map，`Consume` 在一个临界区内完成“检查未过期记录 + 写入”。
- [ ] `Consume` 对空 jti、非未来 expiresAt、已取消 context 返回明确错误；不要把空 jti 当未消费。
- [ ] `IsConsumed` 遇到过期记录时可删除并返回 false，必须并发安全。
- [ ] 从 `TokenManager` 删除分片 map/locks 和 `refreshJTISegments` 等仅服务旧实现的字段/函数。
- [ ] `RefreshTokenContext` 在生成新 token 前调用 store `Consume`；store error 立即返回并且不得签发 token。
- [ ] `firstUse=false` 时按现有策略撤销 session；无 sid 时保留明确 `ErrSessionIDMissing`，不要静默跳过。
- [ ] 已被 token revocation 拒绝的 refresh token 重放路径使用 `IsConsumed` 判断，store error 失败关闭。
- [ ] cleanup routine 只对实现 cleaner 的 refresh store 清理；错误交给 T01-05 的 reporter 机制，当前至少必须返回/记录而不静默。
- [ ] 单元测试内存 store：首次 true、第二次 false、过期后可重新消费、并发 100 goroutine 只有一个 true、context cancel。
- [ ] 集成测试：两个 manager 共享同一 `MemoryRefreshTokenUseStore` 和 `RevocationStore`，并发刷新同一 token，首次消费数必须为 1；检测重放后 session 被撤销。
- [ ] 文档给出外部实现要求：必须使用 SETNX/唯一键/事务，禁止 Get-then-Set。

## 正向测试

1. 内存 store 首次 Consume=true、再次=false、过期后可重新消费。
2. 100 goroutine 并发只有一个 firstUse=true。
3. 两个 manager 共享同一 store 时只有一条刷新成功路径。

## 反向测试

1. 空 JTI、非未来 expiresAt、取消 context 返回错误。
2. store error 时不得签发任何新 access/refresh token。
3. 重放后 session 撤销失败必须返回错误，不静默继续。

## 验证命令

```bash
gofmt -w jwt/*.go
go test -count=1 ./jwt -run 'Refresh|Consume|Replay|Shared'
go test -race -count=1 ./jwt
go test -count=1 ./...
go vet ./...
```

## 不要做

1. 不声称普通 `RevocationStore` 自动提供 refresh 原子消费。
2. 不在库中直接依赖 Redis/数据库 SDK。
3. 不用“先查再写”的两步实现冒充原子操作。
4. 不在 store key 中保存原始 refresh token；只使用 jti 或其不可逆派生 key。

## 完成标准

1. 跨两个 manager 的并发测试稳定只有一次首次消费。
2. store 故障时零新 token 返回。
3. 默认内存限制和外部原子要求已写入文档。
