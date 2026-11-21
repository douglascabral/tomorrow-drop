# Languages (i18n)

Rules defined in `AGENTS.md`. Implementation in `internal/i18n` and `internal/settings`.

## Languages

| Code | Status |
|---|---|
| `pt-BR` | Initial language. Also receives Portuguese variants, such as `pt-PT`. |
| `en` | Mandatory fallback. Used when the detected language is not supported and for missing keys. |

To add a language: create `internal/i18n/locales/<code>.json` with exactly the same keys as `en.json`, and add the code to `i18n.Supported`. The test `TestEmbeddedCatalogsHaveSameKeys` fails if the keys diverge.

## Language choice

Precedence, in `i18n.Resolve`:

1. The language saved by the user (`auto`, `pt-BR` or `en`), if it is a supported language.
2. The operating system language (`i18n.DetectSystem`).
3. `en`.

System detection:

- Linux and other Unix systems: the `LANGUAGE`, `LC_ALL`, `LC_MESSAGES` and `LANG` variables, in that order. The first supported value wins.
- macOS: `defaults read -g AppleLocale`, because apps opened from Finder usually do not inherit `LANG`. Without an answer, it falls back to the environment variables.
- Windows: `GetUserDefaultLocaleName` (kernel32).

Limitation: the native macOS and Windows readings are not covered by automated tests in this environment. The tests cover the environment-variable logic, resolution and fallback, and check that the code compiles for all three platforms.

## How the user chooses

- CLI: `tdrop config language` shows the current choice; `tdrop config language <auto|pt-BR|en>` saves it. The file lives at `os.UserConfigDir()/tomorrow-drop/settings.json` and is written atomically.
- GUI: the settings screen does not exist yet (tasks 046 and 059). When it does, it should use `settings.Load` and `settings.Save`.

Corrupted settings file: the application uses `auto` at startup and reports the error when the `config` command runs.

## Texts

Every user-visible text comes from `locales/*.json`. Messages with values use `%s`/`%d` formatting, applied by `Bundle.T`. Keys follow the `area.subject` pattern, for example `cli.uso` and `gui.indisponivel`.

## Error messages

Errors from the core are not shown as raw text. Interfaces call `apperr.MessageKey(err)` to get a translation key (`error.malformed`, `error.timeout`, `error.identity_changed`, and so on) and translate it with `Bundle.T`. Adding a new sentinel requires adding its entry in `apperr` and a `error.*` key in every locale file; the parity test checks the files.
