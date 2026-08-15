# T04-01 修复 F-ERR-002 恢复标准 errors.Is/As 语义

状态：`[ ]`

## 来源

1. finding_id：F-ERR-002。
2. Handoff：[自定义 Is/As 不遵守标准错误链](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-err-002--自定义-isas-不遵守标准错误链)。
3. 权限类型：service。
4. 数据产量：correctness/availability capability；影响重试、HTTP 映射、分类和根因判断。

## 目标

`errors` 包导出的 `Is/As` 与 Go 标准库完全一致，所有 legacy Error/RichError helper 都能穿透 `%w` 链。错误码匹配使用独立 `IsCode`，不能继续把 string 或 ErrorType 塞给伪 `Is`。

## 项目定位对齐

1. 影响用户角色：所有通过本库判断错误、映射响应或决定重试的下游服务开发者。
2. 信任边界：底层数据库/RPC/业务错误向上层传播时的语义边界。
3. 本任务不决定 HTTP 响应体，只保证错误链和分类事实可靠。

## 允许改动

1. `errors/error_utils.go`。
2. `errors/rich_api.go` 中直接 type assertion 的识别 helper。
3. 必要时 `errors/errors.go` 的 Unwrap/注释，但不改变 Error 字段布局。
4. 新增 `errors/error_utils_test.go`、`errors/error_chain_test.go`。

## 禁止改动

1. 不删除 legacy `Error`、Builder、Validator；弃用由 T04-04 完成。
2. 不在本任务改变 RichError JSON；留给 T04-02。
3. 不通过错误文本比较实现 `Is`。
4. 不吞掉标准库对非法 `As` target 的 panic/契约。

<Context-Queries>

```bash
arc-idx search 'f:^errors/ func Is|func As|GetCode|GetSeverity|GetCategory|GetContext|Chain|Root|FromRichError' --repo . --context 5 --max 220 --format json
arc-idx search 'f:^errors/ \*Error|\*RichError|Original|Unwrap|type assertion|err == target' --repo . --context 4 --max 220 --format json
arc-idx ast --repo . --lang go 'func Is($ERR error, $TARGET interface{}) bool { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. 自定义 `Is` 最终执行 `err == target`，不能穿透标准 `%w`。
2. 自定义 `As` 只遍历特定 `*Error.Original`，遇到 `fmt.wrapError` 或 RichError 即停止。
3. `GetCode/GetSeverity/GetCategory/GetContext` 只看顶层具体类型。
4. `FromRichError`、`RichErrorCode`、`IsRichErrorCode` 等只做顶层 type assertion。
5. `Chain/Root` 只认识 legacy `Error.Original`。

## 固定公开 API

```go
func Is(err, target error) bool
func As(err error, target any) bool
func IsCode(err error, code string) bool
```

1. `Is` 直接返回标准库 `errors.Is(err, target)`。
2. `As` 直接返回标准库 `errors.As(err, target)`。
3. 当前包内导入标准库错误包统一使用别名 `stderrors`，避免包名冲突。
4. `IsCode` 通过 `stderrors.As` 找到链中第一个 `*Error` 并比较 Code；空 code 返回 false。
5. 不保留接收 `interface{}` target 的 overload 或运行时 switch。

## Helper 链式规则

1. `GetCode/GetSeverity/GetCategory/GetContext/GetAllContext` 通过标准 `As` 查找 `*Error`。
2. `FromRichError` 通过标准 `As` 找到链中的 `*RichError`；找到时返回该对象，找不到才包装成 Internal。
3. `RichErrorCode/IsRichErrorCode/IsClientError/IsServerError` 同样穿透 `%w`。
4. `Chain` 对单链反复调用标准 `errors.Unwrap`；nil 返回 nil。
5. `Root` 对单链返回最后一个 error；遇到无单一 `Unwrap() error` 的 joined error 不得猜测某一分支。若保留 join 原对象，必须有测试和注释。

## 执行清单

- [ ] 用 `stderrors` 替换伪 Is/As 内部实现并收紧函数签名。
- [ ] 新增 `IsCode`，把仓库内任何 string-code 调用迁移到新函数；不得为旧签名保留 hidden compatibility。
- [ ] 抽取私有 `asLegacyError`/`asRichError` helper，避免各 getter 重复 type assertion。
- [ ] 修改所有 legacy getter 和 RichError helper 以穿透一层、多层 `%w`。
- [ ] `Chain/Root` 改用标准 Unwrap，并写清 join 约束。
- [ ] 新增标准 sentinel、legacy Error、RichError、多层 fmt.Errorf `%w` 的交叉测试。
- [ ] 全仓搜索 `err ==`/`err !=` 的错误直接比较；本任务修改范围内全部改用标准 `errors.Is`，遵守项目 AGENTS 规则。

## 正向测试

1. `Is(fmt.Errorf("outer: %w", sentinel), sentinel)` 为 true。
2. `As` 能从两层 `%w` 中提取 `*Error` 和 `*RichError`。
3. `IsCode` 能从 `%w` 链找到 legacy code，错误 code 返回 false。
4. GetCode/Severity/Category/Context 对包裹后的 legacy Error 返回原值。
5. FromRichError 对包裹后的 RichError 返回链中同一个指针，不重新包装。
6. Chain 顺序为 outer → inner → root，Root 返回 sentinel。

## 反向测试

1. nil err、nil target、空 code 按标准语义返回 false/默认值。
2. 无对应类型的 As 返回 false，不能修改 target 为伪值。
3. 相同文本的两个独立标准 error 不因文本相等被 Is 判为相同。
4. 不含 RichError 的链由 FromRichError 包装成 Internal，cause 保留原链。
5. joined error 的 Root 行为与文档一致，不静默选择第一/最后分支。

## 验证命令

```bash
gofmt -w errors/*.go
go test -count=1 ./errors -run 'Is|As|IsCode|Chain|Root|GetCode|FromRichError'
go test -race -count=1 ./errors
go test -count=1 ./...
go vet ./...
rg -n 'err\s*(==|!=)\s*' errors --glob '*.go' || true
git diff --check
```

## 不要做

1. 不实现一套“近似标准库”的递归逻辑；直接委托标准库。
2. 不根据 `Error()` 文本判断身份。
3. 不把 code string 转成临时 error 再调用 Is。
4. 不为了兼容旧伪 API 保留 `target interface{}`。
5. 不在 helper 找不到类型时吞掉原 cause。

## 完成标准

1. Is/As 签名和行为与标准库一致，%w 链正反测试通过。
2. code 匹配仅走 IsCode，legacy/Rich helper 均可穿透包装。
3. 本任务范围内不存在直接错误值 `==/!=` 比较。
4. errors 包、race、全量测试和 vet 通过，进度表同步。
