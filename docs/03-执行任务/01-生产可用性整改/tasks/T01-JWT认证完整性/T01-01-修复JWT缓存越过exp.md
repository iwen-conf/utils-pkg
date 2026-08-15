# T01-01 修复 F-AUTH-001 JWT 缓存越过 exp

状态：`[/]`

## 来源

1. finding_id：F-AUTH-001
2. Handoff：[JWT 成功缓存可延长 token 实际有效期](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-auth-001--jwt-成功缓存可延长-token-实际有效期)
3. 权限类型：anonymous/user，前提是持有曾经有效的 token。
4. 数据产量：AUTH capability；影响该 token 的全部下游能力。

## 目标

成功验证结果在 token `exp` 到达后必须立即失效，即使 `CacheTTL` 尚未到期。缓存仍可优化重复解析，但绝不能改变 JWT 自身有效期。

## 允许改动

1. `jwt/jwt.go` 中 `cacheItem`、`checkCache`、`cacheResult` 及紧邻的私有辅助函数。
2. `jwt/jwt_coverage_test.go`，或新增 `jwt/cache_security_test.go`。

## 禁止改动

1. 本任务不处理 claims 深克隆；留给 T01-02。
2. 本任务不改变 parser 的算法、issuer、audience 或 `exp required` 策略；留给 T01-03。
3. 不通过关闭默认缓存、把 `CacheTTL` 改得很短或每次强制重新解析来绕过问题。

<Context-Queries>

```bash
arc-idx search 'f:^jwt/ cacheItem|checkCache|cacheResult|CacheTTL|ExpiresAt' --repo . --context 4 --max 100 --format json
arc-idx search 'f:^jwt/ TestTokenManager_checkCache|GetCacheSize|ExpiresIn' --repo . --context 3 --max 100 --format json
arc-idx ast --repo . --lang go 'func ($M *TokenManager) checkCache($TOKEN string) ($$$)' --format json
```

</Context-Queries>

## 当前行为

1. `cacheItem` 只有写入 `timestamp`。
2. `checkCache` 使用 `time.Since(timestamp) > cacheTTL` 判断缓存过期。
3. 若 token 在临近 `exp` 时首次进入缓存，它可在 `exp` 后继续命中。

## 输出

1. `cacheItem` 增加绝对有效截止时间，例如 `validUntil time.Time`。
2. 成功缓存的截止时间为 `min(now+cacheTTL, claims.ExpiresAt.Time)`。
3. 失败结果保持负缓存能力，但只使用 `now+cacheTTL`。
4. 覆盖真实 token 过期跨缓存命中的回归测试。

## 固定内部契约

1. `cacheItem` 必须包含绝对 `validUntil time.Time`；`timestamp` 如因 LRU 排序保留，不得再参与安全有效期判断。
2. `checkCache` 的公开/私有签名保持当前三返回值形式，不在本任务扩大 API。
3. `cacheResult` 仍负责正负缓存；成功项截止为 `min(now+CacheTTL, exp)`，失败项截止为 `now+CacheTTL`。

## 执行清单

- [ ] 阅读 `jwt/jwt.go:786-827`，确认没有其他代码直接构造 `cacheItem`；测试中的直接构造必须同步。
- [ ] 修改 `cacheItem`：保留 `claims` 和 `err`，将基于写入时间的语义改为绝对 `validUntil`。可以暂时保留 `timestamp` 仅用于 LRU 排序，但不得再用它判断 token 有效性。
- [ ] 在 `cacheResult` 中只读取一次 `now := time.Now()`；负缓存截止为 `now.Add(cacheTTL)`。
- [ ] 成功结果若 `claims == nil`，视为内部编程错误，不写入成功缓存；返回路径不得 panic。
- [ ] 成功结果若 `claims.ExpiresAt != nil`，把截止时间压到 `ExpiresAt.Time`；若截止时间不晚于 `now`，不写缓存。
- [ ] 在 `checkCache` 中使用 `!time.Now().Before(item.validUntil)` 判定失效，持写锁删除后返回 miss。
- [ ] 保持吊销检查顺序不变：token/session revocation 仍在缓存结果返回前执行。
- [ ] 更新现有直接写 `cacheItem{timestamp: ...}` 的测试，使其明确设置 `validUntil`。
- [ ] 新增集成回归：生成约 1 秒有效 token，先验证并确认缓存命中条件，等待到 `ExpiresAt` 后再次验证，必须返回过期错误且缓存项被删除。测试使用标准 `errors.Is` 判断 `jwt.ErrTokenExpired` 或依赖库实际哨兵错误。
- [ ] 新增边界测试：`CacheTTL` 比 token 剩余寿命短时按 TTL 失效；负缓存仍按 TTL 失效。

## 正向测试

1. 未过期 token 在 `validUntil` 前仍可缓存命中。
2. CacheTTL 先到时缓存 miss 后重新验证，token 本身仍合法。
3. 负缓存只受 CacheTTL 控制并按期失效。

## 反向测试

1. token `exp` 先到时，即使 CacheTTL 尚未到期也必须失败并删除缓存。
2. 已过期或 `claims == nil` 的成功项不得写入缓存。
3. 吊销 token/session 不能因缓存命中绕过。

## 验证命令

```bash
gofmt -w jwt/jwt.go jwt/*_test.go
go test -count=1 ./jwt -run 'Cache|Expired'
go test -race -count=1 ./jwt
go test -count=1 ./...
go vet ./...
git diff --check
```

## 期望结果

1. token 在 `exp` 后无法因缓存继续通过。
2. token 未过期且在 cache TTL 内仍可命中。
3. 现有吊销测试继续通过。

## 不要做

1. 不使用 `time.Sleep(5*time.Minute)` 一类长测试；依据 token 实际 `ExpiresAt` 计算最多约 1 秒等待。
2. 不捕获过期错误后返回旧 claims。
3. 不把 `ExpiresAt == nil` 当作永久 token；当前任务只是不缓存，T01-03 将统一要求 exp。
4. 不修改 README 或版本号。

## 完成标准

1. 新回归测试在旧实现上失败、在新实现上通过。
2. `go test -race ./jwt` 与全量测试通过。
3. 进度表记录验证命令和结果，并由 R-verify 将状态改为 `[x]`。
