# Assets

## 身份与会话资产

| 资产 | 入口 | 关键能力 | 信任边界 |
| --- | --- | --- | --- |
| JWT 签发与验证 | `jwt.TokenManager` | access/refresh、claims、缓存 | 外部 token → HMAC/claims 校验 |
| Refresh rotation | `RefreshTokenContext` | 一次性消费、重放检测、session revoke | 客户端 refresh token → 本地/共享状态 |
| 吊销存储 | `RevocationStore` | token/session 提前失效 | 库 → 外部存储适配器 |
| 身份映射 | `Identity` / `TokenProfile` | actor/tenant/roles/scopes | 下游业务身份 → custom claims |

## 密码与密钥资产

| 资产 | 入口 | 关键能力 | 信任边界 |
| --- | --- | --- | --- |
| AES-GCM | `crypto.AESEncryptor` | 对称加解密 | 调用方密钥/明文 → 密文 |
| 密码哈希 | bcrypt/Argon2/scrypt | 密码派生与验证 | 密码、编码 hash → KDF 参数 |
| 密码策略 | `PasswordPolicy` | 长度、字符类、禁用词 | 用户密码 → 策略结果 |
| 随机数据 | `GenerateRandomBytes` | 密钥/nonce 辅助 | 调用方长度 → 内存分配 |

## 签名和一次性验证资产

| 资产 | 入口 | 关键能力 | 信任边界 |
| --- | --- | --- | --- |
| 签名 URL | `url.URLBuilder` / `ValidateSignature` | HMAC、时间限制 | 外部 URL → 资源/API 授权判断 |
| 签名游标 | `pagination.HMACCodec` | cursor 完整性 | 外部 cursor → 查询位置 |
| 验证码 | `captcha.Generate/Validate` | CSPRNG 和比较 | 用户输入/存储值 → 验证结果 |

## 错误与观测资产

| 资产 | 入口 | 关键能力 | 信任边界 |
| --- | --- | --- | --- |
| `Error` 系统 | builder/utils/validator | code/context/error chain | 底层错误 → 日志/响应 |
| `RichError` | HTTP code/status/cause/stack | 业务响应与内部诊断 | cause/stack → JSON 或日志 |

## 通用与发布资产

1. `slice`、`ptr`、`useragent` 公共 API。
2. 根包 `Version()`、Git tag 和 README 版本事实。
3. `go.mod`/`go.sum` 依赖供应链。
4. 当前缺失的 CI、CHANGELOG、LICENSE 和项目级执行约束。
