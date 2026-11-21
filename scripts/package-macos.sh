#!/usr/bin/env bash
# Builds the unsigned macOS application bundle into dist/.
# Requires the Fyne packaging tool: go install fyne.io/tools/cmd/fyne@v1.7.4
# Signing and notarization are not done here (see docs/PACKAGING.md).
set -euo pipefail

cd "$(dirname "$0")/.."

if ! command -v fyne >/dev/null 2>&1; then
	echo "missing tool: fyne (go install fyne.io/tools/cmd/fyne@v1.7.4)" >&2
	exit 1
fi

mkdir -p dist
fyne package \
	--target darwin \
	--src "$PWD/cmd/tomorrow-drop" \
	--icon "$PWD/assets/icon.png" \
	--name "Tomorrow Drop" \
	--app-id org.tomorrowdrop.app \
	--app-version 0.1.0 \
	--app-build 1 \
	--release
mv "Tomorrow Drop.app" dist/ 2>/dev/null || true
echo "Built dist/Tomorrow Drop.app (unsigned)."
