# utils-pkg

面向 Go 1.24+ 业务工程的基础工具库，遵循**构造期快速失败（Fail-Fast）**、**标准错误链（`%w` + `errors.Is` / `errors.As`）**与**安全默认边界**原则。

> 版本状态：`Version()` = `v0.4.0`（技术候选；在许可证 T06-06 未最终确认前不作公开发布声明）。

## 模块概览

| 模块 | 职责与核心安全边界 | 文档 |
| :--- | :--- | :--- |
| [`jwt`](jwt/) | HS256 令牌签发、归一化业务身份 `Identity`、Access/Refresh 严格用途隔离、原子 Refresh Rotation 防重放、Token/Session 双级可插拔吊销、O(1) 受限验证缓存 | [jwt/使用说明.md](jwt/使用说明.md) |
| [`crypto`](crypto/) | AES-GCM 认证加密（支持 AAD 上下文绑定）、Argon2id / scrypt / bcrypt 密码哈希（KDF 计算前校验参数上界）、并发安全密码策略 | [crypto/使用说明.md](crypto/使用说明.md) |
| [`errors`](errors/) | 内外分离的 `RichError`（JSON 仅输出 `code`/`msg`，日志 `%+v` 输出根因与调用栈），完整兼容标准库 `errors.Is` / `errors.As` | [errors/README.md](errors/README.md) |
| [`url`](url/) | 绑定 HTTP Method / Scheme / Host / Path / Query 的 v2 HMAC-SHA256 签名 URL，32 字节密钥下限，强制 `_exp` 过期控制 | [url/使用说明.md](url/使用说明.md) |
| [`pagination`](pagination/) | 游标分页（`Base64JSONCodec` / 防篡改 `HMACCodec`）与防溢出偏移量分页（`OffsetRequest` / `OffsetResponse`） | [pagination/使用说明.md](pagination/使用说明.md) |
| [`captcha`](captcha/) | 基于 `crypto/rand` 的验证码生成与恒定时间比对（库不持有 TTL、限流与消费状态，空值永不通过） | [captcha/使用说明.md](captcha/使用说明.md) |
| [`ptr`](ptr/) | 泛型指针构造与安全解引用（`Of`、`Value`、`Deref`、`Zero`）及基础类型便捷函数 | [ptr/使用说明.md](ptr/使用说明.md) |
| [`slice`](slice/) | 保持顺序且不共享底层数组的泛型切片集合运算（`Unique`、`Union`、`Intersection`、`Difference`、`Filter`、`Map`、`Reduce`、`Sort`） | [slice/使用说明.md](slice/使用说明.md) |
| [`useragent`](useragent/) | 启发式浏览器名称/版本提取与受限分片 LRU 缓存（仅用于展示/统计，**不可用于安全或权限判定**） | [useragent/使用说明.md](useragent/使用说明.md) |

完整破坏性变更与升级指引请参阅 [MIGRATION.md](MIGRATION.md) 与 [CHANGELOG.md](CHANGELOG.md)。

---

## 安装

```bash
go get github.com/iwen-conf/utils-pkg
```

---

## 核心模块快速上手

### 1. JWT 认证生命周期（`jwt`）

```go
package main

import (
	"context"
	"fmt"

	"github.com/iwen-conf/utils-pkg/jwt"
)

func main() {
	// 密钥至少 32 字节（生产环境请从 KMS 或环境变量加载高熵密钥）
	tm, err := jwt.NewTokenManager("0123456789abcdef0123456789abcdef")
	if err != nil {
		panic(err)
	}
	defer tm.Shutdown()

	ctx := context.Background()

	// 1. 登录：基于归一化业务身份签发 Access + Refresh 令牌对
	pair, err := tm.IssueTokenPair(jwt.Identity{
		Subject:   "user-1001",
		ActorType: "user",
		TenantID:  "tenant-acme",
		Roles:     []string{"editor"},
		Scopes:    []string{"docs:write"},
	})
	if err != nil {
		panic(err)
	}

	// 2. 鉴权：一步验证 AccessToken（自动拒绝 RefreshToken）并还原 Identity
	identity, _, err := tm.Authenticate(ctx, pair.AccessToken)
	if err != nil {
		panic(err)
	}
	fmt.Println("当前用户:", identity.Subject, identity.ActorType)

	// 3. 刷新：一次性消费旧 RefreshToken 并轮换出新令牌对（重放旧令牌将自动封禁整个会话）
	rotated, err := tm.RefreshTokenPair(ctx, pair.RefreshToken)
	if err != nil {
		panic(err)
	}

	// 4. 注销：按会话 ID 或令牌吊销整个登录会话
	if err := tm.RevokeSessionByID(ctx, rotated.SessionID); err != nil {
		panic(err)
	}
}
```

### 2. AES-GCM 认证加密与 Argon2id 密码哈希（`crypto`）

```go
package main

import (
	"fmt"

	"github.com/iwen-conf/utils-pkg/crypto"
)

func main() {
	key, err := crypto.GenerateRandomBytes(32)
	if err != nil {
		panic(err)
	}

	enc, err := crypto.NewAESEncryptor(key)
	if err != nil {
		panic(err)
	}

	// 使用 AAD 将密文绑定到具体用户记录，防止密文跨行替换
	aad := []byte("user:1001")
	ct, err := enc.EncryptWithAAD([]byte("sensitive-payload"), aad)
	if err != nil {
		panic(err)
	}
	pt, err := enc.DecryptWithAAD(ct, aad)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(pt))

	// Argon2id 密码哈希与校验
	hasher, err := crypto.NewArgon2Hasher(crypto.DefaultArgon2Params())
	if err != nil {
		panic(err)
	}
	hash, err := hasher.Hash([]byte("Str0ng!Passphrase"))
	if err != nil {
		panic(err)
	}
	ok, err := hasher.Verify([]byte(hash), []byte("Str0ng!Passphrase"))
	fmt.Println("密码匹配:", ok, err)
}
```

### 3. 内外有别的业务错误处理（`errors`）

```go
package main

import (
	"encoding/json"
	stderrors "errors"
	"fmt"

	"github.com/iwen-conf/utils-pkg/errors"
)

func main() {
	dbErr := stderrors.New("dial tcp 10.0.0.8:5432: i/o timeout")
	err := errors.WrapRich(dbErr, errors.RichCodeDBError, "查询账户失败")

	// 对外响应：仅序列化 code 与 msg，绝不泄露底层数据库地址或根因
	body, _ := json.Marshal(err)
	fmt.Printf("HTTP=%d JSON=%s\n", err.HTTPStatus(), body)

	// 对内诊断：兼容标准 errors.Is，且 %+v 输出完整根因与调用栈
	fmt.Println("is dbErr:", errors.Is(err, dbErr))
}
```

### 4. 泛型指针与验证码辅助（`ptr` / `captcha`）

```go
nick := ptr.Of("Alice")
fmt.Println(ptr.Deref(nick, "Guest"))

code, err := captcha.Generate6()
if err == nil && captcha.Validate(code, code) {
	fmt.Println("验证码校验通过（请务必在业务侧配合短 TTL、尝试次数限制与验证后立即消费）")
}
```

---

## 质量门禁

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go mod verify
go mod tidy -diff
bash .github/scripts/check-coverage.sh
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
```
