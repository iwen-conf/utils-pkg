# T04 错误模型与信息边界

## 目标

恢复标准 Go 错误链语义，防止 HTTP/JSON 自动泄漏底层 cause，并给两套错误 API 建立清晰迁移边界。

## 已知范围

1. `errors/errors.go`
2. `errors/error_utils.go`
3. `errors/error_validation.go`
4. `errors/rich_error.go`、`errors/rich_api.go`
5. `errors/*_test.go`、`errors/README.md`

## 子任务索引

| 状态 | 子任务 | Finding |
| --- | --- | --- |
| `[ ]` | [T04-01 恢复标准 errors 语义](T04-01-恢复标准errors语义.md) | F-ERR-002 |
| `[ ]` | [T04-02 阻止 RichError JSON 泄漏](T04-02-阻止RichErrorJSON泄漏.md) | F-ERR-001 |
| `[ ]` | [T04-03 修复旧 Error/Validator 并补测试](T04-03-修复旧ErrorValidator并补测试.md) | F-ERR-002 |
| `[ ]` | [T04-04 确定弃用与迁移路线](T04-04-确定错误模型弃用与迁移路线.md) | GOV-001 |

## 验收标准

1. `%w` 包装后 `Is/As`、code/category/severity 均能按设计工作。
2. JSON 默认不包含 cause/stack。
3. 旧错误系统核心路径有真实测试，不再 0% 覆盖。
4. 新用户只看到一个明确推荐路径，旧 API 有弃用和迁移说明。
