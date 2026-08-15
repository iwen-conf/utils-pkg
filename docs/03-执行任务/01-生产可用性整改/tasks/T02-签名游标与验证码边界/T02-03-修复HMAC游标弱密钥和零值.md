# T02-03 修复 F-SIGN-002 HMAC 游标弱密钥和零值

状态：`[ ]`

## 来源

1. finding_id：F-SIGN-002。
2. Handoff：[HMAC 游标弱密钥和零值仍可工作](../../../../08-安全审计/01-能力与安全评估交接/03-findings.md#f-sign-002--hmac-游标弱密钥和零值仍可工作)。
3. 权限类型：user。
4. 数据产量：BIZ/PII cursor metadata，通常为 L1–L2。

## 目标

只有通过成功构造、持有独立 32 字节以上密钥的 `HMACCodec` 才能编码和解码。短密钥、零值、nil receiver 和未初始化内部 codec 必须明确失败，不能用空密钥继续签名。

## 项目定位对齐

1. 影响用户角色：使用 cursor 翻页的下游用户和服务开发者。
2. 信任边界：客户端可修改的不透明 cursor 回到服务端时的完整性校验。
3. cursor 仍不得承载密码、token 或其他秘密；HMAC 只防篡改，不加密 payload。

## 允许改动

1. `pagination/cursor.go` 中 HMAC 哨兵错误、`HMACCodec`、`NewHMACCodec`、`Encode`、`Decode`。
2. `pagination/cursor_test.go`。
3. `pagination/使用说明.md` 的构造和错误处理示例。

## 禁止改动

1. 不修改 `Base64JSONCodec` 的编码格式。
2. 不改变 cursor wire format `v1.{payload}.{sig}`。
3. 不增加加密；本任务只保证完整性。
4. 不返回“非 nil codec + 警告 error”。

<Context-Queries>

```bash
arc-idx search 'f:^pagination/ HMACCodec|NewHMACCodec|ErrHMACKeyTooShort|ErrEmptyHMACKey|Base64JSONCodec' --repo . --context 5 --max 140 --format json
arc-idx search 'f:^pagination/ warning|警告|v1\.|inner == nil|c.key' --repo . --context 4 --max 140 --format json
arc-idx ast --repo . --lang go 'func NewHMACCodec($KEY []byte) (*HMACCodec, error) { $$$ }' --format json
```

</Context-Queries>

## 当前行为

1. 1–31 字节密钥返回可工作的 codec 和 `ErrHMACKeyTooShort`。
2. constructor 直接保存调用方 `key` slice，调用方后续修改可改变签名行为。
3. `HMACCodec{}` 会在方法内自动补 `Base64JSONCodec`，然后用空 key 计算 HMAC。
4. nil receiver 会 panic。

## 固定目标 API 与错误

1. 保留签名：

   ```go
   func NewHMACCodec(key []byte) (*HMACCodec, error)
   ```

2. 空 key 返回 `nil, ErrEmptyHMACKey`。
3. 1–31 字节 key 返回 `nil, ErrHMACKeyTooShort`；修改错误注释，删除“recommend/警告但可用”语义。
4. 新增 `ErrUninitializedHMACCodec`，用于 nil receiver、零值或 `inner == nil/key < 32` 的方法调用。
5. constructor 使用 `append([]byte(nil), key...)` 或 `bytes.Clone` 保存独立密钥副本。

## 执行清单

- [ ] 修改 constructor：所有失败都返回 nil codec，成功时复制 key 并设置 `Base64JSONCodec{}`。
- [ ] 新增私有 `validate()` helper；`Encode`、`Decode` 第一行调用，失败直接返回哨兵错误。
- [ ] 删除方法中 `if c.inner == nil { c.inner = ... }` 的自愈逻辑。
- [ ] 保持 HMAC-SHA256、Raw URL Base64 和 `v1.payload.sig` 格式不变。
- [ ] 更新现有“短 key 仍返回 codec”的测试为 `codec == nil`。
- [ ] 新增 constructor key-copy 测试：构造后修改原 slice，已生成/新生成 cursor 的验证行为不受影响。
- [ ] 新增 nil receiver、零值、私有字段被清空三类方法测试，均返回 `ErrUninitializedHMACCodec` 且不 panic。
- [ ] 更新使用说明，示例在 error 非 nil 时立即停止，不能继续使用 codec。

## 正向测试

1. 32、48、64 字节 key 均成功构造并完成 round trip。
2. constructor 后修改输入 key 不改变 codec 的签名结果。
3. 同一 key 的现有 v1 cursor 格式仍可解码。

## 反向测试

1. nil、空、1 字节、31 字节 key 均返回 nil codec 和对应错误。
2. `var codec *HMACCodec` 调用 Encode/Decode 不 panic，返回 `ErrUninitializedHMACCodec`。
3. `codec := HMACCodec{}` 调用 Encode/Decode 返回同一哨兵错误。
4. payload/signature/version 篡改仍分别返回 `ErrInvalidSignature` 或 `ErrInvalidCursorFormat`。

## 验证命令

```bash
gofmt -w pagination/cursor.go pagination/cursor_test.go
go test -count=1 ./pagination -run 'HMAC|Cursor|Key|Zero|Nil|Tamper'
go test -race -count=1 ./pagination
go test -count=1 ./...
go vet ./...
git diff --check
```

## 不要做

1. 不保留短 key codec 供“测试环境”使用。
2. 不在 Encode/Decode 内静默生成随机 key；这会导致不可解码且隐藏配置错误。
3. 不把 cursor payload 当作秘密数据或在文档中称为加密。
4. 不用 recover 把 nil panic 转成成功或空 cursor。
5. 不直接保存调用方可变 key slice。

## 完成标准

1. 所有无效构造路径返回 nil 对象和可判断错误。
2. 零值/nil receiver 不再签名、不 panic。
3. key-copy、篡改、round-trip 和 race 测试全部通过。
4. 文档不再展示“收到 warning 后继续用 codec”。
