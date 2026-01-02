# RichError - 企业级富错误处理包

一个为 Go 微服务设计的 **"富错误"** 包，实现了 **内外有别** 的错误处理模式。

## ✨ 核心特性

| 特性 | 说明 |
|------|------|
| **组合优先** | `RichError` 嵌入 `Status`，可直接用于 JSON 响应 |
| **内外有别** | 对外只暴露 `Code`/`Msg`，对内保留 `Cause`/`Stack` |
| **标准兼容** | 完整支持 `errors.Is`/`errors.As` |
| **调试友好** | `%+v` 格式化输出完整堆栈信息 |
| **HTTP 映射** | 业务码自动推导 HTTP 状态码 |

---

## 📦 快速开始

### 安装

```go
import "github.com/iwen-conf/utils-pkg/errors"
```

### 核心类型

```go
// Status 可直接被 JSON 序列化，嵌入到 Response 中
type Status struct {
    Code int    `json:"code"` // 业务码
    Msg  string `json:"msg"`  // 用户提示语
}

// RichError 嵌入 Status，自然拥有 Code 和 Msg 字段
type RichError struct {
    Status        // 可直接访问 e.Code 和 e.Msg
    cause  error  // 根因（不导出）
    stack  *stack // 堆栈（不导出）
}
```

### 预定义业务码

```go
// 业务码规范：HTTP状态码(3位) + 模块码(3位)
const (
    RichCodeSuccess      = 0       // 成功
    RichCodeBadRequest   = 400000  // 参数错误
    RichCodeUnauthorized = 401000  // 未认证
    RichCodeForbidden    = 403000  // 无权限
    RichCodeNotFound     = 404000  // 资源不存在
    RichCodeInternal     = 500000  // 系统内部错误
    RichCodeDBError      = 500001  // 数据库错误
)
```

---

## 🛠️ 三个核心 API

### 1. `NewRich` - 创建业务错误

**适用场景**：Service 层参数校验失败、业务逻辑不满足

```go
err := errors.NewRich(400001, "手机号格式错误")
err := errors.NewRich(errors.RichCodeNotFound, "用户不存在")
```

### 2. `WrapRich` - 包装底层错误

**适用场景**：Repo 层数据库报错、第三方 API 调用失败

```go
user, err := repo.GetUserByID(ctx, id)
if err != nil {
    // 把脏错误包装成干净的业务错误
    return errors.WrapRich(err, errors.RichCodeDBError, "查询用户失败")
}
```

### 3. `FromRichError` - 智能转换

**适用场景**：Controller/Response 层统一错误响应

```go
func handleError(c *gin.Context, err error) {
    e := errors.FromRichError(err)
    
    // 5xx 错误打印详细日志
    if errors.IsServerError(e) {
        log.Printf("%+v", e)  // 打印 Code + Msg + Cause + Stack
    }
    
    // 返回 JSON
    c.JSON(e.HTTPStatus(), gin.H{
        "code": e.Code,
        "msg":  e.Msg,
    })
}
```

---

## 🚀 快捷构造函数

```go
// 参数错误
err := errors.RichBadRequest("邮箱格式不正确")

// 未认证
err := errors.RichUnauthorized()

// 无权限
err := errors.RichForbidden()

// 资源不存在
err := errors.RichNotFound("用户")  // -> "用户不存在"

// 系统错误（隐藏底层错误）
err := errors.RichInternal(dbErr)

// 数据库错误
err := errors.RichDBError(dbErr)
```

---

## 🔗 HTTP 状态码映射

业务码自动推导 HTTP 状态码：取前 3 位

```go
e := errors.NewRich(404001, "用户不存在")
e.HTTPStatus() // -> 404

e := errors.NewRich(500001, "数据库错误")
e.HTTPStatus() // -> 500

e := errors.NewRich(0, "成功")
e.HTTPStatus() // -> 200
```

---

## 📋 分层使用示例

### Repo 层

```go
func (r *UserRepo) GetByID(ctx context.Context, id int64) (*User, error) {
    user := &User{}
    err := r.db.First(user, id).Error
    
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.RichNotFound("用户")
        }
        return nil, errors.RichDBError(err)
    }
    return user, nil
}
```

### Service 层

```go
func (s *UserService) GetUser(ctx context.Context, id int64) (*User, error) {
    if id <= 0 {
        return nil, errors.RichBadRequest("用户ID无效")
    }
    return s.repo.GetByID(ctx, id)  // 错误直接透传
}
```

### Controller 层 - Response 函数

```go
// Response 嵌入 Status
type Response struct {
    errors.Status
    Data interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
    c.JSON(200, Response{
        Status: errors.Status{Code: 0, Msg: "success"},
        Data:   data,
    })
}

func Error(c *gin.Context, err error) {
    e := errors.FromRichError(err)
    
    // 服务端错误打印完整日志
    if errors.IsServerError(e) {
        log.Printf("ERROR: %+v", e)
    }
    
    c.JSON(e.HTTPStatus(), Response{Status: e.GetStatus()})
}
```

---

## 🔍 判断函数

```go
// 是否是客户端错误 (4xx)
if errors.IsClientError(err) { ... }

// 是否是服务端错误 (5xx)
if errors.IsServerError(err) { ... }

// 是否是指定业务码
if errors.IsRichErrorCode(err, errors.RichCodeNotFound) { ... }

// 获取业务码（非 RichError 返回默认值）
code := errors.RichErrorCode(err, 500000)
```

---

## ⛓️ 链式方法

```go
// 修改业务码（返回新对象）
newErr := err.WithCode(400002)

// 修改消息（返回新对象）
newErr := err.WithMsg("自定义消息")
```

---

## 🔍 日志输出格式

### 普通打印 (`%v`)

```
查询用户失败
```

### 详细打印 (`%+v`)

```
Code: 500001
Msg: 查询用户失败
Cause: Error 1045: Access denied for user 'root'@'localhost'
Stack:
    /app/internal/repo/user_repo.go:45
    /app/internal/service/user_service.go:23
```

---

## 🔗 与标准库兼容

```go
// errors.Is 判断底层错误
if errors.Is(richErr, pgx.ErrNoRows) {
    // ✅ 能够穿透判断
}

// errors.As 转换错误
var e *errors.RichError
if errors.As(err, &e) {
    fmt.Println(e.Code, e.Msg)
}
```

---

## ⚡ 性能指标

| 操作 | 耗时 | 内存 |
|------|------|------|
| `NewRich` | 167 ns | 280 B |
| `WrapRich` | 185 ns | 328 B |
| `FromRichError` (RichError) | 0.97 ns | 0 B |
| `HTTPStatus()` | <1 ns | 0 B |

---

## 📁 文件结构

```
errors/
├── rich_error.go      # RichError + Status 核心类型
├── rich_api.go        # API + 预定义业务码 + 快捷函数
├── stack.go           # 堆栈捕获
├── rich_error_test.go # 功能测试
└── rich_benchmark_test.go # 性能测试
```
