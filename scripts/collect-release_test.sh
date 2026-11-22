#!/usr/bin/env bash
# Tests scripts/collect-release.sh: every file of a release carries the version
# in its name, and a tag that is not the version of the VERSION file is refused.
set -euo pipefail

cd "$(dirname "$0")/.."
script="$PWD/scripts/collect-release.sh"
version="$(cat VERSION)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }

src="$work/src"
mkdir -p "$src"
for f in tdrop-linux-amd64 tdrop-windows-amd64.exe tdrop-macos-arm64 tdrop-macos-amd64 \
	TomorrowDrop-linux-amd64.deb TomorrowDrop-windows-amd64.exe \
	TomorrowDrop-macos-arm64.dmg TomorrowDrop-macos-amd64.dmg; do
	echo "content of $f" > "$src/$f"
done

# 1. The version goes after the program name, before the system.
out="$work/release"
"$script" "$out" "$src"/* >/dev/null || fail "collecting the files must succeed"
for want in \
	"tdrop-$version-linux-amd64" "tdrop-$version-windows-amd64.exe" \
	"tdrop-$version-macos-arm64" "tdrop-$version-macos-amd64" \
	"TomorrowDrop-$version-linux-amd64.deb" "TomorrowDrop-$version-windows-amd64.exe" \
	"TomorrowDrop-$version-macos-arm64.dmg" "TomorrowDrop-$version-macos-amd64.dmg"; do
	[ -f "$out/$want" ] || fail "missing $want; the release has: $(ls "$out")"
done
[ "$(ls "$out" | wc -l | tr -d ' ')" = 8 ] || fail "the release must have exactly the 8 files; has: $(ls "$out")"
[ "$(cat "$out/tdrop-$version-macos-arm64")" = "content of tdrop-macos-arm64" ] || fail "the content of a file must not change"
[ -f "$src/tdrop-macos-arm64" ] || fail "the built file must stay where it was"

# 2. A tag that is not v<VERSION> is refused, and nothing is written.
if RELEASE_TAG="v0.0.1" "$script" "$work/wrong" "$src"/* >/dev/null 2>&1; then
	fail "a tag that is not v$version must be refused"
fi
[ ! -e "$work/wrong" ] || [ -z "$(ls -A "$work/wrong")" ] || fail "nothing is written for a refused tag"
RELEASE_TAG="v$version" "$script" "$work/right" "$src"/* >/dev/null || fail "the tag v$version must be accepted"

# 3. Missing files, and names with no system part, are errors.
if "$script" "$work/missing" "$src/does-not-exist" >/dev/null 2>&1; then
	fail "a missing file must be an error"
fi
echo x > "$src/plain"
if "$script" "$work/plain" "$src/plain" >/dev/null 2>&1; then
	fail "a name that cannot take a version must be an error"
fi

echo "collect-release: ok"
