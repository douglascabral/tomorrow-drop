# Core standards

Standards shared by all packages in `internal/`. These points were defined in task 004.

## Errors

- Sentinel errors live in `internal/apperr`.
- Wrap with context using `fmt.Errorf("operation: %w", err)`.
- Check with `errors.Is`, never by comparing strings.
- Messages do not include transferred content, keys, clipboard text or full paths.
- `context` errors must go through `apperr.FromContext` before leaving an operation.
- Errors shown to people are translated: call `apperr.MessageKey(err)` and pass the key to the translator (`Bundle.T`). The internal English text is for logs and tests only. Each sentinel has a key `error.*` in every locale file; unknown errors use `error.unknown`.

## Logging

- Use `log/slog` through the `safelog.New` constructor, which automatically redacts sensitive fields.
- Sensitive fields are identified by name: `private`, `secret`, `token`, `password`, `key`, `content`, `clipboard`, `payload`, `filename` and `path`. `[]byte` values are also redacted.
- Non-sensitive field names, such as `device`, `fingerprint` and `context`, are preserved.
- Never log the private key, not even truncated.

## Cancellation

- Every blocking operation takes `context.Context` as its first parameter.
- Network and long I/O operations check the context at each step.
- Closing the application cancels the root context (PRD §17.3).

## Resource limits

- All limits live in `internal/limits`. There are no magic size or count numbers scattered through the code.
- Sizes coming from the network are validated **before** any allocation.

## Tests

- Write the test first and confirm it fails (TDD, AGENTS.md).
- Bugs require a regression test that reproduces the problem before the fix, kept permanently.
- Do not use `skip`, artificial mocks or assertion changes to hide failures.
