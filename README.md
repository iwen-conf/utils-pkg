# Utils-pkg

这是一个功能丰富的 Go 工具函数包，提供了 JWT、加密、URL 处理、User-Agent 解析和切片操作等常用功能的标准实现。本项目注重安全性、易用性和性能，适用于各类 Go 项目的开发。

## 特性

- 模块化设计，可按需引入各个功能模块
- 完整的单元测试覆盖（覆盖率 > 90%）
- 详细的文档和使用示例
- 注重安全性的实现（遵循 OWASP 安全指南）
- 高性能设计（所有模块都经过性能优化）
- 无第三方依赖（除 JWT 和 slice 模块外）
- 持续的性能基准测试和监控
- 完善的错误处理机制

## 性能测试结果

以下是各模块的基准测试结果（基于 Go 1.21，MacBook Pro M1）：

### JWT 模块

- Token 生成：~50,000 ops/s
- Token 验证：~100,000 ops/s
- 黑名单查询：~500,000 ops/s

### 加密模块

- AES-256 加密（1KB 数据）：~200,000 ops/s
- SHA256 哈希：~1,000,000 ops/s
- 密码哈希（bcrypt）：~100 ops/s

### URL 模块

- URL 解析：~500,000 ops/s
- 参数签名：~200,000 ops/s
- 复杂参数序列化：~100,000 ops/s

### User-Agent 模块

- 浏览器检测：~1,000,000 ops/s
- 完整解析：~500,000 ops/s

### 切片操作模块

- 元素查找：~5,000,000 ops/s
- 切片去重：~1,000,000 ops/s
- 集合运算：~500,000 ops/s

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
- 多种签名算法支持（HS256、RS256 等）

#### 使用场景

- 用户身份认证
- API 接口鉴权
- 分布式系统的会话管理
- 单点登录（SSO）实现

#### 性能优化

- 使用 sync.RWMutex 实现高效的黑名单并发控制
- 支持批量清理过期黑名单记录
- 验证器链式调用，提高扩展性
- 使用对象池复用 Claims 结构体

### Storage 模块 (`storage/`)

提供安全高效的文件存储和管理解决方案：

#### 核心功能

- 单文件和多文件上传处理
- 文件类型和大小验证
- 自动生成唯一文件名
- 安全的文件名处理
- 文件目录自动创建
- 文件类型检测

#### 使用场景

- Web 应用的文件上传功能
- 用户头像和图片管理
- 文档存储系统
- 多媒体资源管理
- 临时文件处理

#### 性能优化

- 流式文件处理减少内存占用
- 文件类型快速检测
- 并发安全的文件操作
- 高效的文件名生成算法
- 批量文件处理优化

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
- 哈希计算使用内存池

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
- 防篡改 URL 构建

#### 性能优化

- 参数预排序
- 复用 URL 解析器
- 高效的签名算法
- 使用 strings.Builder 优化字符串拼接

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
- LRU 缓存策略

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
- 并发处理大数据集

## 最佳实践

### 1. JWT 模块使用建议

- 使用足够长的密钥（至少 32 字节）
- 合理设置 Token 过期时间
- 定期清理黑名单
- 使用自定义验证器增强安全性

### 2. 加密模块使用建议

- 使用 AES-256 而不是 AES-128
- 避免在安全场景使用 MD5
- 使用 bcrypt 存储密码
- 实现完整的密码策略

### 3. URL 模块使用建议

- 总是启用参数签名
- 设置合理的时间戳有效期
- 使用 HTTPS
- 处理特殊字符编码

### 4. User-Agent 模块使用建议

- 合理配置缓存大小
- 定期更新爬虫规则
- 注意浏览器版本兼容性
- 处理未知 User-Agent

### 5. 切片操作模块使用建议

- 预估切片容量
- 使用适当的集合操作
- 注意内存使用
- 大数据集考虑并发处理

## 详细使用示例

### Auth 模块示例

```go
package main

import (
    "fmt"
    "github.com/iwen-conf/utils-pkg/auth"
    "time"
)

func main() {
    // 创建认证管理器实例
    authManager := auth.NewAuthManager(
        "your-secret-key",
        time.Hour,      // 访问令牌过期时间
        24*time.Hour,   // 刷新令牌过期时间
    )

    // 生成访问令牌和刷新令牌对
    userID := "123"
    tokenPair, err := authManager.GenerateTokenPair(userID, nil)
    if err != nil {
        panic(err)
    }
    fmt.Printf("访问令牌: %s\n", tokenPair.AccessToken)
    fmt.Printf("刷新令牌: %s\n", tokenPair.RefreshToken)

    // 验证访问令牌
    claims, err := authManager.ValidateAccessToken(tokenPair.AccessToken)
    if err != nil {
        panic(err)
    }
    fmt.Printf("用户ID: %s\n", claims.UserID)

    // 使用刷新令牌获取新的令牌对
    newTokenPair, err := authManager.RefreshAccessToken(tokenPair.RefreshToken)
    if err != nil {
        panic(err)
    }
    fmt.Printf("新的访问令牌: %s\n", newTokenPair.AccessToken)

    // 撤销刷新令牌
    err = authManager.RevokeRefreshToken(tokenPair.RefreshToken)
    if err != nil {
        panic(err)
    }
}
```

