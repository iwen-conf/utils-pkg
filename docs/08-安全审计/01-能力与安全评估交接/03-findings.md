# Findings

## F-AUTH-001 — JWT 成功缓存可延长 token 实际有效期

- **Severity**: High
- **Permission class**: anonymous 或 user（持有曾经有效的 token）
- **Data yield**: AUTH capability，影响该 token 可访问的全部下游接口
- **Why it scores**: 缓存命中只按写入时间和 `CacheTTL` 判断，未以 token `exp` 为上限，过期凭证可能继续被接受。
- **Evidence**: `jwt/jwt.go:786-808` 的 `checkCache`；`jwt/jwt.go:812-826` 的 `cacheResult`。
- **Exploit conditions**: token 在过期前至少成功验证一次，且缓存 TTL 跨过 `exp`。
- **Blast radius**: 单 token；下游若复用同一 token，可覆盖其全部授权能力。
- **Detection gap**: 当前测试只验证 cache item TTL，没有验证 token expiry 与缓存共同作用。
- **Fix direction**: 成功缓存 `validUntil=min(cacheTTL, exp)`，命中复核绝对有效期并补回归测试。
- **Residual risk**: 下游仍需使用短 access token 和正确时钟。
- **Status**: confirmed

## F-AUTH-002 — 缓存返回可变 claims 指针，调用方可污染后续验证

- **Severity**: High
- **Permission class**: service / in-process library consumer
- **Data yield**: AUTH/BIZ capability only；可影响 subject、roles、scopes、tenant 等授权字段
- **Why it scores**: 首次解析结果同时写入缓存并返回，后续命中返回同一指针；调用方修改后不再受签名保护。
- **Evidence**: `jwt/jwt.go:610-623`、`jwt/jwt.go:651-653`、`jwt/jwt.go:804-808`、`jwt/jwt.go:822-825`。
- **Exploit conditions**: 同一进程中的调用路径修改返回 claims，或无意复用可变 map/slice。
- **Blast radius**: 同一 `TokenManager` 和 token 的后续请求。
- **Detection gap**: 当前测试未在返回后修改 claims 再次验证。
- **Fix direction**: 缓存写入和读取双向深克隆，覆盖嵌套 custom claims。
- **Residual risk**: 下游仍不应把返回对象当作跨请求共享可变状态。
- **Status**: confirmed

## F-AUTH-003 — JWT 注册 claim 解析错误被忽略且算法约束过宽

- **Severity**: Medium
- **Permission class**: service / token issuer
- **Data yield**: AUTH capability only
- **Why it scores**: `exp/nbf/iat/iss/aud/sub` getter 错误被丢弃，解析只验证任意 HMAC 方法，并未要求 `exp`。
- **Evidence**: `jwt/jwt.go:97-102` 忽略 getter error；`jwt/jwt.go:657-667` 只判断 `SigningMethodHMAC`。
- **Exploit conditions**: token 已能通过共享密钥签名，或来自错误配置/兼容 issuer。
- **Blast radius**: 该 manager 接受的 token 集合。
- **Detection gap**: fuzz 关注不 panic，未断言 malformed registered claims 必须失败。
- **Fix direction**: 检查每个 getter error，限定 HS256，要求 exp，增加独立 expected issuer/audience。
- **Residual risk**: HS256 仍依赖密钥安全和服务间不复用密钥。
- **Status**: confirmed

## F-AUTH-004 — Refresh 防重放默认状态仅在单进程内原子

- **Severity**: High
- **Permission class**: user（持有被窃取或并发使用的 refresh token）
- **Data yield**: AUTH capability，可能产生多组有效 token
- **Why it scores**: 外部 `RevocationStore` 可共享，但已消费 JTI 仍保存在 `TokenManager` 本地分片 map；多实例并发可各自首次消费。
- **Evidence**: `jwt/jwt.go:227-229`、`jwt/jwt.go:300-305`、`jwt/jwt.go:361-399`。
- **Exploit conditions**: 多实例部署、同一 refresh token 并发到不同实例且旧 token 尚未完成共享吊销。
- **Blast radius**: 单 session；可能签发多组 access/refresh token。
- **Detection gap**: 当前测试只覆盖进程内和共享吊销，不覆盖共享原子消费。
- **Fix direction**: 独立可插拔的原子 refresh 消费存储；默认内存实现明确仅单进程。
- **Residual risk**: 外部实现必须真正原子，普通 Get-then-Set 不合格。
- **Status**: confirmed

## F-AUTH-005 — JWT 生命周期和存储错误存在静默或并发契约缺口

- **Severity**: Medium
- **Permission class**: service / config-ops
- **Data yield**: availability and AUTH capability
- **Why it scores**: 多个 setter 无同步，`Shutdown` 重复 close 可 panic，部分无 error API 把存储失败变成 `true`、`0` 或仅在默认关闭的日志中记录。
- **Evidence**: `jwt/jwt.go:451-495`、`jwt/jwt.go:938-943`、`jwt/jwt.go:997-1023`。
- **Exploit conditions**: 并发动态配置、重复关闭或外部存储故障。
- **Blast radius**: 当前 manager 的全部调用。
- **Detection gap**: race 测试未并发调用 setter/Shutdown；错误 API 测试接受吞错语义。
- **Fix direction**: 幂等关闭、配置同步/不可变、增加返回 error 的权威 API。
- **Residual risk**: 下游仍需监控外部存储可用性。
- **Status**: confirmed

