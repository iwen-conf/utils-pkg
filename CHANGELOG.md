# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

## [0.4.0] - 待发布

Technical candidate. Public release remains blocked until a license is chosen (T06-06).

### Breaking

- JWT: HS256 + required `exp`; secret length is 32 bytes without character-class checks; cache cannot outlive `exp` and returns cloned claims; refresh consume is a pluggable store; setters return errors; custom zero `JWTOptions` fail construction.
- URL: v2 signatures bind method/scheme/host/path/query; 32-byte secrets; required `_exp`; no legacy fallback.
- Pagination: HMAC codecs reject short/zero keys and uninitialized receivers.
- Captcha: empty or whitespace expected/input never succeeds; custom charset is rune-based.
- Crypto: AES accepts only GCM; Argon2/scrypt reject unbounded params before KDF; PasswordPolicy uses immutable Config; hasher constructors return error.
- Errors: `Is/As` match the standard library; default RichError JSON is only `code` and `msg`.
- Slice: unused parallel helpers and LRU are removed; `golang.org/x/exp` is no longer a direct dependency.
- User-Agent: LRU constructors fail on invalid capacity; TTL is `time.Duration`; results are not a security verdict.
- `Version()` is `v0.4.0`.

### Security

- Fail-closed refresh consume, JWT parse, URL signatures, captcha compare, and KDF parameter bounds.

## [0.3.0]

Previous tagged release.
