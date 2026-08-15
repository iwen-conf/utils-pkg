# T04-02 修复 F-ERR-001 阻止 RichError JSON 泄漏 cause

状态：`[ ]`

## 来源

1. finding_id：F-ERR-001。
2. Handoff：[RichError JSON 自动包含底层 cause](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-err-001--richerror-json-自动包含底层-cause)。
3. 权限类型：anonymous/user；前提是下游把 RichError 直接 JSON 序列化到 HTTP 响应。
4. 数据产量：SECRET/PII/BIZ，通常 L1–L2，取决于 cause 中的 SQL、地址、标识或用户数据。

## 目标

RichError 默认 JSON 永远只包含 `code` 和 `msg`。cause 和 stack 仅能通过 `Cause()`、`Stack()` 或 `%+v` 显式用于内部日志，不能因直接 marshal 自动进入响应。

## 项目定位对齐

1. 影响用户角色：接收 API 错误响应的匿名/已登录用户，以及记录内部日志的服务开发者。
2. 信任边界：内部诊断信息与外部响应序列化之间的边界。
3. `msg` 仍由调用方负责提供面向用户的安全文本；本任务不自动清洗业务 msg。

## 前置

1. T04-01 已恢复标准错误链，RichError cause 仍可由标准 `errors.Is/As` 访问。

## 允许改动

1. `errors/rich_error.go` 的 `MarshalJSON` 和注释。
2. `errors/rich_error_test.go`。
3. `errors/README.md` 的 JSON 与响应示例。

## 禁止改动

1. 不删除 `Cause()`、`Stack()`、`Unwrap()` 或 `%+v` 日志能力。
2. 不把 cause 改成导出字段。
3. 不新增 `debug=true`、环境变量或全局开关让默认 JSON 泄漏。
4. 不把 stack/cause 塞入 `msg`。

<Context-Queries>

```bash
arc-idx search 'f:^errors/ RichError|MarshalJSON|Cause\(|Stack\(|Format\(|GetStatus' --repo . --context 5 --max 160 --format json
arc-idx search 'f:^errors/ json:"cause|cause.Error|JSON should contain cause|日志输出|Controller' --repo . --context 4 --max 160 --format json
arc-idx ast --repo . --lang go 'func ($E *RichError) MarshalJSON() ([]byte, error) { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. struct 注释声称 cause“不导出，不给前端看”。
2. 自定义 `MarshalJSON` 却主动写入 `cause.Error()`。
3. 测试明确要求 JSON 包含 cause，固化了泄漏。
4. README 示例展示 `{"cause":"connection refused"}`。

## 固定 JSON 契约

1. 非 nil RichError：`{"code":<int>,"msg":<string>}`，只允许这两个键。
2. nil receiver：仍序列化为 `null`。
3. 使用 `encoding/json.Marshal(err)`、直接 `err.MarshalJSON()`、嵌入其他 response struct 三条路径结果一致。
4. cause/stack 只能通过显式内部日志接口获得；不新增另一种默认 JSON formatter。

## 执行清单

- [ ] 将 MarshalJSON 的内部 DTO 缩减为 Code/Msg，删除 Cause 字段和赋值。
- [ ] 更新函数注释，说明其用于安全响应序列化，不是详细日志序列化。
- [ ] 翻转现有 TestMarshalJSON：断言 code/msg 存在，cause/stack/底层文本不存在。
- [ ] 新增敏感 cause 回归，例如包含 DSN、token、内部 RPC 地址；检查序列化字符串完全不含这些片段。
- [ ] 新增嵌套 response 测试，防止 `json.Marshal(struct{ Error *RichError })` 泄漏。
- [ ] 保持 `%+v` 测试：内部详细格式仍包含 cause 和 stack。
- [ ] 更新 README JSON 示例和 Controller 示例，优先返回 `GetStatus()` 或安全 RichError JSON。

## 正向测试

1. NewRich JSON 只有 code/msg。
2. WrapRich JSON 同样只有 code/msg。
3. nil RichError JSON 为 `null`。
4. `%+v` 对 WrapRich 仍包含 cause 文本和 stack，证明诊断能力未丢失。
5. `errors.Is` 仍可穿透 cause。

## 反向测试

1. JSON key 集合中不存在 `cause`、`stack`、`details`、`original`。
2. cause 中的 `postgres://user:pass@...`、Bearer token 或内部路径不出现在 JSON 字节中。
3. 普通 `%v`/`Error()` 不输出 cause。
4. 不允许通过全局配置让同一对象在不同环境输出不同 JSON shape。

## 验证命令

```bash
gofmt -w errors/rich_error.go errors/rich_error_test.go
go test -count=1 ./errors -run 'MarshalJSON|Format|Cause|Status|Leak'
go test -race -count=1 ./errors
go test -count=1 ./...
go vet ./...
rg -n 'json:"cause|Cause string|JSON should contain cause' errors || true
git diff --check
```

## 不要做

1. 不对 cause 做字符串脱敏后继续放进响应；默认应完全不输出。
2. 不根据 `GO_ENV`、debug 标志或日志级别改变 MarshalJSON。
3. 不删除标准 Unwrap 以“防泄漏”；错误链与 JSON 是不同边界。
4. 不在安全 msg 中拼接 `err.Error()`。

## 完成标准

1. 三种 JSON 路径均只输出 code/msg，敏感 cause 回归稳定通过。
2. 内部 `%+v`、Cause、Stack、errors.Is/As 仍可用。
3. README 不再展示含 cause 的响应 JSON。
4. errors 包、race、全量测试和 vet 通过。
