# Qualidade e segurança

## Execução

Todas as verificações exigidas por `AGENTS.md` são executadas por:

```sh
scripts/check.sh
```

O script falha se alguma ferramenta estiver ausente. Verificações não são puladas silenciosamente.

## Ferramentas

| Ferramenta | Instalação | Observação |
|---|---|---|
| Go 1.24+ | go.dev | Obrigatória |
| golangci-lint 2.15.0 | `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0` | Configuração em `.golangci.yml` |
| lizard 1.26.1 | `pip install lizard==1.26.1` | |
| semgrep 1.181.0 | `pip install semgrep==1.181.0` | `--config auto` consulta o registro de regras e exige rede |
| trivy 0.76.0 | Script oficial de instalação ou gerenciador de pacotes | Verifica vulnerabilidades e segredos no repositório |
| ruff | `pip install ruff` | Exigida apenas se houver código Python |

## Pipeline

`.github/workflows/ci.yml` executa:

- Job `test`, em Linux, Windows e macOS: gofmt, go vet, go test e go test -race.
- Job `quality`, em Linux: `scripts/check.sh` com todas as ferramentas instaladas.

## Falsos positivos

Falsos positivos devem ser investigados e documentados aqui, com a justificativa, antes de qualquer exclusão. Não desative verificações de forma ampla para contornar um alerta.

Registro:

| Ferramenta | Local | Motivo do falso positivo |
|---|---|---|
| gosec G115 | `internal/protocol/frame.go` (`WriteFrame`) | Conversão `int` → `uint32` do tamanho do payload. O tamanho é validado contra `limits.MaxControlFrameBytes` antes da conversão, então não há estouro possível. Supressão local com justificativa no próprio código. |
