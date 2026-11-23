# AGENTS

Go 1.24 utility module. Prefer fail-fast constructors, standard `%w` + `errors.Is`/`errors.As`, and no stdout warnings.

## Contracts

- JWT: HS256, required `exp`, `kid` key rotation (`SigningKeyID`/`VerificationKeys`/`LegacyVerificationKey`), immutable config, SHA-256 cache keys (`validUntil = min(TTL, exp)`), clone returned claims, on-demand cleanup, pluggable refresh consume store (default in-process).
- signedurl: v2 canonical request; no legacy fallback.
- Captcha: empty expected/input fails; library does not own TTL or attempt limits.
- Crypto: AES-GCM only; KDF params validated before compute.
- User-Agent: heuristic only, not a security control.

## Gates

```bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go mod verify
go mod tidy -diff
bash .github/scripts/check-coverage.sh
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
```

License: MIT (`LICENSE`). Copyright (c) 2025-2026 iwen-conf.
