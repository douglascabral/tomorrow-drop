# Quality and security

## Running the checks

All checks required by `AGENTS.md` are run by:

```sh
scripts/check.sh
```

The script fails if any tool is missing. Checks are never skipped silently.

## Tools

| Tool | Installation | Note |
|---|---|---|
| Go 1.24+ | go.dev | Required |
| golangci-lint 2.15.0 | `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0` | Configured in `.golangci.yml` |
| lizard 1.26.1 | `pip install lizard==1.26.1` | |
| semgrep 1.181.0 | `pip install semgrep==1.181.0` | `--config auto` queries the rule registry and needs network access |
| trivy 0.76.0 | `go install github.com/aquasecurity/trivy/cmd/trivy@v0.76.0` or the official install script | Scans the repository for vulnerabilities and secrets |
| ruff | `pip install ruff` | Required only if Python code exists |

## Pipeline

`.github/workflows/ci.yml` runs:

- Job `test`, on Linux, Windows and macOS: gofmt, go vet, go test and go test -race.
- Job `quality`, on Linux: `scripts/check.sh` with all tools installed.

## False positives

False positives must be investigated and documented here, with the justification, before any exclusion. Do not disable checks broadly to get past an alert.

Record:

| Tool | Location | Reason for the false positive |
|---|---|---|
| gosec G115 | `internal/protocol/frame.go` (`WriteFrame`) | `int` → `uint32` conversion of the payload size. The size is validated against `limits.MaxControlFrameBytes` before the conversion, so no overflow is possible. Local suppression with justification in the code. |
| gosec G304 | `internal/storage/atomic.go`, `storage`, `identity` and `trust` tests | Read/write of a path received from the caller or created in `t.TempDir()`. The storage library does not receive paths from the network. Per-line suppression. |
| gosec G304 | `internal/settings/settings.go` (`Load`) | Reads the configuration file in the user's own directory, with no network input. |
| gosec G703 | `internal/trust/store_test.go` | Same case as above: a path from `t.TempDir()`. |
| gosec G402 and semgrep `bypass-tls-verification` | `internal/transport/tls.go` (`tlsConfig`) | `InsecureSkipVerify` only disables Go's default chain validation, which does not apply to self-signed Ed25519 certificates. The real validation is `verifyPeer` (`VerifyPeerCertificate`), called on every handshake: it requires a valid Ed25519 certificate, a validity period, a signature by the key itself and, when a pinned key is set, equality with it. The semgrep rule was suppressed by its exact id, only on the `tls.Config` literal. |

Real findings fixed (with a regression test) are recorded here:

- TLS session resumption (`SessionTicketsDisabled`): without this option, resumed connections did not go through `VerifyPeerCertificate`, which would bypass the pinned key. Test: `TestConfigDisablesSessionResumption`.
