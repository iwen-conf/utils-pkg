# T02-05 修复 Captcha Unicode 自定义字符集

状态：`[ ]`

## 来源

1. 类型：正确性补充任务，不是独立安全 finding。
2. 设计来源：[Pagination 与 Captcha 目标设计](../../../../01-系统设计/01-生产可用性整改设计边界.md#pagination-与-captcha-目标设计)。
3. 关联 finding：F-OTP-001；同属验证码输入边界，但本任务只处理生成正确性。
4. 影响角色：使用自定义中文、符号或多语言字符集的下游开发者。

## 目标

`GenerateWithCustomCharset` 按 Unicode rune 选择字符，输出始终是有效 UTF-8，长度按 rune 数计算；重复字符去重，少于两个唯一字符时失败。

## 前置

1. T02-04 已完成，并提供可注入 reader 的私有随机 helper 和正确错误链。

## 允许改动

1. `captcha/captcha.go` 的自定义字符集校验、去重和生成 helper。
2. `captcha/captcha_test.go`。
3. `captcha/使用说明.md` 的自定义字符集章节。

## 禁止改动

1. 不改变默认 `DigitCharset` 和 `Generate/Generate4/Generate6/Generate8` 的输出集合。
2. 不增加大小写归一化、相似字符过滤或语言特定规则。
3. 不把字节长度继续当作字符数量。
4. 不允许单一唯一 rune 的字符集生成可预测验证码。

<Context-Queries>

```bash
arc-idx search 'f:^captcha/ GenerateWithCustomCharset|WriteByte|charsetLen|len\(charset\)|ContainsRune' --repo . --context 5 --max 120 --format json
arc-idx search 'f:^captcha/ MinLength|MaxLength|utf8|rune|自定义字符集' --repo . --context 4 --max 120 --format json
arc-idx ast --repo . --lang go 'func GenerateWithCustomCharset($LENGTH int, $CHARSET string) (string, error) { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. charset 长度按字节计算，随机索引可能落在 UTF-8 多字节序列中间。
2. `WriteByte` 会输出无效 UTF-8 或拆碎字符。
3. 单字符字符集被允许，验证码完全可预测。
4. 重复字符会无意改变采样权重，文档未说明。

## 固定目标错误与规则

1. 新增 `ErrInvalidCharset`，用于空 charset、非法 UTF-8 或少于两个唯一 rune；详细原因用 `%w` 包装并保留。
2. 先用 `utf8.ValidString` 检查输入；无效 UTF-8 不做 replacement rune 容错。
3. 转为 `[]rune`，按首次出现顺序去重。
4. 去重后少于 2 个 rune 返回错误。
5. 每个输出位置从去重后的 rune slice 等概率选择。
6. 输出长度使用 rune 个数；`utf8.RuneCountInString(code) == length`。
7. `length` 仍受现有 `MinLength`/`MaxLength` 约束。

## 执行清单

- [ ] 新增私有 `uniqueRunes(charset string) ([]rune, error)` 或等价 helper，负责 UTF-8 校验和稳定去重。
- [ ] `GenerateWithCustomCharset` 在任何随机读取前完成 length 和 charset 校验。
- [ ] 使用 `strings.Builder.WriteRune` 或预分配 `[]rune`，禁止字节索引和 `WriteByte`。
- [ ] 复用 T02-04 的随机 helper和错误链，不能复制一份不同实现。
- [ ] 修改现有“单字符字符集成功”测试为明确失败。
- [ ] 新增中文、emoji、组合 ASCII+Unicode、重复 rune 和非法 UTF-8 测试。
- [ ] 更新文档：length 指 rune 数；重复字符不增加权重；至少两个唯一字符。

## 正向测试

1. charset `"甲乙丙丁"` 生成 4、6、12 rune 的合法 UTF-8 字符串。
2. charset `"A界🙂9"` 输出的每个 rune 都属于原集合。
3. charset `"AABB界界"` 去重后按 `A/B/界` 生成，结果不含其他 rune。
4. 默认数字生成测试保持通过。

## 反向测试

1. 空 charset、单 rune、只重复一个 rune（如 `"AAAA"`）失败并匹配 `ErrInvalidCharset`。
2. `string([]byte{0xff, 0xfe})` 失败，不生成 replacement rune。
3. length 小于 4 或大于 12 仍匹配 `ErrInvalidLength`。
4. 随机 reader 失败时返回空结果并保留 T02-04 定义的错误链。

## 验证命令

```bash
gofmt -w captcha/captcha.go captcha/captcha_test.go
go test -count=1 ./captcha -run 'CustomCharset|Unicode|Rune|UTF8|Duplicate'
go test -race -count=1 ./captcha
go test -count=1 ./...
go vet ./...
git diff --check
```

## 不要做

1. 不通过 `[]byte(charset)` 采样多语言字符。
2. 不把非法 UTF-8 静默替换成 `utf8.RuneError`。
3. 不把重复 rune 当成隐式权重配置；本 API 没有权重语义。
4. 不新增“自动移除易混淆字符”规则，避免改变调用方明确给出的集合。

## 完成标准

1. 所有自定义输出均为有效 UTF-8，长度严格按 rune 计。
2. 少于两个唯一 rune 和非法 UTF-8 在随机读取前失败。
3. 默认数字验证码行为没有回归。
4. 正反测试、race、全量测试和 vet 通过，进度表同步。
