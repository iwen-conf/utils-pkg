# T01-02 修复 F-AUTH-002 隔离缓存 claims 可变状态

状态：`[ ]`

## 来源

1. finding_id：F-AUTH-002
2. Handoff：[缓存返回可变 claims 指针，调用方可污染后续验证](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-auth-002--缓存返回可变-claims-指针调用方可污染后续验证)。
3. 权限类型：service / in-process consumer。
4. 影响：调用方可修改缓存中的 subject、audience、roles、scopes 或嵌套 custom claims。

## 目标

每次 `ValidateToken` 返回的 claims 都是独立对象。调用方修改返回值后，再次验证同一 token 必须得到签名时的原始内容。

## 前置

1. T01-01 已完成并稳定定义 `cacheItem`。

## 允许改动

1. `jwt/jwt.go` 的缓存读写和新克隆辅助函数。
2. 必要时在 `jwt/identity.go` 复用/扩展 clone helper，但不得改变 identity 业务规则。
3. `jwt/*_test.go` 或新增 `jwt/claims_clone_test.go`。

## 禁止改动

1. 不使用 JSON marshal/unmarshal 作为每次缓存命中的深拷贝实现；它会隐藏不支持类型和引入不必要错误路径。
2. 不把缓存改为保存原始 token 后每次重新解析。
3. 不允许“文档要求调用方不要修改”代替代码隔离。

<Context-Queries>

```bash
arc-idx search 'f:^jwt/ StandardClaims|CustomClaims|cloneMapClaims|cloneStringSlice|cacheResult|checkCache' --repo . --context 4 --max 150 --format json
arc-idx search 'f:^jwt/ map\[string\]interface\{|\[\]interface\{|Audience' --repo . --context 2 --max 100 --format json
arc-idx ast --repo . --lang go '$DST := *$SRC' --format json
```

</Context-Queries>

## 当前行为

1. 首次解析得到的 claims 同时被写入缓存并返回给调用方。
2. 后续缓存命中返回同一指针。
3. `Audience`、NumericDate 指针和 CustomClaims 的嵌套 map/slice 都是可变引用。

## 目标实现

1. 新增 `cloneStandardClaims(*StandardClaims) (*StandardClaims, error)` 和 `cloneClaimValue(any) (any, error)`。
2. 克隆 `RegisteredClaims` 中 `Audience` slice，以及 `ExpiresAt`、`NotBefore`、`IssuedAt` 指针。
3. 克隆 `CustomClaims`：支持 JSON 解码后实际会出现的 `nil/string/bool/float64/json.Number/[]interface{}/map[string]interface{}`，同时兼容已有 `[]string` 和 `map[string]string`。
4. 对明确不可变标量原样返回；对不支持的 map/slice/指针等可变类型返回明确错误，不能静默共享。
5. `cacheResult` 固定改为返回 error；写缓存 clone 失败时不写缓存并向 `ValidateTokenContext` 传播错误。缓存命中 clone 失败同样返回错误和 hit，不回退共享对象。

## 执行清单

- [ ] 确认 `StandardClaims` 的全部可变字段和指针字段。
- [ ] 实现 `cloneNumericDate`、`cloneClaimValue`、`cloneStandardClaims`，每个 helper 保持私有且有针对性测试。
- [ ] `cacheResult` 写入 `cloneStandardClaims(claims)`，不保存调用方即将收到的对象。
- [ ] `checkCache` 命中成功项时再次克隆后返回。
- [ ] 错误缓存可复用 immutable error 值；不要试图深拷贝任意 error。
- [ ] 新增测试：首次验证后修改 `Subject`、`TokenID`、`Audience[0]`、顶层 custom claim、嵌套 map 和嵌套 slice；第二次验证值必须全部保持原始内容。
- [ ] 新增测试：修改第二次返回对象也不能污染第三次验证。
- [ ] 新增并发测试：多个 goroutine 验证同一 token 并修改各自返回值，race detector 不报错，最终原始 claims 不变。

## 正向测试

1. 修改首次、第二次返回对象均不影响后续结果。
2. Subject/TokenID/Audience/NumericDate/嵌套 map/slice 全部独立。
3. 多 goroutine 各自修改返回对象无 race。

## 反向测试

1. 不支持的可变 claim 类型触发明确 clone error，不返回共享对象。
2. 浅拷贝实现会被 Audience 和嵌套 custom claim 测试击穿。
3. clone 失败后缓存大小不增加。

## 验证命令

```bash
gofmt -w jwt/jwt.go jwt/identity.go jwt/*_test.go
go test -count=1 ./jwt -run 'Clone|Mutation|Cache'
go test -race -count=1 ./jwt
go test -count=1 ./...
git diff --check
```

## 不要做

1. 不返回 `CustomClaims` 原 map。
2. 不只浅拷贝 `StandardClaims` 结构体。
3. 不把未知可变类型直接共享并声称“调用方责任”。
4. 不改变 claim key 或 identity normalization。

## 完成标准

1. 旧实现能被 mutation 回归测试稳定击穿，新实现不能。
2. 并发验证和修改通过 race。
3. 进度表同步结果。
