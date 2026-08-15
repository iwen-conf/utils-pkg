# Error API migration (v0.4.0)

Prefer:

- `NewRich` / `WrapRich`
- standard `errors.Is` / `errors.As`
- `IsCode` for legacy string codes
- `GetStatus()` or default RichError JSON (`code`, `msg`)
- `%+v` for internal logs

Deprecated:

- `Error`, `New*`, `Wrap*`, `ErrorType`, `FromType`, `Builder`, `ValidationError`, `Validator`

Legacy APIs remain until a later major version. They now follow standard error chains.