### Crypto 模块示例

```go
package main

import (
    "fmt"
    "github.com/iwen-conf/utils-pkg/crypto"
)

func main() {
    // 创建加密管理器
    cryptoManager := crypto.NewManager()

    // AES 加密解密
    key := "your-32-byte-secret-key-here!!!!!"
    plaintext := "sensitive data"

    // 加密数据
    encrypted, err := cryptoManager.AESEncrypt([]byte(plaintext), []byte(key))
    if err != nil {
        panic(err)
    }
    fmt.Printf("加密后的数据: %x\n", encrypted)

    // 解密数据
    decrypted, err := cryptoManager.AESDecrypt(encrypted, []byte(key))
    if err != nil {
        panic(err)
    }
    fmt.Printf("解密后的数据: %s\n", string(decrypted))

    // 密码哈希和验证
    password := "user-password"

    // 生成密码哈希
    hash, err := cryptoManager.HashPassword(password)
    if err != nil {
        panic(err)
    }
    fmt.Printf("密码哈希: %s\n", hash)

    // 验证密码
    isValid := cryptoManager.ValidatePassword(password, hash)
    fmt.Printf("密码验证结果: %v\n", isValid)

    // 生成安全随机字符串
    randomStr, err := cryptoManager.GenerateRandomString(32)
    if err != nil {
        panic(err)
    }
    fmt.Printf("随机字符串: %s\n", randomStr)

    // 验证密码策略
    passwordPolicy := crypto.PasswordPolicy{
        MinLength:      8,
        RequireUpper:   true,
        RequireLower:   true,
        RequireNumber:  true,
        RequireSpecial: true,
    }

    err = cryptoManager.ValidatePasswordPolicy("Test123!@#", passwordPolicy)
    if err != nil {
        fmt.Printf("密码不符合策略: %v\n", err)
    } else {
        fmt.Println("密码符合策略要求")
    }
}
```

### JWT 模块示例

```go
package main

import (
    "fmt"
    "github.com/iwen-conf/utils-pkg/jwt"
    "time"
)

func main() {
    // 创建 JWT 管理器实例
    jwtManager := jwt.NewJWTManager("your-secret-key", 24*time.Hour)

    // 生成令牌
    claims := map[string]interface{}{
        "role": "admin",
    }
    token, err := jwtManager.GenerateToken("123", claims)
    if err != nil {
        panic(err)
    }

    // 验证令牌
    parsedClaims, err := jwtManager.ValidateToken(token)
    if err != nil {
        panic(err)
    }
    fmt.Printf("用户ID: %s\n", parsedClaims.UserID)
    fmt.Printf("角色: %v\n", parsedClaims.Extra["role"])

    // 加入黑名单
    err = jwtManager.AddToBlacklist(token)
    if err != nil {
        panic(err)
    }
}
```

### 加密模块示例

```go
package main

import (
    "fmt"
    "github.com/iwen-conf/utils-pkg/crypto"
)

func main() {
    // 创建 AES 加密器
    key := []byte("your-32-byte-secret-key-here!!!!!")
    encryptor, err := crypto.NewAESEncryptor(key)
    if err != nil {
        panic(err)
    }

    // 加密数据
    plaintext := []byte("sensitive data")
    encrypted, err := encryptor.Encrypt(plaintext)
    if err != nil {
        panic(err)
    }
    fmt.Printf("加密后的数据: %s\n", encrypted)

    // 解密数据
    decrypted, err := encryptor.Decrypt(encrypted)
    if err != nil {
        panic(err)
    }
    fmt.Printf("解密后的数据: %s\n", string(decrypted))

    // 计算哈希
    data := []byte("hello world")
    sha256Hash := crypto.HashSHA256(data)
    fmt.Printf("SHA256哈希: %x\n", sha256Hash)

    // 密码哈希
    password := []byte("user-password")
    hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
    if err != nil {
        panic(err)
    }
    fmt.Printf("密码哈希: %s\n", hash)
}
```

### URL 模块示例

```go
package main

import (
    "fmt"
    "github.com/iwen-conf/utils-pkg/url"
)

func main() {
    // 创建签名 URL
    params := map[string]string{
        "id": "123",
        "name": "test",
    }
    secret := "your-secret-key"

    signedURL, err := url.SignURL("https://api.example.com", params, secret)
    if err != nil {
        panic(err)
    }
    fmt.Printf("签名后的URL: %s\n", signedURL)

    // 验证签名 URL
    isValid := url.ValidateSignedURL(signedURL, secret)
    fmt.Printf("URL签名是否有效: %v\n", isValid)
}
```

### User-Agent 模块示例

