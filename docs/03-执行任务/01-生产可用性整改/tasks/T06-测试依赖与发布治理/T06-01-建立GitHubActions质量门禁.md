# T06-01 GOV-002 建立 GitHub Actions 质量门禁

状态：`[ ]`

## 来源

1. governance_id：GOV-002。
2. Handoff：[发布与供应链治理不足](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#gov-002--发布与供应链治理不足)。
3. 权限类型：maintainer/config-ops。
4. 数据产量：发布与回归能力；当前 PR 没有自动 test/race/vet/module gate。

## 目标

为 pull request 和 main 分支 push 建立最小、可重复、无外部基础设施依赖的 CI。Go 1.24 兼容性与当前 stable 都执行普通测试；race、vet、模块一致性分别作为独立失败面呈现。

## 允许改动

1. 新增 `.github/workflows/ci.yml`。
2. 如确有复用需要，可新增 `.github/actions/` 下本仓库 composite action，但优先保持单文件 workflow。
3. 不修改生产 Go 代码；若门禁暴露失败，应停止并回到对应功能子任务修复根因。

## 禁止改动

1. 不创建 release workflow、自动 tag、自动 push 或包发布。
2. 不引入 Docker、PostgreSQL、Redis、服务容器或远程测试环境。
3. 不用 `continue-on-error`、`|| true` 或允许失败矩阵掩盖红灯。
4. 不在 workflow 中写入 secrets、token 或高权限默认权限。

<Context-Queries>

```bash
arc-idx search 'f:^\.github/ workflow|actions/checkout|setup-go|go test|go vet' --repo . --context 3 --max 120 --format json
arc-idx search 'f:^go.mod go 1.24|module|require' --repo . --context 3 --max 80 --format json
arc-idx search 'f:^docs/ go test -race|go mod tidy -diff|质量门禁' --repo . --context 2 --max 120 --format json
```

</Context-Queries>

## 当前行为

1. 仓库没有 `.github/workflows/`。
2. 本地基线 `go test ./...`、race、vet、mod verify/tidy 均通过，但没有 PR 强制执行证据。
3. `go.mod` 指定 Go 1.24.0；当前审计机为 Go 1.26.5。

## 固定 workflow 结构

文件：`.github/workflows/ci.yml`，name 使用 `CI`。

### 触发与权限

1. `pull_request`。
2. `push` 仅 main。
3. 顶层 `permissions: contents: read`。
4. concurrency group 包含 workflow 和 ref，`cancel-in-progress: true`。

### test job

1. `runs-on: ubuntu-latest`。
2. matrix：`1.24.x`、`stable`；`fail-fast: false`。
3. checkout 固定 `actions/checkout@v4`；setup-go 固定 `actions/setup-go@v5` 并开启 module cache。
4. 顺序：`go version` → `go mod download` → `go mod verify` → `go test -count=1 ./...`。

### race job

1. stable Go，`go test -race -count=1 ./...`。
2. 独立 job，便于识别 race 与普通测试失败。

### vet-and-module job

1. Go 1.24.x。
2. `go vet ./...`。
3. `go mod tidy -diff`，diff 非空必须失败。
4. 再执行 `git diff --exit-code -- go.mod go.sum`，防止旧 Go 行为遗漏修改。

### coverage/security

1. T06-01 不设置百分比阈值；T06-02 在同一 workflow 增加 coverage job。
2. T06-03 增加 govulncheck job。
3. 可以先保留清晰注释，不允许 placeholder 成功步骤。

## 执行清单

- [ ] 创建 ci.yml，按固定 job 拆分，不使用 shell 吞错。
- [ ] workflow 所有 run 使用默认 fail-fast shell；多行脚本加 `set -euo pipefail`（bash）。
- [ ] Go 版本必须包含 1.24.x 和 stable，不能只测审计机版本。
- [ ] 不指定本地绝对路径、开发者用户名或私有环境变量。
- [ ] 检查 YAML 缩进、matrix 表达式和 permissions。
- [ ] 本地重跑与 workflow 对应命令；如果 actionlint 已安装可运行，未安装不为此引入依赖。
- [ ] 提交前检查没有 release/push/write permission。

## 正向验证

1. PR/main push 均触发。
2. test 矩阵显示两个 Go 版本。
3. race、vet/module 是独立可见 job。
4. 干净工作树上所有本地对应命令通过。

## 反向验证

1. 临时在独立验证分支制造失败测试时 job 应失败；验证后恢复该临时改动，不能提交故障代码。
2. `go mod tidy -diff` 有输出时 job 失败。
3. workflow 不包含 `continue-on-error: true`、`pull_request_target`、write-all、Docker 或 release action。

## 验证命令

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go mod verify
go mod tidy -diff
git diff --check
rg -n 'continue-on-error|pull_request_target|write-all|docker|release|git push|git tag' .github/workflows || true
git status --short
```

## 不要做

1. 不让矩阵某个版本 optional。
2. 不把 race 合并到易被 matrix fail-fast 取消的单个步骤。
3. 不在 CI 自动修改 go.mod/go.sum 后继续成功。
4. 不安装或启动本机/CI 数据库和容器；本仓库测试不需要。
5. 不加入发布权限或秘密。

## 完成标准

1. workflow 结构、版本矩阵、只读权限和独立 jobs 与本任务一致。
2. 本地等价命令全绿；首次 GitHub 运行结果写入进度表后才可最终 `[x]`。
3. 无吞错、无 release、无外部服务依赖。