## F-SIGN-001 — URL 签名未绑定方法、主机和路径

- **Severity**: High
- **Permission class**: anonymous（持有一个合法签名 URL）
- **Data yield**: BIZ/AUTH capability；实际 L1–L3 取决于下游受保护端点
- **Why it scores**: HMAC 输入只有 timestamp 与 query，相同 query/signature 可被搬到其他 host/path；文档却将其描述为 API 安全保护。
- **Evidence**: `url/url.go:108-118`、`url/url.go:317-329`。
- **Exploit conditions**: 下游只调用 `ValidateSignature` 决定资源/接口访问，并有其他路径接受相同查询结构。
- **Blast radius**: 使用同一 secret 的全部签名 URL 端点。
- **Detection gap**: 当前测试只修改 secret/时间，不修改 path/host/method。
- **Fix direction**: v2 canonical request 绑定 method/scheme/host/path/query，默认不回退 legacy。
- **Residual risk**: 下游仍需做业务授权和一次性/重放控制。
- **Status**: confirmed

## F-SIGN-002 — HMAC 游标弱密钥和零值仍可工作

- **Severity**: Medium
- **Permission class**: user
- **Data yield**: BIZ/PII cursor metadata，通常 L1–L2
- **Why it scores**: 短密钥构造返回非 nil codec；导出类型零值以空密钥执行 HMAC，违反 fail-fast。
- **Evidence**: `pagination/cursor.go:106-119`、`pagination/cursor.go:122-159`；测试在 `pagination/cursor_test.go:90-97` 固化“警告但可用”。
- **Exploit conditions**: 调用方忽略 error，或直接声明零值 `HMACCodec`。
- **Blast radius**: 使用该 codec 的 cursor 完整性。
- **Detection gap**: 无零值拒绝测试。
- **Fix direction**: 密钥不足返回 nil/error，方法入口验证状态，复制密钥。
- **Residual risk**: cursor payload 仍不应承载秘密。
- **Status**: confirmed

## F-OTP-001 — Captcha 空值与空值比较成功

- **Severity**: High
- **Permission class**: anonymous
- **Data yield**: AUTH capability only
- **Why it scores**: 下游在验证码缺失/过期时若把 expected 留空，空输入可被判定为验证成功。
- **Evidence**: `captcha/captcha.go:165-171`；`captcha/captcha_test.go:114` 明确期望 `Validate("", "") == true`。
- **Exploit conditions**: 下游没有在调用前独立拒绝空 expected/input。
- **Blast radius**: 使用该函数的登录、重置或敏感操作验证码分支。
- **Detection gap**: 当前测试将缺陷视为正确行为。
- **Fix direction**: 任一空值失败，等长时恒定时间比较；更新文档强调库不负责 TTL/尝试次数。
- **Residual risk**: 下游必须在成功后消费验证码并做限流。
- **Status**: confirmed

## F-CRYPTO-001 — Argon2/scrypt 验证接受无上限的编码参数

- **Severity**: Medium
- **Permission class**: service；若 hash 可由外部提交则可为 anonymous
- **Data yield**: availability capability only
- **Why it scores**: 编码 hash 中的 memory/iterations/N/r/p/key length 直接进入 KDF，Argon2 parallelism 还从 uint32 截断为 uint8，可造成资源耗尽或错误参数。
- **Evidence**: `crypto/argon2.go:114-150`、`crypto/scrypt.go:83-105`。
- **Exploit conditions**: 验证不可信、损坏或被篡改的编码 hash。
- **Blast radius**: 当前进程 CPU/内存和认证可用性。
- **Detection gap**: 测试主要覆盖正常参数和格式，不覆盖资源上限。
- **Fix direction**: 统一参数验证、长度上限、溢出检查，在计算前拒绝。
- **Residual risk**: 合法高成本参数仍需容量规划。
- **Status**: confirmed

## F-CRYPTO-002 — PasswordPolicy 的并发与 Unicode 契约不成立

- **Severity**: Medium
- **Permission class**: service
- **Data yield**: AUTH policy integrity capability
- **Why it scores**: `SetDisallowWords` 分两次锁更新，验证只复制 regex slice 后读取未锁的公开 words；直接字段修改也绕过锁。长度按字节而字符检查按 rune。
- **Evidence**: `crypto/password.go:12-23`、`crypto/password.go:44-60`、`crypto/password.go:64-121`。
- **Exploit conditions**: 并发更新策略或 Unicode 密码。
- **Blast radius**: 当前 policy 的密码接受/拒绝结果，可能 panic 或误判。
- **Detection gap**: race 测试未覆盖并发 Set/Validate；无 Unicode 长度测试。
- **Fix direction**: 私有不可变配置/原子替换，克隆输入，按 rune 计数。
- **Residual risk**: 复杂密码规则本身不替代泄漏密码检查。
- **Status**: confirmed

