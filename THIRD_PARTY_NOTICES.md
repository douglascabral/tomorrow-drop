# Third-party notices

## Runtime dependencies

No external runtime dependencies at the moment. The core uses only the Go standard library (PRD §3.3).

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
