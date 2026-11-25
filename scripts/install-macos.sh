#!/usr/bin/env bash
# Builds the macOS application for this Mac and installs it (make install-macos).
# Only the architecture of this machine is built, so it is faster than
# make package-macos. The running application is closed, the installed one is
# replaced and the new one is opened.
#
# Environment (all optional):
#   INSTALL_DIR   where the application goes (default /Applications)
#   NO_OPEN=1     do not open the application afterwards
#   NO_QUIT=1     do not ask a running application to quit first
# The signature is ad hoc unless MACOS_SIGN_IDENTITY is set (docs/PACKAGING.md).
set -euo pipefail

cd "$(dirname "$0")/.."

host="${HOST_ARCH:-$(uname -m)}"
case "$host" in
arm64) arch=arm64 ;;
x86_64) arch=amd64 ;;
*) echo "unsupported Mac architecture: $host" >&2; exit 2 ;;
esac

package="${PACKAGE:-scripts/package-macos.sh}"
dist="${DIST:-dist}"
dest="${INSTALL_DIR:-/Applications}"
app="$dist/TomorrowDrop-macos-$arch.app"
target="$dest/Tomorrow Drop.app"

# A failed build stops here, before anything installed is touched.
MACOS_ARCHES="$arch" "$package"
[ -d "$app" ] || { echo "the build did not produce $app" >&2; exit 1; }

if [ -z "${NO_QUIT:-}" ]; then
	osascript -e 'tell application id "org.tomorrowdrop.app" to quit' >/dev/null 2>&1 || true
	sleep 1
fi

mkdir -p "$dest"
rm -rf "$target"
ditto "$app" "$target"
echo "Installed $target ($arch)."

if [ -z "${NO_OPEN:-}" ]; then
	open "$target"
fi
