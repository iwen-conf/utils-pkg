# Data Map

本仓库不持久化业务数据，因此 data yield 以“库能影响或暴露的数据类别/能力”评估；实际行数取决于下游服务。

| 数据/能力 | 标签 | 入口/载体 | 可能产量 | 说明 |
| --- | --- | --- | --- | --- |
| access/refresh token、sid、jti | AUTH | `jwt/` | L2–L3 capability | 缓存或重放问题可影响账户会话完整性 |
| roles/scopes/tenant/custom claims | AUTH/BIZ/PII | `StandardClaims.CustomClaims` | L1–L3 | 下游可能放入身份、租户或个人字段 |
| JWT/HMAC/AES 密钥 | SECRET | 构造器参数 | L2 capability | 库不应记录或回显原值 |
| 密码与密码 hash | AUTH | `crypto/` | L2 | KDF 资源和策略错误影响认证可用性与强度 |
| 签名 URL 查询与目标资源 | BIZ/AUTH | `url/` | L1–L3 capability | 未绑定 path/method 时可跨资源复用签名 |
| 分页 cursor payload | BIZ/PII | `pagination/` | L1–L2 | payload 可包含记录定位信息 |
| 验证码 | AUTH | `captcha/` | capability only | 空值匹配可绕过下游验证分支 |
| error cause/context/stack | SECRET/PII/BIZ | `errors/` | L1–L2 | 具体内容依赖下游数据库、RPC 和用户上下文 |
| User-Agent | PII-like telemetry | `useragent/` | L1 | 本库仅做内存缓存，不应升级为安全身份 |
