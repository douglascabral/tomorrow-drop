# Build

This document describes the initial build of the core and the CLI. The build of the Fyne GUI will be documented when task 045 is complete, because it requires system dependencies specific to each platform.

## Requirements

- Go 1.24 or later (minimum version declared in `go.mod`).
- Git, to obtain the code.

No external dependencies are needed today.

## Linux

```sh
git clone <repository-url> tomorrow-drop
cd tomorrow-drop
go build -o bin/tdrop ./cmd/tdrop
./bin/tdrop version
go test ./...
```

## macOS

The commands are the same as on Linux:

```sh
go build -o bin/tdrop ./cmd/tdrop
./bin/tdrop version
go test ./...
```

## Windows (PowerShell)

```powershell
go build -o bin\tdrop.exe .\cmd\tdrop
.\bin\tdrop.exe version
go test .\...
```

## Full checks

`scripts/check.sh` runs gofmt, golangci-lint, go vet, tests (with and without `-race`), lizard, semgrep and trivy. It requires all tools to be installed and fails if any is missing. See [QUALIDADE.md](QUALIDADE.md).

## Reproducible binaries

For release builds, use `CGO_ENABLED=0` when the binary does not depend on cgo:

```sh
CGO_ENABLED=0 go build -trimpath -o bin/tdrop ./cmd/tdrop
```
