# Utils-pkg

这是一个功能丰富的 Go 工具函数包，提供了 JWT、加密和 URL 处理等常用功能的标准实现。本项目注重安全性、易用性和性能，适用于各类 Go 项目的开发。

## 特性

- 模块化设计，可按需引入
- 完整的单元测试覆盖
- 详细的文档和示例
- 注重安全性的实现
- 高性能设计
- 无第三方依赖（除 JWT 模块外）

## 依赖要求

- Go 1.21 或更高版本
- github.com/golang-jwt/jwt/v5（仅 JWT 模块需要）

## 安装

```bash
go get github.com/iwen-conf/utils-pkg
```

## 功能模块

### JWT 模块 (`jwt/`)

提供完整的 JWT（JSON Web Token）解决方案：

#### 核心功能
- Token 的生成和验证
- 自定义 Claims 支持
- Token 黑名单机制
- 多种签名算法支持（HS256、RS256等）

#### 使用场景
- 用户身份认证
- API 接口鉴权
- 分布式系统的会话管理
- 单点登录（SSO）实现

#### 性能优化
- 使用 sync.RWMutex 实现高效的黑名单并发控制
- 支持批量清理过期黑名单记录
- 验证器链式调用，提高扩展性

### 加密模块 (`crypto/`)

提供全面的加密和哈希功能：

#### 核心功能
- AES-128/256 对称加密/解密
- 多种哈希算法支持（SHA256、SHA512、MD5）
- 基于 bcrypt 的密码加密
- 密码策略验证
- 安全随机数生成

#### 使用场景
- 敏感数据加密存储
- 密码安全存储
- 数据完整性校验
- 安全令牌生成

#### 性能优化
- 复用 AES 密码块
- 使用 CFB 模式提高加密效率
- 支持批量数据处理

### URL 模块 (`url/`)

提供安全可靠的 URL 处理功能：

#### 核心功能
- URL 构建和解析
- 参数签名验证
- 防重放攻击
- 复杂数据类型序列化

#### 使用场景
- API 接口安全
- 支付链接生成
- 分享链接生成
- 防篡改URL构建

#### 性能优化
- 参数预排序
- 复用 URL 解析器
- 高效的签名算法

## 详细使用示例

### JWT 示例
```go
package main

import (
    "fmt"
    "time"
    "utils-pkg/jwt"
)

func main() {
    // 创建 JWT 管理器（建议使用 32 字节或更长的密钥）
    jwtManager := jwt.NewJWTManager("your-secret-key-at-least-32-bytes-long", 24*time.Hour)

    // 添加自定义声明验证器（支持多个验证器）
    jwtManager.AddValidator(func(claims *jwt.Claims) error {
        if claims.Extra["role"] != "admin" {
            return fmt.Errorf("unauthorized: requires admin role")
        }
        return nil
    })

    // 生成包含自定义声明的 token
    token, err := jwtManager.GenerateToken("12345", "john_doe", map[string]interface{}{
        "role": "admin",
        "permissions": []string{"read", "write"},
    })
    if err != nil {
        panic(err)
    }

    // 验证和解析 token
    claims, err := jwtManager.ValidateToken(token)
    if err != nil {
        panic(err)
    }

    // 将 token 加入黑名单（如用户注销时）
    err = jwtManager.AddToBlacklist(token, time.Now().Add(24*time.Hour))
    if err != nil {
        panic(err)
    }

    // 定期清理过期的黑名单记录（建议通过定时任务执行）
    jwtManager.CleanBlacklist()
}
```

### 加密示例
```go
package main

import (
    "fmt"
    "utils-pkg/crypto"
)

func main() {
    // AES-256 加密示例（推荐使用 32 字节密钥）
    key := []byte("your-secret-key-must-be-32-bytes!!")
    encryptor, err := crypto.NewAESEncryptor(key)
    if err != nil {
        panic(err)
    }

    // 加密数据
    plaintext := []byte("Hello, World!")
    ciphertext, err := encryptor.Encrypt(plaintext)
    if err != nil {
        panic(err)
    }

    // 解密数据
    decrypted, err := encryptor.Decrypt(ciphertext)
    if err != nil {
        panic(err)
    }

    // 密码策略验证
    policy := crypto.NewDefaultPasswordPolicy()
    password := "MyStr0ng!Pass@2024"
    if err := policy.ValidatePassword(password); err != nil {
        panic(err)
    }

    // 密码安全存储（使用 bcrypt）
    hashedPassword, err := crypto.HashPassword([]byte(password))
    if err != nil {
        panic(err)
    }

    // 密码验证
    err = crypto.CompareHashAndPassword(hashedPassword, []byte(password))
    if err != nil {
        fmt.Println("密码不匹配")
        return
    }

    // 安全哈希计算
    data := []byte("需要计算哈希的数据")
    sha256Hash := crypto.HashSHA256(data)
    sha512Hash := crypto.HashSHA512(data)
    // 不推荐在安全场景使用 MD5
    md5Hash := crypto.HashMD5(data)
}
```

