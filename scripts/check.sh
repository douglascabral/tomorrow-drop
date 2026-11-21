#!/usr/bin/env bash
# Runs the quality and security checks required by AGENTS.md.
# Stops on the first failure and never skips a missing tool.
set -euo pipefail

cd "$(dirname "$0")/.."

for tool in gofmt go golangci-lint lizard semgrep trivy; do
	if ! command -v "$tool" >/dev/null 2>&1; then
		echo "missing tool: $tool" >&2
		exit 1
	fi
done

echo "==> gofmt"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
	echo "unformatted files:" >&2
	echo "$unformatted" >&2
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

echo "==> install-android tests"
scripts/install-android_test.sh

echo "==> publish-release tests"
scripts/publish-release_test.sh

echo "==> ruff"
if [ -n "$(find . -name '*.py' -not -path './.git/*' -print -quit)" ]; then
	command -v ruff >/dev/null 2>&1 || { echo "missing tool: ruff" >&2; exit 1; }
	ruff check .
else
	echo "not applicable: no Python files in the project"
fi

echo "==> lizard"
lizard .

echo "==> semgrep"
semgrep scan --config auto --error .

echo "==> trivy"
trivy fs --exit-code 1 .

echo "All checks passed."
