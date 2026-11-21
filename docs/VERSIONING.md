# Versioning

Tomorrow Drop uses [Semantic Versioning 2.0.0](https://semver.org/lang/pt-BR/): every release is `MAJOR.MINOR.PATCH`.

- **MAJOR** changes when a release is not compatible with the previous one: for example, a protocol change that older versions cannot negotiate (see `docs/PROTOCOL.md`), or a trust-store format change that needs manual migration.
- **MINOR** changes when a release adds behavior in a compatible way: a new screen, a new CLI command, a new language.
- **PATCH** changes when a release only fixes defects, without new behavior.

Pre-releases use a suffix such as `1.1.0-rc.1`. Build metadata is not used.

## Where the version lives

| File | Holds | Used by |
|---|---|---|
| `VERSION` | `MAJOR.MINOR.PATCH` | the CLI (`-ldflags` in the `Makefile`), the Linux and macOS packages |
| `FyneApp.toml` | `Version` (the same `MAJOR.MINOR.PATCH`) and `Build` (an integer) | the graphical application, Android and iOS/iPadOS |

`Version` in `FyneApp.toml` must equal `VERSION`. `Build` is not semantic: it is an integer that increases with every build sent to a store, because Android and iOS require it to grow even when the version does not change.

## Releasing

1. Choose the new number from the rules above, and update `VERSION` and `FyneApp.toml`.
2. Write the notes in `docs/releases/v<MAJOR.MINOR.PATCH>.md`.
3. Create and push the tag `v<MAJOR.MINOR.PATCH>`. `scripts/publish-release.sh <tag> <dir>` then publishes the files of that tag as a GitHub release, using `docs/releases/<tag>.md` as its notes.

A version is never reused: once a tag exists, a fix goes into a new PATCH version.
