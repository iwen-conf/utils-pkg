# T02 签名、游标与验证码边界

## 目标

让 URL、cursor 和 captcha 的默认行为严格绑定其安全对象，拒绝空值、弱密钥和未初始化状态。

## 已知范围

1. `url/url.go`、`url/url_test.go`、`url/使用说明.md`
2. `pagination/cursor.go`、`pagination/cursor_test.go`、`pagination/使用说明.md`
3. `captcha/captcha.go`、`captcha/captcha_test.go`、`captcha/使用说明.md`

## 子任务索引

| 状态 | 子任务 | Finding |
| --- | --- | --- |
| `[ ]` | [T02-01 实现 URL v2 规范签名](T02-01-实现URLv2规范签名.md) | F-SIGN-001 |
| `[ ]` | [T02-02 收紧 URL 输入时间与迁移边界](T02-02-收紧URL输入时间与迁移边界.md) | F-SIGN-001 |
| `[ ]` | [T02-03 修复 HMAC 游标弱密钥和零值](T02-03-修复HMAC游标弱密钥和零值.md) | F-SIGN-002 |
| `[ ]` | [T02-04 修复 Captcha 空值验证和错误链](T02-04-修复Captcha空值验证和错误链.md) | F-OTP-001 |
| `[ ]` | [T02-05 修复 Captcha Unicode 字符集](T02-05-修复CaptchaUnicode字符集.md) | correctness |

## 验收标准

1. 修改 method/host/path/query 任一签名组成部分都会验证失败。
2. 默认验证不接受旧签名格式、弱密钥或无有限有效期 URL。
3. HMACCodec 零值和短密钥不可工作。
4. Captcha 空值永不成功，自定义 Unicode 字符集输出有效 UTF-8。
