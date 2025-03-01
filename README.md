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
- 文件哈希计算（支持 MD5、SHA1、SHA256）
- 文件去重存储
- 文件完整性校验

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

### 分页模块 (`pagination/`)

提供高效易用的分页参数处理功能：

#### 核心功能

- 解析和验证分页参数（limit/offset）
- 自定义分页参数范围
- 从不同数据源获取分页参数
- 计算总页数和当前页码
- 支持自定义参数键名

#### 使用场景

- RESTful API 分页实现
- 数据列表展示
- 大数据集分批处理
- 前端分页组件对接

#### 性能优化

- 参数验证快速失败
- 合理的默认值设置
- 高效的页码计算
- 与 Hertz 框架无缝集成

### 事务管理模块 (`txmanager/`)

提供简单易用的数据库事务管理功能：

#### 核心功能

- 在单个事务中执行多个数据库操作
- 自动处理事务的提交和回滚
- 支持自定义事务选项
- 完善的错误处理和传播机制

#### 使用场景

- 需要原子性操作的数据库更新
- 复杂的业务逻辑涉及多个数据库操作
- 需要保证数据一致性的场景
- 高并发环境下的数据库操作

#### 性能优化

- 精简的事务管理流程
- 高效的错误处理机制
- 支持自定义事务隔离级别
- 资源的及时释放

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

### 5. Storage 模块使用建议

- 根据实际需求选择合适的哈希算法（SHA256 安全性高但较慢，MD5 速度快但安全性较低）
- 对于大文件，考虑使用流式哈希计算而非一次性加载到内存
- 启用文件去重功能可以节省存储空间，但需要权衡哈希计算的开销
- 定期清理临时文件和无效文件
- 对于高并发场景，注意文件锁和并发控制

### 6. 切片操作模块使用建议

- 预估切片容量
- 使用适当的集合操作
- 注意内存使用
- 大数据集考虑并发处理

### 7. 分页模块使用建议

- 根据业务需求设置合适的最大和最小limit值
- 对于大数据集，建议设置合理的默认limit值
- 使用自定义键名时确保前后端参数名一致
- 在处理大量数据时，结合数据库查询优化分页性能

### 8. 事务管理模块使用建议

- 将相关的数据库操作组合在同一事务中执行
- 事务函数应该尽量简短，避免长时间占用数据库连接
- 合理使用事务隔离级别，根据业务需求选择适当的隔离级别
- 避免在事务中执行非数据库操作，如网络请求或文件操作

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

### 分页模块示例

```go
package main

import (
    "fmt"
    "github.com/cloudwego/hertz/pkg/app"
    "github.com/cloudwego/hertz/pkg/protocol"
    "github.com/cloudwego/hertz/pkg/protocol/consts"
    "github.com/iwen-conf/utils-pkg/pagination"
)

func main() {
    // 示例1: 基本的分页参数解析
    limitStr := "20"
    offsetStr := "40"
    limit, offset, err := pagination.ParseLimitOffset(limitStr, offsetStr)
    if err != nil {
        panic(err)
    }
    fmt.Printf("基本解析 - Limit: %d, Offset: %d\n", limit, offset)

    // 示例2: 自定义范围的分页参数解析
    minLimit := 5
    maxLimit := 50
    limit, offset, err = pagination.ParseLimitOffsetWithRange("15", "30", minLimit, maxLimit)
    if err != nil {
        panic(err)
    }
    fmt.Printf("自定义范围 - Limit: %d, Offset: %d\n", limit, offset)

    // 示例3: 从map中获取分页参数
    params := map[string]string{
        "page_size": "30",
        "start":     "60",
    }
    limit, offset, err = pagination.GetPaginationParams(params, "page_size", "start")
    if err != nil {
        panic(err)
    }
    fmt.Printf("从Map获取 - Limit: %d, Offset: %d\n", limit, offset)

    // 示例4: 从Hertz框架的Context中获取分页参数
    req := protocol.NewRequest(consts.MethodGet, "/test?limit=25&offset=50", nil)
    c := app.RequestContext{}
    c.Request = *req
    limit, offset, err = pagination.GetPaginationParamsFromContext(&c)
    if err != nil {
        panic(err)
    }
    fmt.Printf("从Context获取 - Limit: %d, Offset: %d\n", limit, offset)

    // 示例5: 计算总页数和当前页码
    totalItems := 101
    limit = 10
    totalPages := pagination.CalculateTotalPages(totalItems, limit)
    currentPage := pagination.CalculateCurrentPage(offset, limit)
    fmt.Printf("总页数: %d, 当前页码: %d\n", totalPages, currentPage)
}
```

### 事务管理模块示例

