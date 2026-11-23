# v0.5.0 升级与 API 迁移指南

本文档汇总 `utils-pkg` 从 `v0.3.0` 升级到 `v0.5.0` 的兼容性影响、修改前后代码对比及迁移步骤。`v0.4.0` 已包含安全破坏性变更；`v0.5.0` 收敛 JWT 生命周期 API，并补充 AES-GCM AAD 与泛型 `ptr` 辅助函数。

---

## 一、JWT 模块迁移（重点）

### 1. 关键安全行为变化

1. **通用验证接口不再接受 `RefreshToken`（防止类型混淆越权）**
   - `ValidateToken`、`ValidateTokenContext`、`ValidateAccessToken`、`Authenticate` 现**仅接受 `TokenType == AccessToken`**。
   - 若业务中间件此前调用 `ValidateToken(token)` 且未手动判断 `claims.TokenType`，升级后客户端携带 `RefreshToken` 访问普通业务接口将被自动拦截并返回 `jwt.ErrTokenTypeMismatch`。
   - 若测试或自省逻辑确需解析验证 `RefreshToken`，请改为调用 `tm.ValidateRefreshToken(ctx, refreshToken)`。
2. **`CustomClaims` 禁止覆盖 JWT 保留声明**
   - 自定义声明不得包含 `iss`、`sub`、`aud`、`exp`、`nbf`、`iat`、`jti`、`type`、`sid`，否则签发返回 `jwt.ErrReservedClaimKey`。请通过 `TokenOptions`（如 `Issuer`、`Audience`、`SessionID`、`TokenID`）传递标准字段。
3. **验证期望与签发默认值解耦**
   - `JWTOptions.Issuer` / `Audience` 仅作为签发默认值；如需在验证时强制校验 `iss` / `aud`，请配置 `JWTOptions.ExpectedIssuer` / `ExpectedAudience`。

### 2. 修改前后调用对比

**修改前（繁琐且易误用）：**

```go
// 登录：需手动维护 SessionID、分两次签发，且容易用非密码学随机数
sessionID := fmt.Sprintf("sid-%d", time.Now().UnixNano())
accessToken, _ := tm.GenerateToken(uid, &jwt.TokenOptions{
	TokenType: jwt.AccessToken,
	SessionID: sessionID,
})
refreshToken, _ := tm.GenerateToken(uid, &jwt.TokenOptions{
	TokenType: jwt.RefreshToken,
	SessionID: sessionID,
})

// 鉴权：需手动检查 TokenType，且使用自定义 TokenProfile 时每次都要重复传参
claims, err := tm.ValidateTokenContext(ctx, accessToken)
if err != nil || claims.TokenType != jwt.AccessToken {
	return err
}
identity := claims.Identity(customProfile)

// 刷新：只返回两个字符串，拿不到过期秒数与会话 ID
newAccess, newRefresh, err := tm.RefreshTokenContext(ctx, refreshToken)
```

**修改后（推荐）：**

```go
// 登录：一步签发共享 CSPRNG SessionID 的标准令牌对
pair, err := tm.IssueTokenPair(jwt.Identity{
	Subject:   uid,
	ActorType: "user",
	Roles:     []string{"member"},
})

// 鉴权：强制要求 AccessToken，自动按管理器 TokenProfile 提取业务身份
identity, claims, err := tm.Authenticate(ctx, pair.AccessToken)

// 刷新：返回完整 *TokenPair（含新 AccessToken、新 RefreshToken、SessionID、ExpiresIn）
newPair, err := tm.RefreshTokenPair(ctx, pair.RefreshToken)

// 注销：支持按令牌注销，也支持后台直接按 SessionID 踢下线（无需保存原始 JWT）
err = tm.RevokeSessionByID(ctx, newPair.SessionID)
```

> **兼容性保证**：原有的 `GenerateToken`、`GenerateTokenPair`、`GenerateIdentityToken`、`GenerateIdentityTokenPair`、`ValidateToken`、`ValidateTokenContext`、`RefreshToken`、`RefreshTokenContext`、`RevokeToken`、`RevokeSession` 签名均完整保留。

---

## 二、Crypto 模块迁移

1. **AES 加密仅支持 `ModeGCM`**：
   - `NewAESEncryptor(key)` 直接创建并发安全的 AES-GCM 实例；`NewAESEncryptorWithMode(key, crypto.ModeCFB)` 直接返回 `crypto.ErrUnsupportedEncryptionMode`。
   - 新增 `EncryptWithAAD` / `DecryptWithAAD`，建议将记录主键或用户 ID 作为 AAD 传入以防止密文跨记录替换。
2. **`PasswordPolicy` 改为不可变配置快照**：
   - 修改前：`policy.MinLength = 10`（存在并发数据竞争，已移除公开可变字段）。
   - 修改后：
     ```go
     cfg := crypto.DefaultPasswordPolicyConfig()
     cfg.MinLength = 10
     policy, err := crypto.NewPasswordPolicy(cfg)
     ```
3. **哈希器构造函数返回 `(Hasher, error)`**：
   - `NewArgon2Hasher(params)`、`NewScryptHasher(params)`、`NewBcryptHasherChecked(cost)` 均在构造期校验参数并返回 `error`。

---

## 三、Errors 模块移除

- `github.com/iwen-conf/utils-pkg/errors`（包含 `RichError` 以及旧版 `errors.Error` / `Builder` / `Validator`）已于 `v0.5.1` 之后正式移除。
- 通用错误链与错误判断请直接使用 Go 标准库 `errors`（`fmt.Errorf("%w", ...)`、`errors.Is`、`errors.As`）。
- 业务状态码、面向客户端的安全文案等应用层错误契约建议在各业务仓库内部以轻量包（如 `internal/fault`）自行定义，不再由 `utils-pkg` 承载。

---

## 四、URL / Pagination / Ptr / UserAgent 迁移要点

- **URL**：签名升级为 `_sigv=2`，密钥至少 32 字节；非 GET 请求请使用 `SetMethod` 与 `ValidateSignatureWithMethod`。
- **Pagination**：`NewHMACCodec(key)` 在密钥为空或不足 32 字节时返回 `(nil, error)`，不再降级运行。
- **Ptr**：新增泛型 `ptr.Of(v)`、`ptr.Deref(p, fallback)`、`ptr.Zero(p)`、`ptr.Value(p)`，原有 `ptr.String` / `ptr.SafeString` / `ptr.ValueString` 等全部保留。
- **UserAgent**：`NewLRUCache(capacity, ttl)` 与 `NewShardedCache(shardCapacity, ttl)` 的 `ttl` 参数类型为 `time.Duration`（如 `30 * time.Minute`），构造期校验参数并返回 `error`。
