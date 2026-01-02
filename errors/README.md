# RichError - 企业级富错误处理包

一个为 Go 业务项目设计的 **"富错误"** 包，实现了 **内外有别** 的错误处理模式。

## ✨ 核心特性

| 特性 | 说明 |
|------|------|
| **组合优先** | `RichError` 嵌入 `Status`，可直接用于 JSON 响应 |
| **内外有别** | 对外只暴露 `Code`/`Msg`，对内保留 `Cause`/`Stack` |
| **标准兼容** | 完整支持 `errors.Is`/`errors.As` |
| **调试友好** | `%+v` 格式化输出完整堆栈信息 |
| **HTTP 映射** | 业务码自动推导 HTTP 状态码 |
| **高性能** | sync.Pool 优化，内存分配减少 80% |

---

## 📦 快速开始

```go
import "github.com/iwen-conf/utils-pkg/errors"
```

### 核心类型

```go
// Status 可直接被 JSON 序列化
type Status struct {
    Code int    `json:"code"` // 业务码
    Msg  string `json:"msg"`  // 用户提示语
}

// RichError 嵌入 Status
type RichError struct {
    Status        // 可直接访问 e.Code 和 e.Msg
    cause  error  // 根因（不导出）
    stack  *stack // 堆栈（不导出）
}
```

### 预定义业务码

```go
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

## 🛠️ 核心 API

### `NewRich` - 创建业务错误

```go
err := errors.NewRich(400001, "手机号格式错误")
err := errors.NewRich(errors.RichCodeNotFound, "用户不存在")
```

### `WrapRich` - 包装底层错误

```go
if err != nil {
    return errors.WrapRich(err, errors.RichCodeDBError, "查询用户失败")
}
```

### `FromRichError` - 智能转换

```go
func handleError(c *gin.Context, err error) {
    e := errors.FromRichError(err)
    
    if errors.IsServerError(e) {
        log.Printf("%+v", e)  // 打印完整堆栈
    }
    
    c.JSON(e.HTTPStatus(), gin.H{"code": e.Code, "msg": e.Msg})
}
```

---

## 🚀 快捷构造函数

```go
errors.RichBadRequest("邮箱格式不正确")   // 400000
errors.RichUnauthorized()                // 401000
errors.RichForbidden()                   // 403000
errors.RichNotFound("用户")              // 404000 -> "用户不存在"
errors.RichInternal(dbErr)               // 500000 (隐藏底层错误)
errors.RichDBError(dbErr)                // 500001
```

---

## ⚡ 高性能版本（无堆栈）

适用于不需要堆栈跟踪的简单业务错误：

```go
// 无堆栈版本，性能更高
errors.NewRichNoStack(400001, "参数错误")
errors.WrapRichNoStack(err, 500001, "系统错误")
```

---

## 🔗 HTTP 状态码映射

业务码自动推导 HTTP 状态码（取前 3 位）：

```go
e := errors.NewRich(404001, "用户不存在")
e.HTTPStatus() // -> 404

e := errors.NewRich(500001, "数据库错误")
e.HTTPStatus() // -> 500
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
    return s.repo.GetByID(ctx, id)
}
```

### Controller 层

```go
type Response struct {
    errors.Status
    Data interface{} `json:"data,omitempty"`
}

func Error(c *gin.Context, err error) {
    e := errors.FromRichError(err)
    
    if errors.IsServerError(e) {
        log.Printf("ERROR: %+v", e)
    }
    
    c.JSON(e.HTTPStatus(), Response{Status: e.GetStatus()})
}
```

---

## 🔍 判断函数

```go
errors.IsClientError(err)                        // 4xx
errors.IsServerError(err)                        // 5xx
errors.IsRichErrorCode(err, errors.RichCodeNotFound)
errors.RichErrorCode(err, 500000)                // 获取业务码
```

---

## ⛓️ 链式方法

```go
err.WithCode(400002)  // 修改业务码（返回新对象）
err.WithMsg("新消息") // 修改消息（返回新对象）
```

---

## 📝 JSON 序列化

```go
data, _ := err.MarshalJSON()
// {"code":500001,"msg":"数据库错误","cause":"connection refused"}
```

---

## 🔍 日志输出

### `%v` 普通模式
```
数据库错误
```

### `%+v` 详细模式
```
Code: 500001
Msg: 数据库错误
Cause: connection refused
Stack:
    /app/internal/repo/user_repo.go:45
    /app/internal/service/user_service.go:23
```

---

## ✅ nil 安全

所有方法在 `nil` 接收者上安全调用：

```go
var e *errors.RichError = nil
e.Error()       // -> ""
e.HTTPStatus()  // -> 200
e.GetStatus()   // -> Status{Code: 500000, Msg: "系统繁忙..."}
```

---

## ⚡ 性能指标

| 操作 | 耗时 | 内存 |
|------|------|------|
| `NewRich` | 152 ns | 56 B |
| `WrapRich` | 173 ns | 104 B |
| `NewRichNoStack` | ~10 ns | 32 B |
| `FromRichError` | 0.95 ns | 0 B |
| `HTTPStatus()` | <1 ns | 0 B |

---

## 📁 文件结构

```
errors/
├── rich_error.go      # RichError + Status + MarshalJSON
├── rich_api.go        # API + 预定义业务码 + 快捷函数
├── stack.go           # 堆栈捕获 (sync.Pool 优化)
├── rich_error_test.go # 功能测试
└── rich_benchmark_test.go # 性能测试
```