## F-CRYPTO-003 — AES mode API、日志和优化状态误导调用方

- **Severity**: Medium
- **Permission class**: service / config-ops
- **Data yield**: SECRET/BIZ confidentiality capability
- **Why it scores**: 构造器接受 CFB mode 但始终使用 GCM，并直接打印 stdout；缓存 key、锁和 bufferPool 不参与真实安全/性能路径。
- **Evidence**: `crypto/aes.go:58-91`、`crypto/aes.go:94-100`、`crypto/aes.go:129-136`、`crypto/crypto.go:8-19`。
- **Exploit conditions**: 调用方依赖 mode 参数或日志输出，或根据文档理解错误。
- **Blast radius**: API 行为和运维日志；当前实际密文仍为 GCM。
- **Detection gap**: 测试未断言 CFB 必须失败，也未检测 stdout。
- **Fix direction**: 只允许 GCM，非法 mode 返回错误，删除死状态并同步文档。
- **Residual risk**: 密钥来源和轮换仍由下游负责。
- **Status**: confirmed

## F-ERR-001 — RichError JSON 自动包含底层 cause

- **Severity**: Medium
- **Permission class**: anonymous 或 user（若错误对象直接进入 HTTP JSON）
- **Data yield**: SECRET/PII/BIZ，通常 L1–L2，依赖下游 cause 内容
- **Why it scores**: 虽然 cause 字段不导出且注释称“不给前端看”，自定义 `MarshalJSON` 主动写出 `cause.Error()`。
- **Evidence**: `errors/rich_error.go:94-115`。
- **Exploit conditions**: Controller 或通用响应直接 JSON marshal `RichError`。
- **Blast radius**: 单次错误响应，可包含 SQL、RPC 地址、内部标识或用户数据。
- **Detection gap**: 当前测试断言 cause 出现在 JSON，没有响应安全测试。
- **Fix direction**: 默认 JSON 只输出 code/msg；详细 cause 仅显式日志方法或 `%+v`。
- **Residual risk**: 调用方仍可能手动把 cause 写入响应。
- **Status**: confirmed

## F-ERR-002 — 自定义 Is/As 不遵守标准错误链

- **Severity**: Medium
- **Permission class**: service
- **Data yield**: correctness/availability capability only
- **Why it scores**: 伪 `Is` 最终执行直接相等，伪 `As` 只遍历特定自定义链；`GetCode` 等也只看顶层，导致 `%w` 包装后分类和处理失效。
- **Evidence**: `errors/error_utils.go:10-66`、`errors/error_utils.go:68-175`。
- **Exploit conditions**: 任意标准 `%w`、第三方 error 或多层包装。
- **Blast radius**: 重试、HTTP 映射、错误分类和根因判断。
- **Detection gap**: 相关旧错误系统覆盖率为 0%。
- **Fix direction**: 委托标准库 `errors.Is/As/Unwrap`，另设 `IsCode`，补全链式测试。
- **Residual risk**: 两套错误模型在未来大版本前仍需迁移期。
- **Status**: confirmed

## F-UTIL-001 — 自研 LRU 对无效容量和并发更新不安全

- **Severity**: Low
- **Permission class**: service
- **Data yield**: availability capability only
- **Why it scores**: capacity <= 0 时 Put 访问空 keys 并 panic；User-Agent Get 在释放读锁后可能删除并发写入的新值。
- **Evidence**: `slice/slice.go:306-349`、`useragent/useragent.go:86-158`。
- **Exploit conditions**: 错误容量配置或并发过期/写入。
- **Blast radius**: 当前进程缓存和可用性。
- **Detection gap**: slice LRU 完全无测试；useragent 缺少无效容量与并发语义测试。
- **Fix direction**: 删除未使用 slice LRU；User-Agent 用明确失败构造和 O(1) list+map 实现。
- **Residual risk**: User-Agent 识别仍是启发式结果。
- **Status**: confirmed

## GOV-001 — 文档、测试宣称和实现发生漂移

- **Severity**: Info
- **Permission class**: config-ops / maintainer
- **Data yield**: capability only
- **Evidence**: README 链接不存在的 `errors/使用说明.md`；slice 文档宣称自动并行但函数私有且无调用；crypto 文档称 `NewAESEncryptor` 使用 CFB，而代码使用 GCM；ptr 文档宣称 100% 覆盖，项目总覆盖率仅 66.6%。
- **Fix direction**: 以实现和验证结果重写文档，禁止没有门禁的绝对宣称。
- **Status**: confirmed

## GOV-002 — 发布与供应链治理不足

- **Severity**: Info
- **Permission class**: config-ops / maintainer
- **Data yield**: capability only
- **Evidence**: tag 为 `v0.3.0`，`main.go` 返回 `v0.2.8`；缺少 CI、CHANGELOG、LICENSE；直接依赖存在可更新版本。
- **Fix direction**: CI、覆盖率、govulncheck、依赖升级、CHANGELOG、版本测试和许可证决策。
- **Status**: confirmed
