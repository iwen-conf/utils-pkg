# utils-pkg

实用工具包集合，提供常用的基础功能支持。

## 包含模块

1. **jwt**: JWT令牌管理
   - 生成与验证JWT令牌
   - 支持访问令牌与刷新令牌
   - 高性能的令牌验证缓存
   - 分段锁设计保证高并发性能

2. **crypto**: 加密工具
   - AES加密/解密
   - 支持多种加密模式
   - 线程安全的加密操作

3. **slice**: 切片工具
   - 切片差集、交集、并集操作
   - 元素查找和去重
   - 切片分页工具

4. **url**: URL工具
   - URL解析和构建
   - 查询参数处理
   - URL标准化

5. **useragent**: 用户代理解析
   - 浏览器信息提取
   - 操作系统识别
   - 设备类型判断

6. **errors**: 通用错误处理系统 ✨ **新增**
   - 灵活的错误码注册系统
   - 支持动态错误码管理
   - 错误分类和判断（系统、客户端、业务）
   - 错误构建器模式
   - 错误格式化器（默认、JSON）
   - 错误处理器链
   - 错误聚合器
   - 完整的堆栈跟踪支持

## 安装

```bash
go get github.com/iwen-conf/utils-pkg
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
- [错误处理系统使用说明](errors/使用说明.md) ✨ **新增**

## 特性

- 🚀 **高性能**: 使用对象池、缓存等优化技术
- 🔒 **线程安全**: 所有模块都经过并发安全设计
- 🎯 **通用性**: 不再局限于特定业务场景，适用于各种项目
- 🔧 **可扩展**: 支持自定义错误码、格式化器、处理器等
- 📚 **完整文档**: 提供详细的使用说明和示例代码
- ✅ **测试覆盖**: 所有模块都有完整的测试覆盖
