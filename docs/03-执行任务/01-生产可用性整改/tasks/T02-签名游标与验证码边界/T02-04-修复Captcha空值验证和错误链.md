# T02-04 修复 F-OTP-001 Captcha 空值验证和错误链

状态：`[ ]`

## 来源

1. finding_id：F-OTP-001。
2. Handoff：[Captcha 空值与空值比较成功](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-otp-001--captcha-空值与空值比较成功)。
3. 权限类型：anonymous。
4. 数据产量：AUTH capability；可能影响登录、找回密码或敏感操作验证码分支。

## 目标

验证码缺失、存储未命中或过期导致 expected 为空时，任何输入都不能验证成功。有效等长输入使用恒定时间比较；随机源失败时既能判断 `ErrGenerationFailed`，也能追踪底层错误。

## 项目定位对齐

1. 影响用户角色：匿名登录/找回密码用户和验证码验证服务。
2. 信任边界：用户输入与服务端保存 expected code 的比较边界。
3. 本包只生成和比较；TTL、存储、限流、尝试次数和成功后消费属于下游。

## 允许改动

1. `captcha/captcha.go` 的 import、`Validate`、随机索引私有 helper 和错误包装。
2. `captcha/captcha_test.go`，可新增测试用 failing reader/helper。
3. `captcha/使用说明.md` 的验证安全边界。

## 禁止改动

1. 不实现 Redis、内存验证码仓库、HTTP handler、短信或邮件发送。
2. 不在本任务处理 Unicode 自定义字符集；留给 T02-05。
3. 不取消 trim 行为；比较前仍去除两端空白。
4. 不用普通 `==` 代替恒定时间比较。

<Context-Queries>

```bash
arc-idx search 'f:^captcha/ Validate|GenerateWithCustomCharset|Generate\(|ErrGenerationFailed|rand.Int|TrimSpace' --repo . --context 5 --max 140 --format json
arc-idx search 'f:^captcha/ 空字符串|expected|错误链|IsValidFormat' --repo . --context 3 --max 140 --format json
arc-idx ast --repo . --lang go 'func Validate($INPUT, $EXPECTED string) bool { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. `Validate("", "")` 和纯空白对纯空白都会返回 true，现有测试固化了该缺陷。
2. 比较使用普通字符串相等，耗时随不匹配位置变化。
3. 随机失败包装使用 `%v`，只能判断外层 sentinel，不能通过错误链访问底层 reader 错误。
4. 生产函数直接调用 `rand.Int(rand.Reader, ...)`，底层失败路径难以无副作用测试。

## 固定目标行为

1. 对 input 和 expected 分别 `strings.TrimSpace`。
2. 任一 trim 后为空，立即返回 false；两个都为空同样失败。
3. byte 长度不同立即返回 false。
4. 等长且非空时调用 `subtle.ConstantTimeCompare([]byte(input), []byte(expected)) == 1`。
5. 随机数 helper 接受 `io.Reader`，公开生成函数固定传 `rand.Reader`；测试通过私有 helper 注入 failing reader。
6. 随机错误使用同时保留 sentinel 和 cause 的 wrapping；在 Go 1.24 可使用两个 `%w`，测试必须同时满足 `errors.Is(err, ErrGenerationFailed)` 和 `errors.Is(err, injectedErr)`。

## 执行清单

- [ ] 导入 `crypto/subtle`，按固定顺序改写 `Validate`。
- [ ] 把现有空字符串测试预期从 true 改为 false，并新增纯空白组合。
- [ ] 抽取最小私有随机 helper，使默认数字生成与自定义生成都能复用底层错误包装；不要暴露测试专用公开 API。
- [ ] 将 `fmt.Errorf("%w: %v", ErrGenerationFailed, err)` 改为保留两层错误的写法。
- [ ] 新增 failing reader 测试，确认错误链同时命中 sentinel 和底层错误。
- [ ] 更新使用说明：expected 不存在/过期时直接失败，验证成功后下游必须原子消费并做限流。
- [ ] 删除“6 位足以抵御暴力破解”的无条件结论，改成只有短 TTL、尝试次数限制和成功消费同时存在时才成立。

## 正向测试

1. 相同非空数字验证码返回 true。
2. 两端空白被 trim 后，相同非空验证码返回 true。
3. 生成默认验证码行为、长度和字符范围保持不变。
4. 注入正常 reader 的私有 helper 可生成合法索引。

## 反向测试

1. `("", "")`、`("   ", "")`、`("", "   ")`、`("   ", "   ")` 全部 false。
2. 任一方为空、长度不同、内容不同均 false。
3. failing reader 返回的错误同时可匹配 `ErrGenerationFailed` 和注入错误。
4. 错误路径不得返回部分验证码。

## 验证命令

```bash
gofmt -w captcha/captcha.go captcha/captcha_test.go
go test -count=1 ./captcha -run 'Validate|Empty|Whitespace|GenerationError'
go test -race -count=1 ./captcha
go test -count=1 ./...
go vet ./...
git diff --check
```

## 不要做

1. 不把 empty expected 当作“验证码功能未启用”并返回成功。
2. 不在库内增加永久全局可替换的 `rand.Reader` 变量；避免测试注入造成并发污染。
3. 不吞掉底层随机源错误或只返回空字符串。
4. 不记录、打印或放进错误消息完整验证码。
5. 不宣称恒定时间比较替代 TTL、限流和一次性消费。

## 完成标准

1. 旧空值测试在目标语义下已翻转，所有空值组合稳定失败。
2. 非空等长比较使用 `subtle.ConstantTimeCompare`。
3. 随机错误链有正向断言且保留根因。
4. captcha 包、race、全量测试和 vet 通过。
