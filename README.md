# utils-pkg

实用工具包集合，提供常用的基础功能支持。

## 包含模块

1. **jwt**: JWT令牌管理
   - 生成与验证JWT令牌
   - 支持访问令牌与刷新令牌
   - 支持统一身份归一化签发 access/refresh token
   - 支持默认 issuer/audience 与自定义身份 claim key
   - 高性能的令牌验证缓存
   - 支持 token/session 吊销和可插拔 RevocationStore

2. **crypto**: 加密工具
   - AES-GCM 认证加密
   - Argon2 / scrypt / bcrypt，参数在计算前校验
   - 并发安全的密码策略

3. **slice**: 切片工具
   - 差集、交集、并集、去重、过滤、映射
   - 使用标准库排序，不提供自动并行

4. **url**: URL工具
   - v2 签名绑定 method/scheme/host/path/query
   - 默认不接受 legacy 签名

5. **useragent**: 用户代理解析
   - 启发式浏览器识别，不是安全判定

6. **errors**: 错误处理
   - 推荐 `RichError`；默认 JSON 只有 `code` 和 `msg`
   - 标准 `errors.Is` / `errors.As`

7. **captcha**: 验证码
   - CSPRNG 数字验证码
   - 空 expected/input 永不成功
   - 自定义字符集按 rune 生成

## 安装

```bash
go get github.com/iwen-conf/utils-pkg
```

## 验证码生成器使用示例

### 基础用法

```go
import "github.com/iwen-conf/utils-pkg/captcha"

// 生成推荐的6位验证码
code, err := captcha.Generate6()
if err != nil {
    panic(err)
}
fmt.Printf("验证码: %s\n", code)

// 生成自定义长度的验证码
code8, err := captcha.Generate(8)
if err != nil {
    panic(err)
}

// 验证验证码
userInput := "123456"
if captcha.Validate(userInput, code) {
    fmt.Println("验证码正确")
} else {
    fmt.Println("验证码错误")
}
```

### 安全特性

```go
// 使用密码学安全的随机数生成器
code, err := captcha.Generate6() // 使用crypto/rand

// 格式验证
if captcha.IsValidFormat(userInput) {
    // 验证码格式正确
}

// 不同长度的验证码
code4, _ := captcha.Generate4()  // 4位，1万种组合
code6, _ := captcha.Generate6()  // 6位，100万种组合（推荐）
code8, _ := captcha.Generate8()  // 8位，1亿种组合
```

## 错误处理模块使用示例

### 基础用法

```go
import "github.com/iwen-conf/utils-pkg/errors"

// 创建简单错误
err := errors.New("USER001", "用户不存在")

// 创建带详细信息的错误
err = errors.NewWithDetails("DATA001", "数据验证失败", "输入参数不符合要求")

// 包装现有错误
dbErr := fmt.Errorf("数据库连接失败")
err = errors.Wrap(dbErr, "DB001", "数据库操作异常")
```

### 错误码注册系统

```go
// 注册自定义错误码
errors.RegisterErrorCode("CUSTOM001", "自定义业务错误")

// 批量注册错误码
errors.RegisterErrorCodes(map[string]string{
    "AUTH001": "认证失败",
    "AUTH002": "权限不足",
    "AUTH003": "令牌过期",
})

// 注册错误码前缀分类
errors.RegisterErrorPrefix("AUTH", "authentication")
```

### 错误构建器

```go
err := errors.NewBuilder().
    Code("USER001").
    Message("用户不存在").
    Details("用户ID无效").
    Context("user_id", "user123").
    Context("request_id", "req_456").
    Build()
```

### 错误分类和判断

```go
// 根据错误码判断类型
if errors.IsSystemError("5000") {
    // 系统级错误
}

if errors.IsClientError("4001") {
    // 客户端错误
}

if errors.IsBusinessErrorCode("6001") {
    // 业务错误
}

// 获取错误分类
category := errors.GetCategoryByCode("5000") // 返回 "server"
```

### 错误格式化

```go
// 使用默认格式化器
formatted := errors.FormatError(err)

// 设置自定义格式化器
errors.SetDefaultFormatter(&errors.JSONFormatter{})

// 使用错误处理器链
handlerChain := errors.NewHandlerChain().
    Add(func(err *errors.Error) error {
        // 记录错误日志
        log.Printf("Error: %s", err.Error())
        return nil
    }).
    Add(func(err *errors.Error) error {
        // 发送错误通知
        return nil
    })

handlerChain.Handle(err)
```

### 错误聚合

```go
aggregator := errors.NewAggregator()

// 收集多个错误
aggregator.Add(err1)
aggregator.Add(err2)
aggregator.Add(err3)

if aggregator.HasErrors() {
    // 处理聚合的错误
    for _, err := range aggregator.Errors() {
        fmt.Printf("Error: %s\n", err.Error())
    }
}
```

## 预定义错误码

系统提供了一套通用的错误码体系：

- **客户端错误 (4xxx)**: 请求错误、未授权、禁止访问等
- **服务端错误 (5xxx)**: 内部错误、服务不可用、网关错误等  
- **业务错误 (6xxx)**: 业务逻辑错误、数据验证失败等

所有错误码都支持动态注册和自定义扩展。

## 详细文档

各模块使用方法详见各自目录下的文档。

- [JWT令牌管理器使用说明](jwt/使用说明.md)
- [加密工具使用说明](crypto/使用说明.md)
- [切片工具使用说明](slice/使用说明.md)
- [URL工具使用说明](url/使用说明.md)
- [用户代理解析工具使用说明](useragent/使用说明.md)
- [错误处理系统使用说明](errors/README.md)
- [验证码生成器使用说明](captcha/使用说明.md)

## 特性

- 生产可用性整改见 `CHANGELOG.md` 的 `0.4.0` 候选说明
- CI 执行 test / race / vet / coverage / govulncheck
- 当前版本：`Version()` = `v0.4.0`（技术候选；许可证未决时不可公开发布）
