#!/usr/bin/env bash
# Signs a macOS application bundle or disk image.
# Usage: scripts/sign-macos.sh <path> [bundle-identifier]
#
# MACOS_SIGN_IDENTITY is the name or hash of a "Developer ID Application"
# certificate in the keychain. Without it the signature is ad hoc ("-"): the
# bundle is intact and runs, but macOS does not know who made it and warns on
# the first open. A Developer ID signature uses the hardened runtime and a
# secure timestamp, which notarization requires (scripts/notarize-macos.sh).
set -euo pipefail

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
	echo "usage: $0 <path> [bundle-identifier]" >&2
	exit 2
fi

target="$1"
identifier="${2:-}"
identity="${MACOS_SIGN_IDENTITY:--}"

args=(--force --sign "$identity")
# macOS keys the permissions of an app (such as Local Network) by its identifier.
if [ -n "$identifier" ]; then
	args+=(--identifier "$identifier")
fi
if [ "$identity" != "-" ]; then
	args+=(--timestamp --options runtime)
fi

codesign "${args[@]}" "$target"
codesign --verify --strict --verbose=2 "$target"

if [ "$identity" = "-" ]; then
	echo "Signed $target ad hoc: not signed by a Developer ID, so macOS warns on the first open."
else
	echo "Signed $target with: $identity"
fi
