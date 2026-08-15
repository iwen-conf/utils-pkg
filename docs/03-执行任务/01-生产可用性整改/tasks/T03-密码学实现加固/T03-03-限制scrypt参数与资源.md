# T03-03 修复 F-CRYPTO-001 限制 scrypt 参数与资源

状态：`[ ]`

## 来源

1. finding_id：F-CRYPTO-001（scrypt 部分）。
2. Handoff：[Argon2/scrypt 验证接受无上限的编码参数](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-crypto-001--argon2scrypt-验证接受无上限的编码参数)。
3. 权限类型：service；不可信 hash 可使 anonymous 输入影响可用性。
4. 数据产量：availability capability。

## 目标

scrypt 的 Hash/Verify 在任何 salt 分配或 KDF 计算前校验 N/r/p、估算内存、乘法溢出、salt/key 长度和编码总长度。恶意参数必须快速返回可判断错误，不能触发高内存/CPU。

## 前置

1. T03-02 已建立 KDF 参数校验、错误包装和测试风格，scrypt 应保持一致而非另造宽松语义。

## 允许改动

1. `crypto/scrypt.go`。
2. `crypto/crypto_test.go`，或新增 `crypto/scrypt_test.go`。
3. `crypto/使用说明.md` 的 scrypt 参数范围。

## 禁止改动

1. 不改变默认/fast 参数的当前值。
2. 不自动 clamp 参数或在失败后退回默认配置。
3. 不通过 recover 捕获 `scrypt.Key` 的内存问题。
4. 不在测试中真的执行 256 MiB 上限或超限计算。

<Context-Queries>

```bash
arc-idx search 'f:^crypto/ ScryptParams|DefaultScryptParams|HashWithScrypt|VerifyScryptHash|NewScryptHasher' --repo . --context 5 --max 180 --format json
arc-idx search 'f:^crypto/ N=|R|P|SaltLength|KeyLength|scrypt.Key|strings.Split|fmt.Sscanf' --repo . --context 4 --max 180 --format json
arc-idx ast --repo . --lang go 'func VerifyScryptHash($HASH, $PASSWORD []byte) (bool, error) { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. Hash 在参数校验前按 `SaltLength` 分配。
2. Verify 把编码中的 N/r/p 和 key 长度直接传给 `scrypt.Key`。
3. 无总长度、salt/key 长度或内存上限。
4. 乘法溢出和 scrypt 自身约束只依赖底层返回 error，攻击输入已进入计算路径。
5. `NewScryptHasher` 保存 params 指针，调用方可并发修改。

## 固定参数范围

1. encoded hash 总长度：1–4096 字节。
2. N：大于 1 且必须是 2 的幂。
3. r：1–32。
4. p：1–16。
5. salt length：16–64 字节。
6. key length：16–64 字节。
7. 估算内存 `128*N*r` 不超过 256 MiB。
8. 计算 `128*N*r` 及底层约束相关乘法前必须检查 `int` 溢出。
9. 同时检查 scrypt 约束 `r*p < 1<<30`，乘法先防溢出。

## 固定错误契约

1. 新增 `ErrInvalidScryptParams`，所有参数/格式错误以它分类并保留底层解析原因。
2. 超长编码统一返回以 `ErrInvalidScryptParams` 包装的明确错误；不新增第二个导出 sentinel。
3. 密码不匹配返回 `(false, nil)`；KDF/格式错误返回 `(false, error)`。

## 目标实现

1. 私有 `validateScryptParams(ScryptParams) error` 被 Hash 和 Verify 共用。
2. 私有安全乘法 helper 明确返回 `(value int64/uint64, ok bool)`，禁止让 int wrap-around。
3. Verify 严格解析 `$scrypt$N=...,r=...,p=...$salt$key`，拒绝尾随垃圾和额外段。
4. `NewScryptHasher` 固定改为 `func NewScryptHasher(params *ScryptParams) (*ScryptHasher, error)`；内部保存值副本，无效 params fail-fast。

## 执行清单

- [ ] 定义范围常量、内存上限和 validator；先写纯 validator 边界测试。
- [ ] `HashWithScrypt` 复制 params 并在 `make(salt)` 前验证。
- [ ] Verify 先检查总长度和精确段数，再严格解析 N/r/p。
- [ ] Base64 decode 后校验实际 salt/key 长度，再调用共用 validator。
- [ ] 只有 validator 成功后才能调用 `scrypt.Key`。
- [ ] constructor 按固定新签名返回 error且不保存调用方指针；更新 `PasswordHasher` 测试和文档示例。
- [ ] 测试内存估算临界值使用 helper，不执行大成本 KDF。
- [ ] 更新文档：N 必须为 2 的幂、资源上限、错误密码与错误 hash 的区别。

## 正向测试

1. Default/Fast params validator 和 round trip 通过。
2. 最小合法 N/r/p 与 salt/key 长度通过 validator；实际 KDF 使用低成本合法值。
3. constructor 后修改输入 params 不改变 hasher。
4. 错误密码返回 false/nil。

## 反向测试

1. N 为 0、1、3、负数、非 2 次幂失败。
2. r/p 为 0、负数、超过上限失败。
3. 内存估算等于上限通过 validator，超过 1 字节或发生乘法溢出失败；不要执行该 KDF。
4. `r*p` 超限或溢出失败。
5. 4097 字节、错误前缀、缺段、多段、尾随字符、非法 Base64、salt/key 越界失败。
6. 恶意极大 N 在调用 `scrypt.Key` 前返回。

## 验证命令

```bash
gofmt -w crypto/scrypt.go crypto/*_test.go
go test -count=1 ./crypto -run 'Scrypt|KDFLimit|Overflow|PowerOfTwo|HashTooLong'
go test -race -count=1 ./crypto
go test -count=1 ./...
go vet ./...
git diff --check
```

## 不要做

1. 不依赖底层 `scrypt.Key` 帮忙做全部输入校验。
2. 不使用可能溢出的 `128 * N * R` 后再比较上限。
3. 不吞掉 KDF error 并返回“密码不匹配”。
4. 不在错误或日志中输出明文密码。
5. 不把超限配置自动替换为 Fast 参数。

## 完成标准

1. 所有不可信参数在分配和 KDF 前完成严格验证。
2. 内存上限和整数溢出有不执行大 KDF 的单元测试。
3. 默认合法 hash 保持兼容，constructor 不受外部可变指针影响。
4. crypto 包、race、全量测试和 vet 通过。
