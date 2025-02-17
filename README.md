# Utils-pkg

这是一个功能丰富的 Go 工具函数包，提供了 JWT、加密、URL 处理、User-Agent 解析和切片操作等常用功能的标准实现。本项目注重安全性、易用性和性能，适用于各类 Go 项目的开发。

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
- golang.org/x/exp（仅 slice 模块需要）

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

### User-Agent 模块 (`useragent/`)

提供高效的浏览器 User-Agent 解析功能：

#### 核心功能
- 快速识别浏览器类型
- 提取浏览器版本信息
- 区分爬虫和真实用户
- 高性能缓存机制

#### 使用场景
- 浏览器兼容性检测
- 爬虫识别和管理
- 访问统计分析
- 设备适配优化

#### 性能优化
- 预编译正则表达式
- 使用 sync.RWMutex 实现并发安全的缓存
- 常见标识符快速查找

### 切片操作模块 (`slice/`)

提供泛型支持的切片操作工具：

#### 核心功能
- 元素查找和包含判断
- 切片去重操作
- 集合运算（交集、并集、差集）
- 函数式操作（Filter、Map、Reduce）

#### 使用场景
- 数据处理和转换
- 集合运算
- 数据过滤和映射
- 列表去重和合并

#### 性能优化
- 使用泛型减少代码重复
- 预分配内存优化性能
- 高效的 map 实现去重

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

### User-Agent 示例
```go
package main

import (
    "fmt"
    "utils-pkg/useragent"
)

func main() {
    // 示例 User-Agent 字符串
    ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

    // 快速检查是否为浏览器请求
    if useragent.IsBrowser(ua) {
        fmt.Println("这是一个浏览器请求")
    }

    // 获取浏览器详细信息
    info := useragent.GetBrowserInfo(ua)
    fmt.Printf("浏览器名称: %s\n", info.Name)
    fmt.Printf("浏览器版本: %s\n", info.Version)

    // 检查是否为爬虫
    botUA := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
    if !useragent.IsBrowser(botUA) {
        fmt.Println("这可能是一个爬虫请求")
    }
}
```

### 切片操作示例
```go
package main

import (
    "fmt"
    "utils-pkg/slice"
)

func main() {
    // 基本切片操作
    numbers := []int{1, 2, 2, 3, 3, 4, 5}
    
    // 去重
    unique := slice.Unique(numbers)
    fmt.Println("去重后:", unique) // [1 2 3 4 5]

    // 检查元素是否存在
    exists := slice.Contains(numbers, 3)
    fmt.Println("包含 3:", exists) // true

    // 集合操作
    set1 := []int{1, 2, 3, 4}
    set2 := []int{3, 4, 5, 6}

    // 交集
    intersection := slice.Intersection(set1, set2)
    fmt.Println("交集:", intersection) // [3 4]

    // 并集
    union := slice.Union(set1, set2)
    fmt.Println("并集:", union) // [1 2 3 4 5 6]

    // 差集
    diff := slice.Difference(set1, set2)
    fmt.Println("差集:", diff) // [1 2]

    // 函数式操作
    // 过滤偶数
    even := slice.Filter(numbers, func(n int) bool {
        return n%2 == 0
    })
    fmt.Println("偶数:", even) // [2 2 4]

    // 映射操作：每个数乘以 2
    doubled := slice.Map(numbers, func(n int) int {
        return n * 2
    })
    fmt.Println("加倍:", doubled) // [2 4 4 6 6 8 10]

    // 归约操作：求和
    sum := slice.Reduce(numbers, 0, func(acc int, n int) int {
        return acc + n
    })
    fmt.Println("总和:", sum) // 20
}