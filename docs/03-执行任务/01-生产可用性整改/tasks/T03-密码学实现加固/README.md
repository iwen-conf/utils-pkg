# T03 密码学实现加固

## 目标

让密码学 API 的模式、参数、资源消耗和并发配置可预测，删除误导性优化与静默行为。

## 已知范围

1. `crypto/aes.go`、`crypto/crypto.go`
2. `crypto/argon2.go`、`crypto/scrypt.go`
3. `crypto/password.go`、`crypto/bcrypt.go`、`crypto/benchmark.go`
4. `crypto/crypto_test.go`、`crypto/使用说明.md`

## 子任务索引

| 状态 | 子任务 | Finding |
| --- | --- | --- |
| `[ ]` | [T03-01 收敛 AES-GCM 接口契约](T03-01-收敛AESGCM接口契约.md) | F-CRYPTO-003 |
| `[ ]` | [T03-02 限制 Argon2 参数与资源](T03-02-限制Argon2参数与资源.md) | F-CRYPTO-001 |
| `[ ]` | [T03-03 限制 scrypt 参数与资源](T03-03-限制scrypt参数与资源.md) | F-CRYPTO-001 |
| `[ ]` | [T03-04 重构 PasswordPolicy 与随机辅助 API](T03-04-重构PasswordPolicy与随机辅助API.md) | F-CRYPTO-002 |

## 验收标准

1. AES 只接受 GCM，不输出 stdout 警告。
2. KDF 在分配/计算前拒绝越界和溢出参数。
3. PasswordPolicy 并发配置不 race，Unicode 长度一致。
4. 所有新增错误保留根因并可用 `errors.Is/As` 判断。
