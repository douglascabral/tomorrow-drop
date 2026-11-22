#!/usr/bin/env bash
# Sends a file to Apple's notary service, waits for the result, and staples the
# ticket to a target. Only an "Accepted" result is a success.
# Usage: scripts/notarize-macos.sh <file-to-submit> <file-to-staple>
#
# The file to submit is a .dmg, or a .zip of the application. The target is the
# .dmg or the .app that gets the ticket, so it opens without a network check.
#
# Credentials are an App Store Connect API key:
#   MACOS_NOTARY_KEY      path of the AuthKey .p8 file
#   MACOS_NOTARY_KEY_ID   the key id
#   MACOS_NOTARY_ISSUER   the issuer id
set -euo pipefail

if [ "$#" -ne 2 ]; then
	echo "usage: $0 <file-to-submit> <file-to-staple>" >&2
	exit 2
fi

: "${MACOS_NOTARY_KEY:?MACOS_NOTARY_KEY (the .p8 key file) is not set}"
: "${MACOS_NOTARY_KEY_ID:?MACOS_NOTARY_KEY_ID is not set}"
: "${MACOS_NOTARY_ISSUER:?MACOS_NOTARY_ISSUER is not set}"

submit="$1"
staple="$2"
auth=(--key "$MACOS_NOTARY_KEY" --key-id "$MACOS_NOTARY_KEY_ID" --issuer "$MACOS_NOTARY_ISSUER")

result="$(xcrun notarytool submit "$submit" "${auth[@]}" --wait --output-format json)"
status="$(plutil -extract status raw -o - - <<<"$result")"
id="$(plutil -extract id raw -o - - <<<"$result" 2>/dev/null || true)"

if [ "$status" != "Accepted" ]; then
	echo "notarization of $submit was not accepted: $status" >&2
	if [ -n "$id" ]; then
		xcrun notarytool log "$id" "${auth[@]}" >&2 || true
	fi
	exit 1
fi

xcrun stapler staple "$staple"
xcrun stapler validate "$staple"
echo "Notarized and stapled $staple"
