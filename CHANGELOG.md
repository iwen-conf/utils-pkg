# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

## [0.7.0]

### Added

- **JWT**:
  - Key rotation support via `JWTOptions.SigningKeyID`, `VerificationKeys`, and `LegacyVerificationKey`, plus typed sentinels `ErrInvalidKeyID`, `ErrDuplicateKeyID`, and `ErrUnknownKeyID`.
- **Pagination**:
  - Generic strongly-typed cursor helpers `EncodeCursor[T]` and `DecodeCursor[T]` with `ErrNilCodec`.

### Changed

- **URL -> `signedurl`**: Renamed package `github.com/iwen-conf/utils-pkg/url` to `github.com/iwen-conf/utils-pkg/signedurl` to avoid shadowing standard library `net/url`.
- **Crypto**: `NewBcryptHasher(cost)` now returns `(*BcryptHasher, error)` with fail-fast cost validation; `Encryptor` interface now includes `EncryptWithAAD`, `DecryptWithAAD`, `EncryptWithAADAndEncoding`, and `DecryptWithAADAndEncoding`.
- **Pagination**: `Base64JSONCodec.Decode` wraps base64/JSON errors with `%w` to preserve `errors.Is`/`errors.As` chains; `HMACCodec.Decode` uses zero-allocation `strings.Cut`.
- **User-Agent**: Global browser info cache is now private, strongly typed (`BrowserInfo` without `any`), lazily initialized via `sync.Once`, and uses single-pass regex matching.
- **JWT**:
  - Configuration is immutable after `NewTokenManager`; validation cache keys store `[32]byte` SHA-256 token digests instead of raw JWT strings; background cleanup goroutine starts on demand only when needed; `ErrorReporter` defaults to silent no-op when nil.
  - Renamed `RevokeTokenContext`, `RevokeSessionContext`, `IsTokenRevokedContext`, `IsSessionRevokedContext`, `CleanBlacklistContext`, and `GetBlacklistSizeContext` to `RevokeToken`, `RevokeSession`, `IsTokenRevoked`, `IsSessionRevoked`, `CleanBlacklist`, and `GetBlacklistSize`.

### Removed

- **Ptr**: Removed all 60 pre-generics typed pointer helpers; kept only `Of[T]`, `Value[T]`, `Deref[T]`, `Zero[T]`, and `ErrNilPointer`.
- **SignedURL**: Removed weak-typed `ParseURL`, `SerializeParams`, and `DeserializeParams`.
- **User-Agent**: Unexported `LRUCache`, `NewLRUCache`, `ShardedCache`, `NewShardedCache`, and `ErrInvalidCacheCapacity` / `ErrInvalidCacheTTL` / `ErrUninitializedCache`.
- **Crypto**: Removed `EncryptionMode`, `ModeGCM`, `ModeCFB`, `ErrUnsupportedEncryptionMode`, `NewAESEncryptorWithMode`, `NewBcryptHasherChecked`, `BenchmarkPasswordHashers`, `BenchmarkPasswordHashersChecked`, and `ErrInvalidBenchmarkIterations`.
- **JWT**: Removed `EnableLog`, runtime setters (`EnableCache`, `SetCacheTTL`, `SetCacheSize`, `SetTokenExpiry`, `SetIssuer`, `SetAudience`), and legacy/deprecated methods (`GenerateTokenPair`, `GenerateIdentityToken`, `GenerateIdentityTokenPair`, `ValidateToken`, `ValidateTokenContext`, `RefreshToken`, `RefreshTokenContext`, `IsBlacklisted`, and old no-context `RevokeToken`/`RevokeSession`/`CleanBlacklist`/`GetBlacklistSize`).
- `Version()` is `v0.7.0`.

## [0.6.0]

### Removed

- **Errors**: Removed `github.com/iwen-conf/utils-pkg/errors` (`RichError` and legacy `Error`/`Builder`/`Validator`). Application-specific error codes and client-safe messages should live in each service's internal package; all remaining `utils-pkg` modules use standard library `errors` (`%w`, `errors.Is`, `errors.As`).
- `Version()` is `v0.6.0`.

## [0.5.1]

MIT license in root `LICENSE` (SPDX `MIT`; copyright 2025-2026 iwen-conf). T06-06.

- `Version()` is `v0.5.1`.

## [0.5.0]

Technical candidate. Public release remains blocked until a license is chosen (T06-06).

### Added