### URL 示例
```go
package main

import (
    "fmt"
    "time"
    "utils-pkg/url"
)

func main() {
    // 创建安全的 URL 构建器
    secretKey := "your-url-signing-secret-key"
    builder := url.NewURLBuilder("https://api.example.com/users", secretKey)

    // 添加查询参数
    builder.AddParam("page", "1")
    builder.AddParam("limit", "10")
    builder.AddParam("sort", "name")

    // 添加多值参数
    builder.AddParam("tag", "golang")
    builder.AddParam("tag", "utils")

    // 启用参数排序（用于签名验证）
    builder.SetSortParams(true)

    // 设置防重放时间戳
    builder.SetTimestamp(time.Now().Unix())

    // 构建签名 URL
    fullURL, err := builder.Build()
    if err != nil {
        panic(err)
    }

    // 验证 URL 签名（1小时有效期）
    isValid, err := url.ValidateSignature(fullURL, secretKey, 3600)
    if err != nil {
        panic(err)
    }

    // 解析 URL 组成部分
    parsedURL, err := url.ParseURL(fullURL)
    if err != nil {
        panic(err)
    }

    // 复杂参数序列化示例
    params := map[string]interface{}{
        "id": "123",
        "tags": []string{"go", "utils"},
        "filter": map[string]string{"status": "active"},
        "range": map[string]interface{}{
            "start": 0,
            "end": 100,
        },
    }
    queryString := url.SerializeParams(params)

    // 参数反序列化
    deserializedParams := url.DeserializeParams(queryString)
}
```

## 安全性建议

### JWT 安全
- 使用足够长的密钥（至少 32 字节）
- 设置合理的过期时间（建议 24 小时内）
- 敏感操作必须验证 token
- 使用 HTTPS 传输
- 定期清理黑名单
- 使用自定义验证器控制权限

### 加密安全
- 优先使用 AES-256
- 定期轮换密钥
- 避免使用 MD5
- 安全存储密钥
- 强制密码策略
- 使用 bcrypt 存储密码
- 使用密码学安全的随机数

### URL 安全
- URL 参数必须编码
- 验证参数合法性
- 处理特殊字符
- 注意长度限制
- 使用签名防篡改
- 合理设置时间戳有效期
- 加密敏感参数

## 性能优化

### 通用建议
- 复用对象和连接
- 使用适当的缓存策略
- 避免不必要的内存分配
- 合理设置并发控制

### 模块优化
1. JWT 模块
   - 使用 sync.Pool 缓存 token 解析器
   - 定期批量清理黑名单
   - 使用读写锁优化并发

2. 加密模块
   - 复用 AES 密码块
   - 预分配缓冲区
   - 使用 CFB 模式提升性能

3. URL 模块
   - 预排序参数提升签名效率
   - 复用 URL 解析器
   - 优化序列化性能

## 常见问题

### JWT 相关
1. Token 过期处理
   - 实现令牌刷新机制
   - 提前通知客户端刷新
   - 使用滑动过期时间

2. 并发控制
   - 使用读写锁控制黑名单访问
   - 定期清理过期记录
   - 合理设置缓存大小

### 加密相关
1. 密钥管理
   - 使用密钥管理服务
   - 定期轮换密钥
   - 安全存储密钥

2. 性能问题
   - 选择合适的加密模式
   - 复用加密对象
   - 使用适当的缓冲区大小

### URL 相关
1. 参数处理
   - 正确处理特殊字符
   - 注意 URL 长度限制
   - 处理多值参数

2. 签名验证
   - 时间戳偏差处理
   - 参数顺序一致性
   - 处理 URL 编码问题

## 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

### 贡献要求

- 遵循项目代码风格
- 添加完整的单元测试
- 更新相关文档
- 遵循语义化版本规范

## 许可证

MIT License

## 问题反馈

如果你发现任何问题或有改进建议，欢迎：

1. 提交 Issue
2. 发送 Pull Request
3. 联系维护者