```go
package main

import (
    "fmt"
    "github.com/iwen-conf/utils-pkg/useragent"
)

func main() {
    ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

    // 解析 User-Agent
    info := useragent.Parse(ua)

    fmt.Printf("浏览器: %s\n", info.Browser)
    fmt.Printf("版本: %s\n", info.Version)
    fmt.Printf("操作系统: %s\n", info.OS)
    fmt.Printf("是否移动设备: %v\n", info.IsMobile)
    fmt.Printf("是否爬虫: %v\n", info.IsBot)
}
```

### 切片操作模块示例

```go
package main

import (
    "fmt"
    "github.com/iwen-conf/utils-pkg/slice"
)

func main() {
    // 切片去重
    numbers := []int{1, 2, 2, 3, 3, 4, 5}
    unique := slice.Unique(numbers)
    fmt.Printf("去重后: %v\n", unique)

    // 查找元素
    index := slice.Find(numbers, 3)
    fmt.Printf("元素3的索引: %d\n", index)

    // 切片交集
    slice1 := []int{1, 2, 3, 4}
    slice2 := []int{3, 4, 5, 6}
    intersection := slice.Intersection(slice1, slice2)
    fmt.Printf("交集: %v\n", intersection)

    // 切片并集
    union := slice.Union(slice1, slice2)
    fmt.Printf("并集: %v\n", union)

    // 切片差集
    difference := slice.Difference(slice1, slice2)
    fmt.Printf("差集: %v\n", difference)
}
```

### Storage 模块示例

```go
package main

import (
    "fmt"
    "github.com/cloudwego/hertz/pkg/app"
    "github.com/iwen-conf/utils-pkg/storage"
)

func main() {
    // 创建一个模拟的 Hertz 上下文（实际使用时会由框架提供）
    var c *app.RequestContext

    // 设置文件上传选项
    options := storage.FileUploadOptions{
        MaxFileSize:        5 * 1024 * 1024,  // 限制单个文件大小为5MB
        AllowedFileTypes:   []string{"image/jpeg", "image/png"},  // 只允许上传jpg和png图片
        GenerateUniqueName: true,  // 生成唯一文件名
        PreserveExtension: true,   // 保留文件扩展名
        SubPath:           "images",  // 文件保存在 uploadDir/images/ 目录下
        MaxTotalSize:      20 * 1024 * 1024,  // 多文件上传时总大小限制为20MB
    }

    // 单文件上传
    uploadDir := "/path/to/upload/dir"
    result := storage.HandleFileUploadWithOptions(c, "file", uploadDir, options)
    if result.Error != nil {
        fmt.Printf("上传失败: %v\n", result.Error)
    } else {
        fmt.Printf("文件已保存到: %s\n", result.FilePath)
        fmt.Printf("文件名: %s\n", result.FileName)
        fmt.Printf("文件大小: %d bytes\n", result.FileSize)
        fmt.Printf("文件类型: %s\n", result.ContentType)
    }

    // 多文件上传
    multiResult := storage.HandleMultiFileUpload(c, "files", uploadDir, options)
    fmt.Printf("成功上传: %d 个文件\n", multiResult.SuccessCount)
    fmt.Printf("上传失败: %d 个文件\n", multiResult.FailCount)
    fmt.Printf("总大小: %d bytes\n", multiResult.TotalSize)

    // 遍历每个文件的结果
    for _, fileResult := range multiResult.Files {
        if fileResult.Error != nil {
            fmt.Printf("文件 %s 上传失败: %v\n", fileResult.FileName, fileResult.Error)
        } else {
            fmt.Printf("文件 %s 上传成功，保存在: %s\n", fileResult.FileName, fileResult.FilePath)
        }
    }

    // 检查文件类型
    contentType := "image/jpeg"
    if storage.IsImageFile(contentType) {
        fmt.Println("这是一个图片文件")
    }

    // 获取安全的文件名
    unsafeFilename := "unsafe/file*name?.jpg"
    safeFilename := storage.GetSafeFilename(unsafeFilename)
    fmt.Printf("安全的文件名: %s\n", safeFilename)
}
```

## 性能优化建议

### 1. 内存管理

- 使用对象池复用结构体
- 预分配合适的切片容量
- 及时释放不需要的资源
- 避免不必要的内存分配

### 2. 并发处理

- 合理使用 goroutine
- 正确管理锁的粒度
- 使用 channel 进行协调
- 注意并发安全

### 3. 算法优化

- 选择合适的数据结构
- 优化热点代码路径
- 减少不必要的计算
- 使用高效的算法实现

### 4. IO 优化

- 使用缓冲 IO
- 批量处理数据
- 复用连接
- 异步处理

## 注意事项

1. 安全性

   - 及时更新依赖包
   - 使用安全的加密算法
   - 正确处理敏感数据
   - 实施访问控制

2. 性能

   - 监控资源使用
   - 定期进行性能测试
   - 及时处理性能瓶颈
   - 优化关键路径

3. 可维护性

   - 遵循代码规范
   - 编写完整的测试
   - 保持文档更新
   - 做好版本管理

4. 兼容性
   - 注意 API 兼容性
   - 处理版本升级
   - 支持不同环境
   - 向后兼容

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交变更
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证，详见 LICENSE 文件。
