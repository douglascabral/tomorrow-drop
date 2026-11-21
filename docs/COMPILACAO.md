# Compilação

Este documento descreve a compilação inicial do núcleo e da CLI. A compilação da GUI Fyne será documentada quando a tarefa 045 for concluída, pois exige dependências de sistema próprias de cada plataforma.

## Requisitos

- Go 1.24 ou superior (versão mínima declarada em `go.mod`).
- Git, para obter o código.

Nenhuma dependência externa é necessária hoje.

## Linux

```sh
git clone <url-do-repositorio> tomorrow-drop
cd tomorrow-drop
go build -o bin/tdrop ./cmd/tdrop
./bin/tdrop version
go test ./...
```

## macOS

Os comandos são os mesmos do Linux:

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

## Verificações completas

`scripts/check.sh` executa gofmt, golangci-lint, go vet, testes (com e sem `-race`), lizard, semgrep e trivy. Ele exige que todas as ferramentas estejam instaladas e falha se alguma estiver ausente. Ver [QUALIDADE.md](QUALIDADE.md).

## Binários reproduzíveis

Para builds de release, use `CGO_ENABLED=0` quando o binário não depender de cgo:

```sh
CGO_ENABLED=0 go build -trimpath -o bin/tdrop ./cmd/tdrop
```
