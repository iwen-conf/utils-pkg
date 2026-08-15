# T06-03 GOV-002 升级兼容依赖并接入 govulncheck

状态：`[ ]`

## 来源

1. governance_id：GOV-002。
2. Handoff：[发布与供应链治理不足](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#gov-002--发布与供应链治理不足)。
3. 2026-08-15 基线：`jwt/v5 v5.3.0` 可到 v5.3.1；`x/crypto v0.42.0` 最新 v0.55.0 需要 Go 1.25，Go 1.24 兼容的已确认最新候选为 v0.48.0。
4. govulncheck v1.6.0 基线：0 个可达漏洞；导入包有 3 个、模块图有 22 个不可达告警。
5. 权限类型：maintainer/config-ops。
6. 数据产量：供应链与可达漏洞事实。

## 目标

在不提升 `go 1.24.0` directive 的前提下，把直接依赖升级到执行时最新的 Go 1.24 兼容版本，逐项验证行为；CI 固定运行 govulncheck，任何可达漏洞或扫描失败都阻断。

## 前置

1. T05-01 已移除 `golang.org/x/exp`。
2. T06-01 CI 和 T06-02 测试/覆盖门禁已稳定。

## 允许改动

1. `go.mod`、`go.sum`。
2. 因依赖兼容性产生的最小调用点/测试修改，必须先更新本任务“允许改动”并记录；禁止顺手重构。
3. `.github/workflows/ci.yml` 增加 govulncheck job。
4. 新增 `docs/08-安全审计/02-govulncheck依赖检查.md` 并更新该目录 README 索引。

## 禁止改动

1. 不执行 `go get -u ./...` 或一次性全模块漂移。
2. 不把 go directive 升到 1.25/1.26；若安全修复只能通过更高 Go 版本获得，标 `[!]` 请求用户决策。
3. 不忽略 govulncheck exit code，不把可达漏洞降级成 warning。
4. 不把不可达告警直接宣称为“已修复”，也不把它们等同可利用漏洞。

<Context-Queries>

```bash
arc-idx search 'f:^go.mod require|github.com/golang-jwt/jwt|golang.org/x/crypto|golang.org/x/exp|golang.org/x/sys' --repo . --context 4 --max 120 --format json
arc-idx search 'f:^\.github/workflows/ govulncheck|go run golang.org/x/vuln' --repo . --context 3 --max 100 --format json
arc-idx search 'f:^docs/08-安全审计/ GOV-002|dependency|vulnerability' --repo . --context 2 --max 100 --format json
```

</Context-Queries>

## 当前行为

1. 直接依赖版本低于可用更新，且 x/crypto 的最新版本已提高最低 Go 版本。
2. 仓库没有 pinned govulncheck CI；扫描只在审计机手工运行。
3. 当前扫描为 0 reachable，但不可达告警和工具/DB 版本没有项目内报告。

## 固定选择规则

1. 执行时先运行 `go list -m -u -mod=readonly all`。
2. 对每个候选运行 `go list -m -json module@version`，读取 `GoVersion`。
3. 只选择 `GoVersion <= 1.24.x` 的最高稳定版本。
4. 2026-08-15 已确认候选：
   - `github.com/golang-jwt/jwt/v5@v5.3.1`（Go 1.21）；
   - `golang.org/x/crypto@v0.48.0`（Go 1.24，依赖 x/sys v0.41.0）。
5. 若执行时出现更新候选，按规则重新确认；不能盲目复制旧版本，也不能选择要求 Go 1.25 的 x/crypto v0.49+。

## 固定升级顺序

1. 单独升级 jwt/v5，运行 jwt 包、全量、race、vet。
2. 单独升级 x/crypto，运行 crypto 包、全量、race、vet。
3. `go mod tidy`，审查 indirect 变化。
4. `go mod verify`、`go mod tidy -diff`。
5. `govulncheck ./...`。
6. 任何一步失败先定位 API/行为差异；不通过 downgrade 或 replace 掩盖，除非有明确兼容/安全理由并记录。

## govulncheck CI 契约

1. 新增独立 `govulncheck` job，使用 stable Go。
2. 运行固定工具版本，例如审计基线 `go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...`。
3. 工具版本升级必须独立审查；不能用 `@latest` 造成不可重复。
4. job 失败即 CI 失败，不使用 continue-on-error。
5. 本任务不把 x/vuln 加入业务 `require`；它是 CI tool invocation。

## 执行清单

- [ ] 记录升级前 `go list -m all` 和 govulncheck 摘要。
- [ ] 按固定规则确认每个候选 GoVersion，不提升 go directive。
- [ ] 逐依赖升级并在每步运行对应门禁，保留失败证据。
- [ ] 检查 jwt/v5 parser/error 变化是否影响 T01 的严格验证测试。
- [ ] 检查 x/crypto Argon2/scrypt/bcrypt 行为是否影响 T03 的边界测试。
- [ ] tidy 后确认 x/exp 不回归，indirect 只有可解释变化。
- [ ] 增加 pinned govulncheck CI job。
- [ ] 写依赖检查报告：日期、Go/tool/DB 版本、直接依赖前后、可达结果、不可达摘要和剩余风险。

## 正向验证

1. 两个直接依赖升级到兼容候选，所有门禁通过。
2. go directive 保持 1.24.0。
3. govulncheck exit 0 且 0 reachable vulnerabilities。
4. CI 使用 pinned 工具版本。

## 反向/阻塞条件

1. 候选要求 Go 1.25+：不升级，记录最高兼容版本；若该版本包含必要安全修复则标 `[!]` 请求提升 Go 决策。
2. 出现可达漏洞：任务不得 `[x]`；记录 OSV ID、调用路径和升级/修复子任务。
3. 依赖升级导致安全回归测试失败：修复根因或明确阻塞，不删除测试。
4. `go mod tidy` 重新引入 x/exp：查找真实导入，不手删校验行。

## 验证命令

```bash
go list -m -u -mod=readonly all
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go mod verify
go mod tidy -diff
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
rg -n '^go 1\.24\.0$|golang.org/x/exp' go.mod go.sum
git diff --check
```

## 不要做

1. 不自动选择最新但提高最低 Go 版本的依赖。
2. 不用 replace 固定本地路径或 fork。
3. 不删除失败测试迎合新版依赖。
4. 不把“无可达漏洞”写成“依赖图无任何漏洞”。
5. 不安装/运行数据库或容器。

## 完成标准

1. 直接依赖选择有 Go 1.24 兼容证据，升级逐项验证。
2. govulncheck 本地和 CI 都是可重复的 pinned invocation。
3. 0 reachable；不可达告警如实记录。
4. 模块、测试、race、vet、coverage 全部门禁通过，进度表同步版本和结果。
