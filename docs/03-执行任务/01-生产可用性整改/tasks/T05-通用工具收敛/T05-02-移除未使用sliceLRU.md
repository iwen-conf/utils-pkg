# T05-02 修复 F-UTIL-001 移除未使用 slice LRU

状态：`[ ]`

## 来源

1. finding_id：F-UTIL-001（slice 部分）。
2. Handoff：[自研 LRU 对无效容量和并发更新不安全](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-util-001--自研-lru-对无效容量和并发更新不安全)。
3. 权限类型：service。
4. 数据产量：availability capability；`capacity <= 0` 时首次 Put 可 panic。

## 目标

删除无仓库调用者、无测试且不属于切片核心职责的导出 `slice.LRUCache`。不提供新的缓存替代品；调用方如确有需求应选择独立、受测的缓存实现。

## 前置

1. T05-01 已完成，slice 文件和文档已去除失效并行部分。

## 允许改动

1. `slice/slice.go`：删除 `LRUCache`、`NewLRUCache`、`Get`、`Put`、`moveToEnd`。
2. `slice/使用说明.md`：删除 LRU API、示例、性能和故障排查内容。
3. `slice/slice_test.go` 仅在 import 清理或公共 API 回归需要时修改。

## 禁止改动

1. 不把 slice LRU 搬到另一个包继续保留。
2. 不用 deprecated wrapper、type alias 或 stub 假装兼容。
3. 不修改 `useragent` 内部缓存；留给 T05-03。
4. 不自行扫描/修改外部下游仓库。

<Context-Queries>

```bash
arc-idx search 'f:^slice/ LRUCache|NewLRUCache|moveToEnd|\.Put\(|\.Get\(' --repo . --context 5 --max 160 --format json
arc-idx search 'slice.NewLRUCache|slice.LRUCache' --repo . --context 3 --max 100 --format json
arc-idx search 'f:^slice/ LRU缓存|高效的LRU|容量' --repo . --context 3 --max 140 --format json
```

</Context-Queries>

## 当前行为

1. exported LRU 只在 `slice/使用说明.md` 出现，生产 Go 代码和测试无调用。
2. constructor 接受 0/负容量；Put 会读取空 `keys[0]` 并 panic。
3. Get 先读锁后写锁，中间状态语义复杂；moveToEnd 为 O(n)。
4. 在 slice 包中维护缓存扩大了工具包职责和维护成本。

## 固定兼容策略

1. 该导出 API 在目标 `v0.4.0` 直接删除。
2. 不提供自动迁移代码。
3. 本包文档不推荐特定第三方库；只说明调用方应在自己的基础设施层选择有容量校验和并发测试的缓存。
4. 删除事项由 T06-04 写入 CHANGELOG/MIGRATION；本任务只保证包内代码和说明不存在。

## 执行清单

- [ ] 使用 `rg` 确认仓库内除定义和文档外无调用；若发现真实 Go 调用，停止并把任务标 `[!]`，不得直接删除。
- [ ] 删除完整 LRU 类型与方法，不留下未使用 helper/import。
- [ ] 删除使用说明中的 LRU 快速开始、完整示例、实现片段、性能建议和 FAQ。
- [ ] 运行 `go doc ./slice`，确认不再导出 LRU 标识。
- [ ] 运行 slice 和全量测试，确认其他切片能力不依赖该类型。
- [ ] 把“v0.4.0 删除 slice.LRUCache，需要调用方迁移”作为交接项记录给 T06-04。

## 正向验证

1. Contains/Unique/Intersection/Union/Difference/Filter/Map/Reduce/Sort 全部仍可用。
2. `go doc ./slice` 只展示切片职责相关 API。

## 反向验证

1. Go 源码和 slice 文档中不存在 `LRUCache`、`NewLRUCache`、`moveToEnd`。
2. 不存在 deprecated wrapper 或 panic stub。
3. 无新增缓存依赖。

## 验证命令

```bash
gofmt -w slice/slice.go slice/slice_test.go
go test -count=1 ./slice
go test -race -count=1 ./slice
go test -count=1 ./...
go vet ./...
go doc ./slice
rg -n 'LRUCache|NewLRUCache|moveToEnd|LRU缓存' slice || true
git diff --check
```

## 不要做

1. 不返回一个永远 miss 的 stub 维持编译；这会隐藏调用方错误。
2. 不在 slice 包引入 `container/list` 重写它；目标就是删除职责外能力。
3. 不把外部兼容性未知当作保留缺陷的理由；目标版本已允许明确破坏性变更。
4. 不修改版本号或发布记录。

## 完成标准

1. 仓库内无真实调用者证据已记录。
2. slice LRU 代码和文档完全移除，无替代 stub。
3. 其他 slice API、race、全量测试和 vet 通过。
4. 破坏性迁移事项已记录到进度表供 T06-04 使用。
