#!/usr/bin/env bash
# Executa as verificações de qualidade e segurança exigidas por AGENTS.md.
# Para em qualquer falha e não pula ferramentas ausentes.
set -euo pipefail

cd "$(dirname "$0")/.."

for ferramenta in gofmt go golangci-lint lizard semgrep trivy; do
	if ! command -v "$ferramenta" >/dev/null 2>&1; then
		echo "ferramenta ausente: $ferramenta" >&2
		exit 1
	fi
done

echo "==> gofmt"
nao_formatados="$(gofmt -l .)"
if [ -n "$nao_formatados" ]; then
	echo "arquivos não formatados:" >&2
	echo "$nao_formatados" >&2
	exit 1
fi

echo "==> golangci-lint"
golangci-lint run ./...

echo "==> go vet"
go vet ./...

echo "==> go test"
go test ./...

echo "==> go test -race"
go test -race ./...

echo "==> ruff"
if [ -n "$(find . -name '*.py' -not -path './.git/*' -print -quit)" ]; then
	command -v ruff >/dev/null 2>&1 || { echo "ferramenta ausente: ruff" >&2; exit 1; }
	ruff check .
else
	echo "não aplicável: nenhum arquivo Python no projeto"
fi

echo "==> lizard"
lizard .

echo "==> semgrep"
semgrep scan --config auto --error .

echo "==> trivy"
trivy fs --exit-code 1 .

echo "Todas as verificações passaram."
