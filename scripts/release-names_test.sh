#!/usr/bin/env bash
# Tests what a release contains: the macOS files are named "macos" (never the
# Go name "darwin"), the command-line programs for both Mac architectures are
# built, and the release workflow builds the macOS graphical application on a
# macOS runner and publishes it.
set -euo pipefail

cd "$(dirname "$0")/.."

fail() { echo "FAIL: $*" >&2; exit 1; }

workflow=.github/workflows/release.yml

cross="$(make -n cross)"
for arch in arm64 amd64; do
	grep -q -- "-o bin/tdrop-macos-$arch " <<<"$cross" || fail "make cross must build bin/tdrop-macos-$arch"
done
grep -q -- '-o [^ ]*tdrop-darwin' <<<"$cross" && fail "make cross must not name a file after darwin"

grep -q 'tdrop-darwin' "$workflow" && fail "$workflow must not name a file after darwin"
for arch in arm64 amd64; do
	grep -q "tdrop-macos-$arch" "$workflow" || fail "$workflow must publish tdrop-macos-$arch"
	grep -q "TomorrowDrop-macos-$arch.dmg" "$workflow" || fail "$workflow must publish the macOS disk image for $arch"
done
grep -q 'runs-on: macos' "$workflow" || fail "$workflow must build the macOS application on a macOS runner"
grep -q 'make package-macos' "$workflow" || fail "$workflow must run make package-macos"
# The release is checksummed and published once, after every job has finished.
grep -q 'needs: \[package, macos\]' "$workflow" || fail "$workflow must publish after the package and macos jobs"
# The signing material comes from secrets and is never written in the workflow.
for secret in MACOS_CERTIFICATE MACOS_CERTIFICATE_PASSWORD MACOS_NOTARY_KEY MACOS_NOTARY_KEY_ID MACOS_NOTARY_ISSUER; do
	grep -q "secrets.$secret" "$workflow" || fail "$workflow must read $secret from the repository secrets"
done
grep -q 'MACOS_SIGN_IDENTITY' "$workflow" || fail "$workflow must pass the signing identity to make package-macos"

# Names shown to people in the current docs follow the same rule. The notes of
# a published version (docs/releases/v1.0.0.md) keep the names of its files.
if grep -rn 'tdrop-darwin' README.md docs/*.md Makefile scripts .github 2>/dev/null | grep -v 'release-names_test.sh'; then
	fail "the files are named tdrop-macos-<arch>, not tdrop-darwin-<arch>"
fi

echo "release names: ok"
