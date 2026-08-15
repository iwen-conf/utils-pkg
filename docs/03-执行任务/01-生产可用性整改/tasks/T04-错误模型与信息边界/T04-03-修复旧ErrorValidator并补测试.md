# T04-03 修复 F-ERR-002 旧 Error/Validator 正确性并补测试

状态：`[ ]`

## 来源

1. finding_id：F-ERR-002 的旧错误体系覆盖与正确性部分。
2. Handoff：[自定义 Is/As 不遵守标准错误链](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-err-002--自定义-isas-不遵守标准错误链)。
3. 设计要求：`In/NotIn` 不得对不可比较值使用 `==`；长度与空值判断覆盖 Unicode 和全部数值种类。
4. 权限类型：service。
5. 数据产量：correctness/availability capability。

## 目标

在弃用 legacy API 前先保证它不 panic、不误判 Unicode/零值，并用真实行为测试覆盖 Error、Builder、Validator 和 utility 核心路径。不得用只调用 getter 的低价值测试虚增覆盖率。

## 前置

1. T04-01 已提供标准 Is/As/IsCode 语义。
2. T04-02 已固定 RichError JSON，本任务不再修改该契约。

## 允许改动

1. `errors/error_validation.go`。
2. `errors/error_builder.go`、`errors/errors.go`、`errors/error_utils.go` 中被新增测试证明存在的 legacy 正确性问题。
3. 新增 `errors/error_validation_test.go`、`errors/legacy_error_test.go`、`errors/error_utils_test.go`；避免和 T04-01 测试重名。

## 禁止改动

1. 不删除 legacy 类型或公开函数。
2. 不在本任务做文档弃用；留给 T04-04。
3. 不改变错误 code 常量值和 Category/Severity 文本。
4. 不把字段验证升级成通用 schema 框架或第三方 validator 依赖。

<Context-Queries>

```bash
arc-idx search 'f:^errors/ Validator|ValidationError|isEmpty|MinLength|MaxLength|Length|In\(|NotIn\(|ValidateLength' --repo . --context 5 --max 240 --format json
arc-idx search 'f:^errors/ NewBuilder|WrapWithType|Merge|Clone|deepCopyValue|GetErrors|GetError' --repo . --context 4 --max 240 --format json
arc-idx ast --repo . --lang go 'func isEmpty($VALUE interface{}) bool { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. `In/NotIn` 对 interface 值直接 `==`；slice/map/function 等不可比较动态值会 panic。
2. `MinLength/MaxLength/Length/ValidateLength` 按字节计数。
3. `isEmpty` 的多类型 type switch 把动态数值保存在 interface 中再与 `0` 比较，部分具体类型可能误判；typed nil、不同 slice/map 类型也未覆盖。
4. legacy Error/Builder/Validator 大量公开路径没有测试，项目级覆盖率只有 17.8%。
5. `Clone/deepCopyValue` 对非 string-key map 或具名 slice 可能改变类型或 panic；目标是用 reflect 递归复制 map/slice/array/pointer/interface 并保留具体类型，其他类型原值返回，不允许 panic。

## 固定 Validator 规则

1. 长度规则统一按 Unicode rune 数计算。
2. `In/NotIn` 使用 `reflect.DeepEqual` 比较任意值；不得对动态值使用 `==`。
3. `isEmpty` 精确定义：
   - nil interface、nil pointer/interface/slice/map/channel/function 为 empty；
   - trim 后空 string 为 empty；
   - 所有有符号/无符号整数的 0、float 的 0、false 为 empty；
   - array/slice/map/string 长度 0 为 empty；
   - 非 nil pointer、非空集合和其他 struct 为 non-empty。
4. `Email/URL/Regex/Numeric/Integer/Date/Before/After/Custom` 保持现有业务语义，只补正反测试。
5. ValidationError 的 Field/Rule/Value/Params 与内嵌 Error.Context 保持一致。

## legacy Error 测试矩阵

1. `New/NewWithDetails/Wrap/WrapWithDetails` 字段和 Unwrap。
2. `FromType/WrapWithType` 的 code、severity、category。
3. Builder 的 Code/Message/Details/Wrap/Context/Severity/Category。
4. `Merge` 对 0、1、多个、nil 混合 error。
5. `Clone` 对嵌套 `map[string]any`、`[]any` 的隔离；若发现非 string-key map 会 panic，修复为保留具体类型的递归 clone 或明确返回原不可变值，不能 panic。
6. `IsRetryable/IsCritical/Format/Compare` 的正反路径。

## 执行清单

- [ ] 先为 In/NotIn 的 slice/map 输入写会触发旧 panic 的测试，再用 DeepEqual 修复。
- [ ] 抽取 rune length helper，四个长度 API 共用。
- [ ] 用 reflect.Kind 安全实现 isEmpty，所有 nilable kind 调用 `IsNil` 前先确认 kind，避免反射 panic。
- [ ] 补齐 Validator fluent chain、单错误/多错误 GetError、Clear 和 convenience functions 测试。
- [ ] 补 legacy constructors、builder、wrap/unwrap、merge/clone、format/category 测试。
- [ ] 重写 clone 反射逻辑：保留 map/slice/array/pointer/interface 的具体类型并递归复制；struct、scalar、func、channel 等按原值返回，不用 recover 掩盖。
- [ ] 所有 error 判断使用标准库 `errors.Is/As` 或本包 `IsCode`，禁止直接比较。
- [ ] 运行 coverage，确认新增测试覆盖错误/边界分支，而非只覆盖 happy path。

## 正向测试

1. 中文/emoji 长度按 rune 正确通过 Min/Max/Exact/ValidateLength。
2. In/NotIn 可比较和不可比较值都返回正确结果且不 panic。
3. int8/int64/uint32/float32 的零值、typed nil slice/map/pointer 被 Required 判空。
4. non-zero 数值、非空具名 slice/map、非 nil pointer 通过 Required。
5. 多 ValidationError 合并后包含字段、数量和消息；Clear 后无错误。
6. Clone 后修改嵌套 context 不影响原对象。

## 反向测试

1. Unicode rune 数超下限/上限返回对应 rule。
2. In 未命中、NotIn 命中产生 ValidationError。
3. 非法 regex、email、URL、numeric、integer、date 和 Custom=false 产生对应 rule。
4. reflect 输入包括 func/channel/typed nil 时不 panic。
5. `%w` 包裹后的 legacy Error 仍被 T04-01 helper 正确识别。

## 验证命令

```bash
gofmt -w errors/*.go
go test -count=1 ./errors -run 'Validator|Validation|Legacy|Builder|Clone|Merge|Unicode|Empty|In|NotIn'
go test -race -count=1 ./errors
go test -count=1 -cover ./errors
go test -count=1 ./...
go vet ./...
git diff --check
```

## 不要做

1. 不用 recover 把 interface 比较或 reflect panic 转成 validation failure。
2. 不把 Unicode 字符数继续写成 byte 长度。
3. 不为了覆盖率逐行调用无断言 API。
4. 不改变 code/category/severity 常量来让测试更容易。
5. 不引入第三方验证库扩大任务范围。

## 完成标准

1. In/NotIn、isEmpty、Unicode 长度无 panic且有正反测试。
2. legacy 核心构造、包装、Builder、Validator、Merge/Clone 均有行为断言。
3. errors 包覆盖率达到 T06-02 后续门槛所需基础，且测试能在旧缺陷上失败。
4. race、全量测试和 vet 通过，进度表同步。
