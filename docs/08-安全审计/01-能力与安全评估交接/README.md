# 能力与安全评估交接

本目录是 2026-08-15 本地只读评估向整改任务的 Handoff Package。它记录资产、敏感数据类别、已确认 Finding、人工缺口和任务种子；具体执行以 [生产可用性整改任务](../../03-执行任务/01-生产可用性整改/README.md) 为准。

## 文件

1. [00-项目定位与口径](00-项目定位与口径.md)
2. [01-assets](01-assets.md)
3. [02-data-map](02-data-map.md)
4. [03-findings](03-findings.md)
5. [04-manual-gaps](04-manual-gaps.md)
6. [06-task-seed](06-task-seed.md)

## 审计边界

1. 本次仅审查本地仓库代码、文档、测试和依赖元数据。
2. 未运行 DAST、远程攻击、生产探测或数据库检查。
3. 未运行独立 SAST/SCA/secrets 扫描器；依赖只执行了 `go mod verify`、`go mod tidy -diff` 和可更新版本查询。