```go
package main

import (
    "context"
    "fmt"
    "github.com/iwen-conf/utils-pkg/txmanager"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    // 连接到PostgreSQL数据库
    ctx := context.Background()
    connString := "postgres://username:password@localhost:5432/dbname"
    pool, err := pgxpool.New(ctx, connString)
    if err != nil {
        panic(err)
    }
    defer pool.Close()

    // 创建事务管理器
    tm := txmanager.NewTxManager(pool)

    // 示例1: 在单个事务中执行多个操作
    err = tm.RunInTransaction(ctx,
        // 第一个操作: 创建用户
        func(ctx context.Context, tx pgx.Tx) error {
            _, err := tx.Exec(ctx, "INSERT INTO users(name, email) VALUES($1, $2)", "张三", "zhangsan@example.com")
            return err
        },
        // 第二个操作: 创建用户配置
        func(ctx context.Context, tx pgx.Tx) error {
            _, err := tx.Exec(ctx, "INSERT INTO user_settings(user_id, theme) VALUES(currval('users_id_seq'), $1)", "default")
            return err
        },
    )
    if err != nil {
        fmt.Printf("事务执行失败: %v\n", err)
    } else {
        fmt.Println("事务执行成功")
    }

    // 示例2: 使用自定义事务选项
    opts := pgx.TxOptions{
        IsoLevel: pgx.Serializable, // 设置隔离级别为可序列化
    }
    err = tm.RunInTransactionWithOptions(ctx, opts,
        func(ctx context.Context, tx pgx.Tx) error {
            _, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", 100, 1)
            return err
        },
        func(ctx context.Context, tx pgx.Tx) error {
            _, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance + $1 WHERE id = $2", 100, 2)
            return err
        },
    )
    if err != nil {
        fmt.Printf("转账事务失败: %v\n", err)
    } else {
        fmt.Println("转账事务成功")
    }
}
```
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
    // 创建 User-Agent 解析器
    parser := useragent.NewParser()

    // 解析 User-Agent 字符串
    ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
    info, err := parser.Parse(ua)
    if err != nil {
        panic(err)
    }

    fmt.Printf("浏览器: %s\n", info.Browser)
    fmt.Printf("版本: %s\n", info.Version)
    fmt.Printf("操作系统: %s\n", info.OS)
    fmt.Printf("设备类型: %s\n", info.DeviceType)
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
    "github.com/iwen-conf/utils-pkg/storage"
    "os"
    "path/filepath"
)

func main() {
    // 创建文件存储管理器
    uploadDir := "/path/to/uploads"
    fileManager := storage.NewFileManager(uploadDir)

    // 设置文件上传选项
    options := storage.FileUploadOptions{
        MaxFileSize:        5 * 1024 * 1024,  // 限制单个文件大小为5MB
        AllowedFileTypes:   []string{"image/jpeg", "image/png"},  // 只允许上传jpg和png图片
        GenerateUniqueName: true,  // 生成唯一文件名
        PreserveExtension: true,   // 保留文件扩展名
        SubPath:           "images",  // 文件保存在 uploadDir/images/ 目录下
        MaxTotalSize:      20 * 1024 * 1024,  // 多文件上传时总大小限制为20MB
        EnableHash:        true,   // 启用文件哈希计算
        HashAlgorithm:     "sha256",  // 使用SHA256算法
        DeduplicationEnabled: true,  // 启用文件去重存储
    }

    // 打开要上传的文件
    file, err := os.Open("example.jpg")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    // 上传单个文件
    result, err := fileManager.SaveFile(file, "example.jpg", options)
    if err != nil {
        panic(err)
    }

    fmt.Printf("保存路径: %s\n", result.Path)
    fmt.Printf("文件大小: %d bytes\n", result.Size)
    fmt.Printf("文件类型: %s\n", result.MimeType)
    fmt.Printf("文件哈希: %s\n", result.Hash)
    fmt.Printf("是否为重复文件: %v\n", result.IsDuplicate)

    // 检查文件完整性
    isValid, err := fileManager.VerifyFileIntegrity(result.Path, result.Hash, options.HashAlgorithm)
    if err != nil {
        panic(err)
    }
    fmt.Printf("文件完整性验证: %v\n", isValid)

    // 多文件上传示例
    files := []storage.FileToUpload{
        {File: file, Filename: "example1.jpg"},
        {File: file, Filename: "example2.jpg"},
    }

    results, err := fileManager.SaveMultipleFiles(files, options)
    if err != nil {
        panic(err)
    }

    // 处理重复文件
    for _, res := range results {
        if res.IsDuplicate {
            fmt.Printf("文件 %s 是重复文件，使用已存在的文件: %s\n", res.OriginalName, res.Path)
        } else {
            fmt.Printf("文件 %s 成功上传到: %s\n", res.OriginalName, res.Path)
        }
    }

    // 获取文件哈希信息
    hashInfo, err := fileManager.GetFileHashInfo(filepath.Join(uploadDir, "images", result.Filename))
    if err != nil {
        panic(err)
    }
    fmt.Printf("MD5: %s\n", hashInfo.MD5)
    fmt.Printf("SHA1: %s\n", hashInfo.SHA1)
    fmt.Printf("SHA256: %s\n", hashInfo.SHA256)
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
