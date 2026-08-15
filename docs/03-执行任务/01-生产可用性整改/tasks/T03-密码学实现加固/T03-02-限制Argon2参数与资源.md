# T03-02 修复 F-CRYPTO-001 限制 Argon2 参数与资源

状态：`[ ]`

## 来源

1. finding_id：F-CRYPTO-001（Argon2 部分）。
2. Handoff：[Argon2/scrypt 验证接受无上限的编码参数](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-crypto-001--argon2scrypt-验证接受无上限的编码参数)。
3. 权限类型：service；当编码 hash 可由外部提交或篡改时可扩大到 anonymous。
4. 数据产量：availability capability；恶意参数可消耗进程 CPU/内存。

## 目标

Hash 和 Verify 在分配 salt、转换整数或调用 Argon2 前使用同一套严格参数验证。编码 hash 的长度、memory、iterations、parallelism、salt 和 key 都有固定上限，未知算法类型和整数截断必须失败。

## 允许改动

1. `crypto/argon2.go`。
2. `crypto/crypto_test.go`，或新增 `crypto/argon2_test.go`。
3. `crypto/使用说明.md` 的 Argon2 参数单位、范围和不可信 hash 说明。

## 禁止改动

1. 不降低 `DefaultArgon2Params` 和 `FastArgon2Params` 的现有合法默认值。
2. 不把超限参数自动 clamp 到上限。
3. 不在测试中实际分配 256 MiB 或运行最大成本参数。
4. 不增加服务级超时、队列或远程哈希服务。

<Context-Queries>

```bash
arc-idx search 'f:^crypto/ Argon2Params|DefaultArgon2Params|HashWithArgon2|VerifyArgon2Hash|NewArgon2Hasher' --repo . --context 5 --max 180 --format json
arc-idx search 'f:^crypto/ memory|iterations|parallelism|SaltLength|KeyLength|uint8\(parallelism\)|strings.Split' --repo . --context 4 --max 180 --format json
arc-idx ast --repo . --lang go 'func VerifyArgon2Hash($HASH, $PASSWORD []byte) (bool, error) { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. `HashWithArgon2` 在校验参数前执行 `make([]byte, SaltLength)` 和 KDF。
2. Verify 只检查 memory 非零；iterations、parallelism、salt/key 长度和编码总长度无上限。
3. parallelism 先解析为 uint32，后直接转 uint8，值大于 255 会截断。
4. 未知 `Argon2Type` 在 Hash 中落入 default Argon2id，隐藏配置错误。
5. `NewArgon2Hasher` 保存调用方 params 指针，调用方并发修改可改变行为并产生 race。
6. `Memory` 注释写“字节”，实际 Argon2 API 单位为 KiB。

## 固定参数范围

1. encoded hash 总长度：1–4096 字节。
2. memory：`8*1024`–`256*1024` KiB。
3. iterations：1–10。
4. parallelism：1–16。
5. salt length：16–64 字节。
6. key length：16–64 字节。
7. type：仅 `Argon2i`、`Argon2id`。
8. version：仅 19。

## 固定错误契约

1. 新增 `ErrInvalidArgon2Params`，所有范围、类型和解析错误均以它为外层分类，并用 `%w` 保留底层 `strconv`/Base64 错误。
2. 超长编码统一返回以 `ErrInvalidArgon2Params` 包装的明确错误；不新增第二个导出 sentinel。
3. 密码不匹配返回 `(false, nil)`；格式/参数错误返回 `(false, error)`。

## 目标实现

1. 私有 `validateArgon2Params(Argon2Params) error` 被 Hash 和 Verify 共用。
2. Verify 在 `strings.Split`、Base64 decode 和 KDF 之前先检查总长度。
3. 解析数字使用能拒绝尾随垃圾和溢出的 `strconv` 路径，不依赖宽松 `fmt.Sscanf`。
4. parallelism 在验证 1–16 后才转 uint8。
5. `NewArgon2Hasher` 固定改为 `func NewArgon2Hasher(params *Argon2Params) (*Argon2Hasher, error)`；保存 params 值副本，不持有调用方指针，nil 使用默认值副本。

## 执行清单

- [ ] 定义集中常量和 `validateArgon2Params`；修正 Memory 注释为 KiB。
- [ ] `HashWithArgon2` 在 `make(salt)` 前复制并验证 params；nil 使用默认副本。
- [ ] 未知 Argon2 type 直接返回错误，不落入 Argon2id。
- [ ] Verify 先拒绝空/超长字符串，再严格验证六段格式、算法名、version 和参数键。
- [ ] Base64 decode 后把实际 salt/key 长度写入参数对象，再调用同一 validator。
- [ ] 只有 validator 通过后才能调用 `argon2.Key/IDKey`。
- [ ] `Argon2Hasher` 内部改存值；constructor 按固定新签名返回 error，同步全部调用点、测试和 v0.4.0 迁移说明，禁止返回无效对象。
- [ ] 测试 validator 边界时直接调用私有 helper，不实际运行最大成本 KDF。
- [ ] 更新文档单位、上下限和“不要验证不受信任的无界 hash”说明。

## 正向测试

1. 默认和 Fast params 均通过 validator，hash/verify 正确密码成功，错误密码为 false/nil。
2. 最小合法参数通过 validator；实际 KDF 测试可选用低但合法的 8 MiB 配置。
3. Argon2i 和 Argon2id 都有合法 round trip。
4. constructor 后修改原 params 不影响 hasher。

## 反向测试

1. memory、iterations、parallelism、salt/key 每个字段分别测试下界外和上界外。
2. parallelism 256、257 等值在转换前失败，不能截断后计算。
3. 4097 字节编码、未知 type、错误 version、缺段、多段、重复/未知参数键和尾随垃圾失败。
4. 非法 Base64、解码后过短/过长 salt/key 失败。
5. 恶意 `m=4294967295` 等参数在 KDF 前快速失败；测试不发生巨量内存分配。

## 验证命令

```bash
gofmt -w crypto/argon2.go crypto/*_test.go
go test -count=1 ./crypto -run 'Argon2|KDFLimit|HashTooLong|Params'
go test -race -count=1 ./crypto
go test -count=1 ./...
go vet ./...
git diff --check
```

## 不要做

1. 不先调用 Argon2 再根据耗时或 panic 判断参数不安全。
2. 不用 uint8 转换“自然限制” parallelism；转换前必须验证。
3. 不自动修正攻击者参数或退回默认值。
4. 不在错误或日志中输出明文密码或完整 hash。
5. 不以超长测试真的申请数百 MiB 来证明拒绝逻辑。

## 完成标准

1. Hash/Verify 共用同一 validator，所有分配和 KDF 均发生在验证之后。
2. 资源上限、解析严格性和指针隔离都有正反测试。
3. 默认合法 hash 兼容，错误密码语义不变。
4. crypto 包、race、全量测试和 vet 通过。