- **JWT**:
  - Unified primary lifecycle APIs: `IssueTokenPair`, `IssueAccessToken`, `Authenticate`, `ValidateAccessToken`, `ValidateRefreshToken`, `RefreshTokenPair`, `ExtractIdentity`, and `RevokeSessionByID`.
  - Controllable clock injection via `JWTOptions.Clock` (plus `NewMemoryRevocationStoreWithClock` and `NewMemoryRefreshTokenUseStoreWithClock`) for deterministic time testing.
  - Optional `RefreshTokenUseReleaser` interface (implemented by `MemoryRefreshTokenUseStore`) to roll back JTI consumption if post-consume revocation storage fails transiently.
  - Typed sentinel errors: `ErrTokenTypeMismatch`, `ErrReservedClaimKey`, `ErrDuplicateClaimKey`, `ErrEmptySubject`, `ErrEmptyToken`, `ErrInvalidTokenFormat`, `ErrTokenTooLarge`, `ErrInvalidToken`, `ErrMissingTokenID`.
- **Crypto**:
  - AES-GCM Associated Authenticated Data (AAD) APIs: `EncryptWithAAD`, `DecryptWithAAD`, `EncryptWithAADAndEncoding`, `DecryptWithAADAndEncoding`.
  - `NewBcryptHasherChecked` with fail-fast cost validation and `BenchmarkPasswordHashersChecked` with explicit error propagation.
- **Ptr**:
  - Generic pointer helpers `ptr.Of[T]`, `ptr.Value[T]`, `ptr.Deref[T]`, and `ptr.Zero[T]`.

### Breaking & Security Fixes

- **JWT**:
  - **Token Type Isolation**: `ValidateToken` and `ValidateTokenContext` now strictly accept `AccessToken` only (rejecting `RefreshToken` with `ErrTokenTypeMismatch`). Callers needing to inspect or validate refresh tokens must explicitly call `ValidateRefreshToken` or `RefreshTokenPair`.
  - **HS256 + Required `exp`**: Only `HS256` is accepted; `exp` is mandatory; secret length requires `>= 32` bytes without character-class heuristics.
  - **Positive-Only Bounded O(1) Validation Cache**: Validation failures (such as pre-`nbf` clock skew or forged tokens) are never cached; `validUntil = min(now+CacheTTL, exp)`; `CacheSize` (including `1..4`) is strictly bounded via O(1) LRU; dynamic `EnableCache(false)`, `SetCacheSize`, and `SetCacheTTL` take immediate effect on cached entries; returned claims are deep-cloned.
  - **Reserved Claim & Profile Protection**: `CustomClaims` and `TokenProfile` cannot use reserved JWT claim keys (`iss`, `sub`, `aud`, `exp`, `nbf`, `iat`, `jti`, `type`, `sid`) or duplicate profile mappings. `claims.Identity()` automatically uses the manager's `TokenProfile` and strips profile keys from `Identity.CustomClaims` so `Identity` is round-trippable.
  - **Atomic Refresh Rotation & Safe Retry**: New tokens are pre-generated in memory before consuming `jti`; transient revocation failures roll back `Release` instead of falsely revoking user sessions on retry; replaying a consumed refresh token without `sid` still returns `ErrRefreshReuseDetected`.
- **URL**: v2 signatures bind method/scheme/host/path/query; 32-byte secrets; required `_exp`; dynamic build-time timestamp when `SetTimestamp` is unset; no legacy fallback.
- **Pagination**: HMAC codecs reject short/zero keys and uninitialized receivers; `OffsetRequest`/`OffsetResponse` prevent `int` overflow on large offsets; `Decode` bounds cursor length (`MaxCursorLength = 16384`).
- **Captcha**: Empty or whitespace expected/input never succeeds; custom charset is rune-based.
- **Crypto**: AES accepts only GCM, precomputes `cipher.AEAD`, and rejects nil/zero `AESEncryptor` (`ErrUninitializedEncryptor`); Argon2/scrypt validate resource bounds before KDF; `PasswordPolicy` uses immutable atomic config snapshots.
- **Errors**: `Is/As` match the standard library; default `RichError` JSON outputs only `code` and `msg`; `RichError.WithCode`/`WithMsg` and legacy `*Error` methods are nil-safe; stack capture uses `runtime.CallersFrames` and skips shortcut helper frames; `Clone` safely handles `nil` inside context maps/slices.
- **Slice**: `Union` and `Difference` always allocate independent slices and deduplicate properly when an input is empty; `Sort` uses `slices.Sort` for total `NaN` ordering; unused parallel helpers and `slice.LRUCache` removed.
- **User-Agent**: `LRUCache`/`ShardedCache` validate capacity/TTL on construction and handle nil receivers safely; IE (`MSIE` and `Trident/rv:`) version extraction fixed; oversized `User-Agent` inputs (`> 512` bytes) are truncated and excluded from the global cache; results remain heuristic only.
- `Version()` is `v0.5.0`.

## [0.4.0]

Technical candidate tagged at `9459a63`. Production-readiness remediations: HS256 JWT with required `exp`, URL v2 signatures, AES-GCM only, fail-closed captcha/KDF, standard error chains, and User-Agent heuristic contract.

## [0.3.0]

Previous tagged release.
