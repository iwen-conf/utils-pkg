# T03-04 修复 F-CRYPTO-002 重构 PasswordPolicy 与随机辅助 API

状态：`[ ]`

## 来源

1. finding_id：F-CRYPTO-002。
2. Handoff：[PasswordPolicy 的并发与 Unicode 契约不成立](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-crypto-002--passwordpolicy-的并发与-unicode-契约不成立)。
3. 设计补充：`GenerateRandomBytes` 的调用方长度直接进入 `make`，需按 fail-fast 原则补边界。
4. 权限类型：service。
5. 数据产量：AUTH policy integrity；随机辅助错误影响可用性和密钥生成调用。

## 目标

PasswordPolicy 的全部配置对外通过值对象读写，验证一次读取不可变快照，按 rune 计长度且无 data race。随机字节 helper 在分配前拒绝无效或过大长度，不 panic、不静默返回空值。

## 允许改动

1. `crypto/password.go`。
2. `crypto/bcrypt.go` 中 `GenerateRandomBytes` 和相关错误。
3. `crypto/crypto.go`：若 PasswordPolicy 不再使用 `nonAlphanumericRegex`，删除该全局和 import。
4. `crypto/crypto_test.go`，或新增 `crypto/password_policy_test.go`、`crypto/random_test.go`。
5. `crypto/使用说明.md` 的密码策略与随机字节章节。

## 禁止改动

1. 不引入全局可变默认策略。
2. 不把复杂密码规则描述为泄漏密码检查或 MFA 替代品。
3. 不改变 bcrypt cost 语义。
4. 不用只锁 setter、不锁 reader 的半同步方案。

<Context-Queries>

```bash
arc-idx search 'f:^crypto/ PasswordPolicy|SetDisallowWords|ValidatePassword|compileRegexes|regexMutex|nonAlphanumericRegex' --repo . --context 5 --max 200 --format json
arc-idx search 'f:^crypto/ GenerateRandomBytes|make\(\[\]byte, length\)|rand.Read|MinLength|DisallowWords' --repo . --context 4 --max 180 --format json
arc-idx ast --repo . --lang go 'func ($P *PasswordPolicy) ValidatePassword($PASSWORD string) error { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. Min/Max/Require/DisallowWords 全部公开，可绕过锁直接修改。
2. `SetDisallowWords` 先改 words、解锁，再另一次加锁编译 regex，验证可观察到不一致快照。
3. Validate 复制 regex slice 后，在锁外读取公开 `DisallowWords[i]`，并发更新可 race 或越界。
4. 长度使用 byte 数，Unicode 密码与字符类型检查的 rune 语义不一致。
5. `GenerateRandomBytes(-1)` 在 `make` 处 panic；0 或非常大长度没有明确契约。

## 固定公开 API

```go
type PasswordPolicyConfig struct {
    MinLength      int
    MaxLength      int
    RequireUpper   bool
    RequireLower   bool
    RequireNumber  bool
    RequireSpecial bool
    DisallowWords  []string
}

