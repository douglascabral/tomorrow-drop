#!/usr/bin/env bash
# Writes a disk image (.dmg) with the application and a link to Applications.
# Usage: scripts/dmg-macos.sh <Application.app> <output.dmg>
#
# The application keeps the name "Tomorrow Drop.app" inside the image, whatever
# the bundle is called in dist/. The image is signed when MACOS_SIGN_IDENTITY is
# set (scripts/sign-macos.sh); an ad-hoc signature on an image is of no use, so
# without an identity it is left unsigned.
set -euo pipefail

if [ "$#" -ne 2 ]; then
	echo "usage: $0 <Application.app> <output.dmg>" >&2
	exit 2
fi

app="$1"
dmg="$2"
if [ ! -d "$app" ]; then
	echo "no application bundle at $app" >&2
	exit 1
fi

here="$(cd "$(dirname "$0")" && pwd)"
stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT

# ditto keeps the bundle exactly as it was signed (permissions, links, attributes).
ditto "$app" "$stage/Tomorrow Drop.app"
ln -s /Applications "$stage/Applications"

mkdir -p "$(dirname "$dmg")"
rm -f "$dmg"
hdiutil create -volname "Tomorrow Drop" -srcfolder "$stage" -fs HFS+ -format UDZO -ov "$dmg" >/dev/null

if [ -n "${MACOS_SIGN_IDENTITY:-}" ] && [ "$MACOS_SIGN_IDENTITY" != "-" ]; then
	"$here/sign-macos.sh" "$dmg"
fi
echo "Wrote $dmg"
