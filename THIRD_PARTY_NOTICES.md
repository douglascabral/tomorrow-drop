# Third-party notices

## Runtime dependencies

The core uses only the Go standard library (PRD §3.3). The graphical application adds one direct dependency, required by the PRD:

| Dependency | Version | License | Use |
|---|---|---|---|
| fyne.io/fyne/v2 | v2.8.2 | BSD-3-Clause | Graphical interface (`cmd/tomorrow-drop` only) |

Fyne brings transitive dependencies (for example `github.com/go-gl/gl`, `github.com/go-text/typesetting`, `github.com/fsnotify/fsnotify`, `fyne.io/systray`). They are listed in `go.mod` and `go.sum`. A full per-package license inventory is part of task 067 and must be generated before publication.

Each new dependency must be recorded here with name, version, license and reason before being added to `go.mod`. Dependencies must be compatible with the MIT license.

## Development tools

Tools used only for verification and not distributed with the application:

| Tool | Verified version | License | Use |
|---|---|---|---|
| golangci-lint | 2.15.0 | GPL-3.0 | Lint (`.golangci.yml`) |
| lizard | 1.26.1 | MIT | Cyclomatic complexity |
| semgrep | 1.181.0 | LGPL-2.1 | Static security analysis |
| trivy | 0.76.0 | Apache-2.0 | Vulnerabilities and secrets in the repository |
| ruff | 0.17.1 | MIT | Python lint (not applicable today; there is no Python code) |

These tools are run by `scripts/check.sh` and by CI. They are not part of the binary.
