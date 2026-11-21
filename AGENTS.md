# AGENTS.md — Tomorrow Drop

## Structure

- Keep a **Git monorepo with a single Go module** (`go.mod` at the root).
- `cmd/tdrop/`: CLI.
- `cmd/tomorrow-drop/`: Fyne graphical application.
- `internal/`: shared core.
- `assets/`: visual resources.
- `docs/`: PRD and technical documentation.
- `scripts/`: build and distribution automation.
- Isolate platform-specific integrations behind interfaces and *build tags*.
- **Never import Fyne in the shared core.**

## Development

- Follow the PRD and execute `TOMORROW_DROP_TASKS.md` in the defined order.
- **TDD is mandatory:** write the test first, confirm it fails, implement and refactor.
- Prefer the Go standard library and minimize dependencies.
- Never implement custom cryptography or weaken security checks.
- Keep changes small, cohesive and testable.

## Platforms (mandatory)

Every change to user-visible behavior, protocol, discovery, pairing, trust or transfer must be considered for **every** platform, and applied to each one it affects in the same block:

- **CLI:** `cmd/tdrop`.
- **Desktop graphical app:** `cmd/tomorrow-drop` (Fyne), on Linux, Windows and macOS. macOS is built as two bundles, arm64 and amd64 (`scripts/package-macos.sh`).
- **Android:** the Fyne application, built with `scripts/install-android.sh`.
- **iPhone and iPad:** the native SwiftUI app in `ios/` over the core `cmd/tomorrow-drop-ios`. It does not use the Fyne screens, so a fix in `cmd/tomorrow-drop` does not reach it.
- **Shared core:** `internal/`. A change here reaches every platform that calls it. Check each caller, including `cmd/tomorrow-drop-ios`.

Rules:

- Before closing a change, list the platforms it touches and the ones it does not, with the reason.
- A platform that is not changed yet must not be left silently behind: add a task or a note in `TOMORROW_DROP_TASKS.md` and in the platform's doc (`docs/MOBILE.md`, `docs/GUI.md`, `docs/PACKAGING.md`), and say so in the final message.
- Each platform gets its own regression test where the code can be tested (for example, `cmd/tomorrow-drop-ios` has its own test package). A test on one platform does not cover another.
- Platform builds that cannot run here (physical iPhone/iPad, Intel Mac, Android device) are reported as built or not built, never as verified on hardware.

## Internationalization (i18n)

- The software is multilingual. Initial language: **Brazilian Portuguese (`pt-BR`)**. Mandatory fallback: **English (`en`)**.
- By default, the language is detected from the operating system (the user's locale). Unsupported languages fall back to `en`.
- The user can choose the language in the application settings, overriding automatic detection. The choice is persisted, and the "auto" option returns to system detection.
- Every user-visible text (GUI, CLI, displayed error messages) comes from translation files, never from literals scattered through the code.
- Translation files are versioned with the code, with the same keys in every language. A key missing from the chosen language uses the `en` text.
- Detection, selection and fallback logic lives in the core and does not depend on Fyne. Only the presentation layer consumes translations.
- Required tests: system detection, settings override, fallback to `en` and key parity across translation files.

## Test rules (non-negotiable)

- **Never skip, disable, remove or weaken a test to make the suite pass.**
- Forbidden: `skip`, `xfail`, exclusions, artificial mocks or changes to assertions to hide failures.
- If a test fails, investigate and fix the root cause.
- **Every bug or error identified requires a regression test before the fix.**
- The regression test must reproduce the problem and fail before the fix.
- After fixing, run the regression test and the full suite.
- Keep regression tests permanently to prevent the bug from returning.
- Never declare success with failing tests or unexecuted checks.

## Mandatory validation

At the end of **each block of tasks**, run:

```bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
ruff check .
lizard .
semgrep scan --config auto .
trivy fs .
```

- Fix the problems found before moving on.
- Ruff applies only when Python code exists; otherwise record it as not applicable.
- Investigate and document false positives, without disabling checks indiscriminately.
- Do not mark tasks as completed while relevant failures remain.
- Change `[   ]` to `[x]` only after implementation and validation.

## Commits

Commits follow [Conventional Commits](https://www.conventionalcommits.org/). Commit messages are always written in English, including the subject, body and footers, even though the user communicates in Portuguese.

- Format: `type(scope): subject`. Scope is optional and names the affected package or area (`identity`, `trust`, `transport`, `i18n`, `cli`, `docs`, `ci`).
- Allowed types: `feat`, `fix`, `refactor`, `test`, `docs`, `build`, `ci`, `chore`.
- Subject: imperative mood, lowercase after the type, no trailing period, 72 characters or fewer.
- Body (when needed): wrapped at 72 characters, explaining *why* the change was made. Reference the task when there is one, for example `Refs: task 013`.
- Breaking changes: add a `BREAKING CHANGE:` footer describing the impact.
- One logical change per commit. Do not mix refactors, features and formatting in the same commit.
- A `fix` commit includes its regression test in the same commit (see "Test rules").
- Commit only after `scripts/check.sh` passes for the affected block, and only when the user asks for a commit.

Examples:

```
feat(identity): generate and persist Ed25519 identity

Keys use crypto/rand and are stored as PKCS#8 PEM with 0600 permissions.
Refs: task 006
```

```
fix(storage): remove partial files when a write fails

The regression test reproduces the partial file before the fix and stays in
the suite afterwards.
```

## Completion criteria

A block is complete only when:

- The implementation meets the PRD.
- TDD and regression tests pass.
- The full suite is green.
- Quality and security tools have been run and their results addressed.
- The required documentation is up to date.

**Final rule: fix the code, never game the tests.**