func DefaultPasswordPolicyConfig() PasswordPolicyConfig
func NewPasswordPolicy(config PasswordPolicyConfig) (*PasswordPolicy, error)
func NewDefaultPasswordPolicy() *PasswordPolicy
func (p *PasswordPolicy) Config() (PasswordPolicyConfig, error)
func (p *PasswordPolicy) SetConfig(config PasswordPolicyConfig) error
func (p *PasswordPolicy) SetDisallowWords(words []string) error
func (p *PasswordPolicy) ValidatePassword(password string) error
```

1. `PasswordPolicy` 不再导出配置字段。
2. `NewDefaultPasswordPolicy` 使用已知合法默认值，保持无 error 便捷入口。
3. Config、constructor 和 setter 均克隆 `DisallowWords`。
4. `Config()` 返回克隆和 error，调用方修改不能影响内部状态；nil receiver 返回 `ErrInvalidPasswordPolicy`。

## 固定内部模型

1. 使用不可变 `compiledPasswordPolicy` 快照，包含 config、规范化禁用词和编译结果。
2. constructor/SetConfig 完整校验并编译新快照，成功后一次原子替换；失败时旧快照保持不变。
3. 可使用 `atomic.Pointer[compiledPasswordPolicy]`；若使用 mutex，Validate 必须一次复制完整快照，不能分字段读取。
4. `SetDisallowWords` 基于当前 Config 构造新配置，再调用 `SetConfig`。

## 固定校验规则

1. `MinLength >= 1`，`MaxLength >= MinLength`；无效配置返回 `ErrInvalidPasswordPolicy`。
2. 长度使用 `utf8.RuneCountInString` 或 `len([]rune(password))`。
3. Upper/Lower/Number/Special 继续使用 Unicode 分类。
4. 禁用词匹配保持大小写不敏感和字面量语义；配置 slice 中的空字符串必须拒绝，不能编译成匹配所有密码。
5. 密码规则错误增加稳定哨兵，如 `ErrPasswordTooShort`、`ErrPasswordTooLong`、`ErrPasswordMissingUpper` 等；详细长度用 `%w` 包装。

## 随机字节固定规则

1. 导出 `MaxRandomBytesLength = 1 << 20`（1 MiB）。
2. `GenerateRandomBytes(length)` 在 `length <= 0` 时返回 `ErrInvalidRandomLength`。
3. `length > MaxRandomBytesLength` 返回同一分类错误并说明上限。
4. 只有范围合法后才能 `make`；`rand.Read` 错误用 `%w` 保留。

## 执行清单

- [ ] 定义 Config、错误和不可变 snapshot；先写构造/克隆/无效配置测试。
- [ ] 移除 PasswordPolicy 公开可变字段和两阶段 `compileRegexes`。
- [ ] 实现 `SetConfig` 的“先构造、后原子替换”，失败不修改现有策略。
- [ ] Validate 开始时只加载一次 snapshot，并完全基于该值判断。
- [ ] 长度改按 rune；补 Unicode 正反案例。
- [ ] `SetDisallowWords` 返回 error，克隆调用方 slice；更新全部调用点。
- [ ] 为 `GenerateRandomBytes` 增加分配前上下限和错误测试。
- [ ] 更新使用说明中的直接字段赋值示例为 Config 构造；所有 error 必须处理，不再用 `_` 忽略。

## 正向测试

1. 默认配置保持现有强度规则，合法 ASCII 密码通过。
2. 自定义 `MinLength=4/MaxLength=4` 时 `"中A1!"` 按 4 rune 通过。
3. constructor/Config/SetDisallowWords 后修改原 slice 或返回 slice，内部策略不变。
4. 并发 goroutine 反复 `SetConfig` 与 `ValidatePassword`，race 无报告且每次结果对应某个完整快照。
5. `GenerateRandomBytes(1)`、32、上限值在范围校验上成功；实际测试不要为上限反复分配。

## 反向测试

1. min 0、max 小于 min、空禁用词配置失败，旧 snapshot 不变。
2. Unicode 密码的过短/过长按 rune 判断，不按 UTF-8 字节误判。
3. 缺少每种要求分别匹配对应哨兵错误。
4. nil receiver 的 Config/Validate/SetConfig/SetDisallowWords 全部返回 `ErrInvalidPasswordPolicy`，不 panic。
5. `GenerateRandomBytes(-1)`、0、上限+1 返回 error 且不 panic/不分配。

## 验证命令

```bash
gofmt -w crypto/password.go crypto/bcrypt.go crypto/crypto.go crypto/*_test.go
go test -count=1 ./crypto -run 'PasswordPolicy|Unicode|Snapshot|RandomBytes|InvalidRandom'
go test -race -count=1 ./crypto
go test -count=1 ./...
go vet ./...
git diff --check
```

## 不要做

1. 不把公开字段保留为“兼容别名”；它们会继续绕过同步。
2. 不在 Validate 中一边持锁一边执行可变外部回调；本策略没有外部回调需求。
3. 不用 byte 长度和 rune 类型检查混合语义。
4. 不让空禁用词 regex 匹配所有密码。
5. 不捕获负长度 panic 后返回 nil；必须在分配前校验。

## 完成标准

1. PasswordPolicy 配置不可被外部直接或间接修改，Validate 读取完整快照。
2. Unicode、配置失败原子性和并发更新通过测试/race。
3. 随机 helper 对无效长度 fail-fast 且不 panic。
4. 所有示例使用新 Config API并处理错误，进度表同步。
