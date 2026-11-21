#!/usr/bin/env bash
# Builds the Fyne packaging tool with the iOS patch applied.
#
# The official tool hard-codes the iOS deployment target 13.0, which current
# Xcode rejects, and it does not declare NSLocalNetworkUsageDescription, which
# iOS 14+ needs before the app can use the local network. The patch changes
# both. See docs/MOBILE.md.
set -euo pipefail

readonly TOOLS_VERSION="v1.7.4"
root="$(cd "$(dirname "$0")/../.." && pwd)"
out="${1:-$root/build/tools}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

src="$(go mod download -json "fyne.io/tools@$TOOLS_VERSION" | python3 -c 'import json,sys; print(json.load(sys.stdin)["Dir"])')"
cp -R "$src/." "$work/"
chmod -R u+w "$work"

patch -d "$work" -p1 < "$root/scripts/fyne-tools/ios-deployment-and-local-network.patch"

mkdir -p "$out"
(cd "$work" && go build -o "$out/fyne-ios15" ./cmd/fyne)
echo "built $out/fyne-ios15"
