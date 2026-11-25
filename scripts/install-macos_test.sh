#!/usr/bin/env bash
# Tests scripts/install-macos.sh with a fake packaging script: the real build is
# never run. Checks the architecture choice, the copy and the replacement.
set -euo pipefail

cd "$(dirname "$0")/.."
script="$PWD/scripts/install-macos.sh"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }

# The fake packager records the architecture it was asked for and writes a
# bundle with a marker file, in the dist folder of this test.
cat > "$work/package" <<PKG
#!/usr/bin/env bash
echo "\$MACOS_ARCHES" > "$work/arches"
mkdir -p "$work/dist/TomorrowDrop-macos-\$MACOS_ARCHES.app/Contents"
echo new > "$work/dist/TomorrowDrop-macos-\$MACOS_ARCHES.app/Contents/marker"
PKG
chmod +x "$work/package"

run() {
	PACKAGE="$work/package" DIST="$work/dist" INSTALL_DIR="$work/apps" NO_OPEN=1 NO_QUIT=1 "$script" "$@"
}

# Intel and Apple silicon machines build their own architecture.
mkdir -p "$work/apps"
HOST_ARCH=arm64 run >/dev/null
[ "$(cat "$work/arches")" = arm64 ] || fail "arm64 host must build arm64"
HOST_ARCH=x86_64 run >/dev/null
[ "$(cat "$work/arches")" = amd64 ] || fail "x86_64 host must build amd64"

# The application lands in the install folder.
[ "$(cat "$work/apps/Tomorrow Drop.app/Contents/marker")" = new ] || fail "app not installed"

# An older installed version is replaced, not merged.
echo stale > "$work/apps/Tomorrow Drop.app/Contents/old-file"
HOST_ARCH=arm64 run >/dev/null
[ ! -e "$work/apps/Tomorrow Drop.app/Contents/old-file" ] || fail "old version was merged, not replaced"

# An unknown architecture fails with a message that names it.
out="$(HOST_ARCH=ppc run 2>&1)" && fail "unknown architecture must fail"
case "$out" in *ppc*) ;; *) fail "architecture message missing: $out" ;; esac

# A failed build leaves the installed application untouched.
cat > "$work/package" <<'PKG'
#!/usr/bin/env bash
exit 1
PKG
HOST_ARCH=arm64 run >/dev/null 2>&1 && fail "failed build must fail the install"
[ "$(cat "$work/apps/Tomorrow Drop.app/Contents/marker")" = new ] || fail "failed build removed the installed app"

echo "install-macos tests passed"
