# Tomorrow Drop

**Transferiu. Usou. Pronto.** (roughly "Transferred. Used. Done.")

Free, minimalist application to transfer files, texts, links and clipboard content between devices on the same local network. It uses no accounts, cloud, central servers or internet connection.

> **Status:** identity, trust and pairing implemented (blocks 01 and 02). Discovery, transfer and interfaces are not implemented yet. See [TOMORROW_DROP_TASKS.md](TOMORROW_DROP_TASKS.md).

## Documentation

- [PRD v1.9](docs/PRD.md): requirements, product decisions and acceptance criteria.
- [Build](docs/BUILD.md): build for Linux, Windows and macOS.
- [Protocol](docs/PROTOCOL.md): framing, version, states, limits and timeouts.
- [Discovery and connectivity](docs/DISCOVERY.md): UDP datagrams, rate limits, query schedule, device list and manual connection.
- [User guide](docs/USER_GUIDE.md): installation, pairing, sending, receiving, the 30-second rule and the limits.
- [Threat model](docs/THREAT_MODEL.md) and [security review](docs/SECURITY_REVIEW.md).
- [Performance](docs/PERFORMANCE.md), [dependencies](docs/DEPENDENCIES.md) and [acceptance checklist](docs/ACCEPTANCE.md).
- [Packaging](docs/PACKAGING.md): macOS bundle, metadata and what is still missing.
- [Mobile](docs/MOBILE.md): Android and iOS/iPadOS requirements and verification status.
- [Graphical application](docs/GUI.md): screens, availability with the window, safety rules and known limits.
- [Command-line interface](docs/CLI.md): commands, exit codes, authorization rules and headless use.
- [Transfers](docs/TRANSFERS.md): offers, decisions, staging, all-or-nothing publishing and failure handling.
- [Identity and trust](docs/IDENTITY.md): keys, fingerprint, TLS, pairing and message policy.
- [Languages](docs/LANGUAGES.md): pt-BR as the initial language, English as fallback, system detection and user choice.
- [Core standards](docs/STANDARDS.md): errors, logging, cancellation and tests.
- [Quality](docs/QUALITY.md): checks and tools.
- [Third-party notices](THIRD_PARTY_NOTICES.md): dependencies and licenses.

## Structure

```
cmd/tdrop/            CLI
cmd/tomorrow-drop/    graphical application (Fyne, to be integrated in task 045)
internal/             core shared by the GUI and the CLI
assets/               visual resources
docs/                 PRD and technical documentation
scripts/              quality and security check automation
```

Core packages: `identity`, `trust`, `discovery`, `transport`, `protocol`, `transfer`, `storage`, `lifecycle`, `ui`, plus `apperr`, `safelog`, `limits`, `i18n` and `settings` (shared base).

Architecture rule: the core never imports Fyne. Platform-specific integrations stay behind interfaces and build tags.

## Development

Requirements: Go 1.24 or later.

The `Makefile` wraps the common tasks. Run `make help` to list them:

```sh
make cli       # bin/tdrop, without cgo
make test      # go test ./...
make check     # the full validation required by AGENTS.md
```

The plain commands also work:

```sh
go test ./...
go test -race ./...
```

To run all quality and security checks, see [docs/QUALITY.md](docs/QUALITY.md).

## License

[MIT](LICENSE).
