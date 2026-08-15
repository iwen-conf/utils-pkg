# T03-01 修复 F-CRYPTO-003 收敛 AES-GCM 接口契约

状态：`[ ]`

## 来源

1. finding_id：F-CRYPTO-003。
2. Handoff：[AES mode API、日志和优化状态误导调用方](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-crypto-003--aes-mode-api日志和优化状态误导调用方)。
3. 权限类型：service / config-ops。
4. 数据产量：SECRET/BIZ confidentiality capability。

## 目标

AES API 只提供实际实现的 GCM 认证加密。CFB、未知 mode 和未知 encoding 必须返回错误；构造过程不打印 stdout，不用启发式熵检查拒绝合法 AES key，并删除没有参与真实路径的字段、锁和 buffer pool。

## 项目定位对齐

1. 影响用户角色：调用本库保存密钥材料、业务字段或敏感配置的下游服务。
2. 信任边界：明文进入本地密码学 API，到认证密文返回调用方。
3. 密钥生成、存储、轮换和访问控制仍由下游或密钥管理系统负责。

## 允许改动

1. `crypto/aes.go`。
2. `crypto/crypto.go` 中只服务 AES 的 `bufferPool` 和相关 import；保留密码策略仍使用的 `nonAlphanumericRegex`，直到 T03-04 决定是否删除。
3. `crypto/crypto_test.go`，或新增 `crypto/aes_test.go`。
4. `crypto/使用说明.md` 的 AES 章节。

## 禁止改动

1. 不实现 CFB、CBC、CTR、ECB、JWE 或自定义 nonce 格式。
2. 不改变现有密文布局 `nonce || ciphertext || tag`。
3. 不增加全局日志器或把 warning 改写到 stderr。
4. 不在本任务更改 Argon2、scrypt 或 PasswordPolicy。

<Context-Queries>

```bash
arc-idx search 'f:^crypto/ AESEncryptor|NewAESEncryptorWithMode|ModeCFB|ModeGCM|getBlock|bufferPool|calculateKeyEntropy' --repo . --context 5 --max 180 --format json
arc-idx search 'f:^crypto/ EncodingType|getEncoder|fmt.Print|sync.Pool|blockMutex|key \[\]byte' --repo . --context 4 --max 180 --format json
arc-idx ast --repo . --lang go 'func NewAESEncryptorWithMode($KEY []byte, $MODE EncryptionMode) (*AESEncryptor, error) { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. `ModeCFB` 只打印 warning，返回对象实际仍使用 GCM。
2. 16/24 字节合法 AES key 会打印 stdout；库调用污染进程日志。
3. `calculateKeyEntropy` 可能拒绝长度合法但重复模式明显的测试/派生 key，却不能证明 key 来源安全。
4. `AESEncryptor.key` 不参与加解密；`blockMutex` 保护构造后从不修改的 block。
5. `bufferPool.Get` 的 buffer 没有作为 `Seal` 的 dst 使用，属于无效优化。
6. 未知 `EncodingType` 被静默当作标准 Base64。

## 固定错误与 API

1. 保留 `NewAESEncryptor(key)`，其语义明确为 GCM。
2. 保留 `NewAESEncryptorWithMode(key, mode)` 作为迁移入口，但只有 `ModeGCM` 成功。
3. 新增 `ErrUnsupportedEncryptionMode`；`ModeCFB` 和所有未知枚举返回 `nil` 与该错误。
4. 新增 `ErrUnsupportedEncoding`；`EncryptWithOptions`、`DecryptWithOptions` 对未知 encoding 返回该错误。
5. AES key 只按标准长度 16/24/32 字节校验；不做字符类别或熵猜测。
6. 所有新错误允许标准库 `errors.Is` 判断，底层 AES/base64/GCM 错误用 `%w` 保留。

## 目标内部结构

```go
type AESEncryptor struct {
    block cipher.Block
}
```

1. `block` 构造后只读，可安全并发使用。
2. 删除 `key`、`blockMutex`、`getBlock`、`calculateKeyEntropy` 和 AES 专用 `bufferPool`。
3. nonce 仍由 `crypto/rand.Reader` 每次生成。
4. encoding helper 改为返回 `(*base64.Encoding, error)`，不能有 default 静默降级。

## 执行清单

- [ ] 先更新/新增测试，固定 CFB 和未知 mode 必须构造失败。
- [ ] 删除所有 `fmt.Print*` warning 和熵 heuristic；文档改为要求 key 来源于 CSPRNG/KMS。
- [ ] 精简 `AESEncryptor` 为只读 block，直接在 Encrypt/Decrypt 中使用。
- [ ] 删除未使用 buffer pool，并清理 `crypto/crypto.go` import；确认 `nonAlphanumericRegex` 是否仍由 PasswordPolicy 使用。
- [ ] 改造 encoding helper，Standard 和 URLSafe 之外全部失败。
- [ ] 保持 nonce size、Seal/Open、Base64 两种合法编码和 ciphertext layout 不变。
- [ ] 更新 AES 文档：`NewAESEncryptor` 不是 CFB/弃用入口；`ModeCFB` 从 v0.4.0 起硬失败。
- [ ] 删除“GCM 比 CFB 快 2–3 倍”“FIPS 认证”等没有本项目证据的绝对宣称。

## 正向测试

1. 16/24/32 字节合法 key 均可构造 GCM，并 round trip 空明文、普通文本和二进制数据。
2. `EncodingStandard`、`EncodingURLSafe` 均可加解密。
3. 同一 encryptor 并发加解密通过 race；不同加密调用产生不同 nonce/密文。
4. 现有合法 GCM 密文仍能解密。

## 反向测试

1. 15/17/23/25/31/33 字节 key 构造失败。
2. `ModeCFB`、负枚举和任意未知 mode 返回 `ErrUnsupportedEncryptionMode` 且 encryptor 为 nil。
3. 未知 encoding 返回 `ErrUnsupportedEncoding`，不输出结果。
4. 截断密文、非法 Base64、篡改 tag 解密失败并保留底层错误。
5. 静态检查 `crypto/` 不再存在 `fmt.Print`、`calculateKeyEntropy`、`bufferPool` 或 `blockMutex`。

## 验证命令

```bash
gofmt -w crypto/aes.go crypto/crypto.go crypto/*_test.go
go test -count=1 ./crypto -run 'AES|GCM|Mode|Encoding|Tamper|Concurrent'
go test -race -count=1 ./crypto
go test -count=1 ./...
go vet ./...
rg -n 'fmt\.Print|calculateKeyEntropy|bufferPool|blockMutex' crypto || true
git diff --check
```

## 不要做

1. 不接受 CFB 后静默改用 GCM。
2. 不通过 stdout/stderr warning 代替 error。
3. 不自己判断 key 是否“看起来随机”；只校验 AES 长度并在文档约束来源。
4. 不复用 nonce、固定 nonce 或把 nonce 从密文中移除。
5. 不保留未使用的优化结构以维持旧代码外观。

## 完成标准

1. API 行为与“只支持 GCM”完全一致，非法枚举失败关闭。
2. 加解密路径无 stdout 输出、无死状态、无 silent encoding fallback。
3. GCM 正反测试、race、全量测试和 vet 通过，文档同步。
