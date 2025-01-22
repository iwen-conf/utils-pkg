# Utils-pkg

这是一个通用的 Go 工具函数包，提供了常用的工具函数和实用程序。

## 功能模块

### JWT 模块 (`jwt/`)

处理 JWT（JSON Web Token）相关的功能：
- Token 的生成和验证
- Token 的解析和校验
- 自定义 Claims 支持

### 加密模块 (`crypto/`)

提供各种加密和哈希功能：
- 对称加密/解密
- 哈希计算
- 安全随机数生成

### URL 模块 (`url/`)

处理 URL 相关的功能：
- URL 序列化
- URL 反序列化
- 参数处理

## 安装

```bash
go get github.com/your-username/utils-pkg
```

## 使用示例

### JWT 示例
```go
package main

import (
    "fmt"
    "time"
    "utils-pkg/jwt"
)

func main() {
    // 创建 JWT 管理器
    jwtManager := jwt.NewJWTManager("your-secret-key", 24*time.Hour)

    // 生成 token
    userID := "12345"
    username := "john_doe"
    extra := map[string]interface{}{
        "role": "admin",
    }
    
    token, err := jwtManager.GenerateToken(userID, username, extra)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Generated token: %s\n", token)

    // 验证和解析 token
    claims, err := jwtManager.ValidateToken(token)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Validated token claims: %+v\n", claims)
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
    // 创建 AES 加密器（密钥长度必须是 16、24 或 32 字节）
    key := []byte("0123456789abcdef") // 16 字节的密钥
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
    fmt.Printf("加密后的数据: %s\n", ciphertext)

    // 解密数据
    decrypted, err := encryptor.Decrypt(ciphertext)
    if err != nil {
        panic(err)
    }
    fmt.Printf("解密后的数据: %s\n", string(decrypted))

    // 计算 SHA256 哈希
    hash := crypto.HashSHA256(plaintext)
    fmt.Printf("SHA256 哈希值: %x\n", hash)

    // 生成随机字节
    randomBytes, err := crypto.GenerateRandomBytes(16)
    if err != nil {
        panic(err)
    }
    fmt.Printf("随机字节: %x\n", randomBytes)
}
```

### URL 示例
```go
package main

import (
    "fmt"
    "utils-pkg/url"
)

func main() {
    // 使用 URLBuilder 构建 URL
    builder := url.NewURLBuilder("https://api.example.com/users")
    builder.AddParam("page", "1")
    builder.AddParam("limit", "10")
    builder.AddParam("sort", "name")
    builder.SetFragment("top")

    fullURL, err := builder.Build()
    if err != nil {
        panic(err)
    }
    fmt.Printf("构建的 URL: %s\n", fullURL)

    // 解析 URL
    parsedURL, err := url.ParseURL(fullURL)
    if err != nil {
        panic(err)
    }
    fmt.Printf("解析的 URL 组成部分: %+v\n", parsedURL)

    // 序列化参数
    params := map[string]interface{}{
        "id": "123",
        "tags": []string{"go", "utils"},
        "filter": map[string]string{"status": "active"},
    }
    queryString := url.SerializeParams(params)
    fmt.Printf("序列化的查询参数: %s\n", queryString)

    // 反序列化参数
    deserializedParams := url.DeserializeParams(queryString)
    fmt.Printf("反序列化的参数: %+v\n", deserializedParams)
}
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License