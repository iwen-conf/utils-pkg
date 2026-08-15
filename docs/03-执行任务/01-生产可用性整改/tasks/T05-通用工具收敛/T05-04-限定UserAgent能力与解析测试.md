# T05-04 GOV-001 限定 User-Agent 能力与解析测试

状态：`[ ]`

## 来源

1. governance_id：GOV-001；关联 F-UTIL-001 的 User-Agent 缓存整改。
2. Handoff：[文档、测试宣称和实现发生漂移](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#gov-001--文档测试宣称和实现发生漂移)。
3. 设计要求：User-Agent 结果只用于统计和展示，不用于权限、风控或机器人封禁。
4. 权限类型：service/maintainer。
5. 数据产量：展示/统计能力；User-Agent 是客户端可伪造字符串。

## 目标

让 `IsBrowser` 和 `GetBrowserInfo` 共享一套可测试的启发式解析事实，覆盖主流浏览器顺序和异常输入；文档明确结果不可用于鉴权、反作弊、机器人拦截或设备指纹。

## 前置

1. T05-03 已完成，缓存构造与并发语义稳定。

## 允许改动

1. `useragent/useragent.go` 的浏览器正则、解析 helper、`IsBrowser`、`GetBrowserInfo` 和内部默认缓存使用。
2. `useragent/useragent_test.go`。
3. `useragent/使用说明.md`。

## 禁止改动

1. 不实现设备指纹、IP 风控、自动封禁、可信 bot 验证或 Client Hints 协商。
2. 不新增外部 User-Agent 数据库依赖。
3. 不把“未识别”强制标记为攻击者/bot。
4. 不基于 UA 赋予或拒绝权限。

<Context-Queries>

```bash
arc-idx search 'f:^useragent/ chromeRegex|safariRegex|edgeRegex|botIdentifiers|commonBrowserIdentifiers|IsBrowser|GetBrowserInfo|extractBrowserInfo' --repo . --context 5 --max 220 --format json
arc-idx search 'f:^useragent/ isBrowserCache|browserInfoCache|is_bot|非浏览器|爬虫|安全|封禁' --repo . --context 4 --max 220 --format json
arc-idx ast --repo . --lang go 'func GetBrowserInfo($UA string) BrowserInfo { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. IsBrowser 与 GetBrowserInfo 分别缓存/计算，可能随规则演进产生不一致。
2. bot substring 优先返回 false，但任何客户端都能伪造或隐藏这些标识。
3. Safari 当前提取 `Safari/x` 引擎版本，而不是更有展示意义的 `Version/x`。
4. 测试只覆盖少量 happy path，没有 Opera、IE、CLI、大小写、畸形和一致性矩阵。
5. 文档示例把 `!IsBrowser` 写成 `is_bot`，容易被下游当成安全事实。

## 固定能力边界

1. `BrowserInfo` 是 best-effort 展示结果，不是可信身份。
2. `IsBrowser(ua)` 必须等于 `GetBrowserInfo(ua).IsBrowser`。
3. 未识别/空/CLI UA 返回 `IsBrowser:false`，但文档称“未识别为浏览器”，不能等价为恶意 bot。
4. bot substring 只影响启发式分类；文档必须说明可伪造。
5. 不导出 `IsBot`，避免制造错误安全承诺。

## 固定解析优先级

1. 明确 bot 标识。
2. Edge（`Edg/` 或兼容 `Edge/`）。
3. Opera（`OPR/`）。
4. Chrome（`Chrome/`；不得吞掉 Edge/Opera）。
5. Firefox（`Firefox/`）。
6. Internet Explorer（MSIE/Trident）。
7. Safari：同时满足 Safari 且不是 Chrome/Edge/Opera；优先从 `Version/` 提取展示版本。
8. 其他含浏览器通用标识但无法识别名称的输入可返回 `IsBrowser:true, Name:"Unknown"`；若保留空 Name，文档和测试必须固定一种，不能随机变化。

## 目标实现

1. 抽取单一私有 `parseBrowserInfo(userAgent string) BrowserInfo`。
2. `GetBrowserInfo` 使用一个 BrowserInfo cache；`IsBrowser` 直接复用 GetBrowserInfo 或同一解析结果，删除重复 bool cache。
3. 解析输出是值类型，缓存返回不会造成可变共享。
4. 正则和版本提取失败时返回空 Version，不 panic。

## 执行清单

- [ ] 先建立表驱动 UA fixture：Chrome/Edge/Opera/Firefox/Safari/IE/bot/curl/空/畸形。
- [ ] 抽取单一 parse helper并固定优先级，修复 Safari 版本提取。
- [ ] 移除重复 `isBrowserCache`，保证两个公开 API 一致。
- [ ] 增加大小写、重复调用、缓存命中和并发一致性测试。
- [ ] 更新使用说明所有 `is_bot: !isBrowser` 示例，改为 `is_browser`/browser info 展示，不做安全判断。
- [ ] 增加醒目边界：UA 可由客户端任意填写，不能用于登录、权限、风控、支付或可信 bot 判断。
- [ ] 删除未经基准验证的“高效”“自动优化”等绝对性能宣称；保留可复现 benchmark 命令即可。

## 正向测试

1. 六类浏览器返回正确 Name/Version/IsBrowser。
2. Edge/Opera UA 同时含 Chrome 时仍识别为 Edge/Opera。
3. Safari 使用 `Version/14.1.1` 一类展示版本。
4. 对所有 fixture，`IsBrowser == GetBrowserInfo.IsBrowser`。
5. 并发重复解析通过 race，缓存前后结果一致。

## 反向测试

1. 空、curl、任意畸形字符串和明确 bot 返回非浏览器但不 panic。
2. bot 名称大小写变化仍被启发式识别。
3. 文档和代码不出现“用 UA 鉴权/封禁”“非浏览器即 bot”的建议。
4. 不新增 `IsBot` 或安全评分字段。

## 验证命令

```bash
gofmt -w useragent/useragent.go useragent/useragent_test.go
go test -count=1 ./useragent -run 'Browser|UserAgent|Consistency|Safari|Edge|Opera|Bot|Unknown'
go test -race -count=1 ./useragent
go test -count=1 ./...
go vet ./...
rg -n 'is_bot.*!|鉴权|权限|封禁|反作弊|设备指纹|IsBot' useragent || true
git diff --check
```

## 不要做

1. 不把字符串启发式结果升级成安全控制。
2. 不把所有 unknown 归类为 bot 或爬虫。
3. 不维护两套解析/缓存导致结果分叉。
4. 不为追求完整浏览器数据库引入大范围依赖。
5. 不用测试 fixture 之外的猜测声称“支持所有浏览器”。

## 完成标准

1. 两个公开 API 共享单一解析事实并始终一致。
2. 主流、异常、优先级、缓存和并发路径有测试。
3. 文档明确非安全用途且无错误 bot/鉴权示例。
4. useragent 包、race、全量测试和 vet 通过。
