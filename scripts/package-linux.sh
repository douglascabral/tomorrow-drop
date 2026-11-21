#!/usr/bin/env bash
# Builds the Linux graphical application as dist/TomorrowDrop-linux-amd64.deb
# (make package-linux). The binary is compiled for amd64 inside the fyne-cross
# Linux image, so Docker must be running; the .deb is then assembled with
# dpkg-deb: the program, its menu entry, its icon and the libraries it needs.
set -euo pipefail

cd "$(dirname "$0")/.."
root="$PWD"
image="fyneio/fyne-cross-images:linux"
version="0.1.0"
stage="$root/build/linux"

fail() {
	echo "package-linux: $*" >&2
	exit 1
}

command -v docker >/dev/null 2>&1 || fail "docker not found (on macOS, start Colima or Docker Desktop)"
command -v dpkg-deb >/dev/null 2>&1 || fail "dpkg-deb not found: brew install dpkg"
docker info >/dev/null 2>&1 || fail "the Docker daemon is not running"

rm -rf "$stage"
mkdir -p "$stage/out" dist

# The sources are mounted read-only and copied, so the build leaves nothing in
# the repository. The module cache is a volume, so later builds are fast.
docker run --rm \
	-v "$root":/src:ro \
	-v tomorrow-drop-gomod:/go/pkg/mod \
	-v "$stage/out":/out \
	-e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=amd64 -e CC=x86_64-linux-gnu-gcc \
	-e PKG_CONFIG_LIBDIR=/usr/lib/x86_64-linux-gnu/pkgconfig:/usr/share/pkgconfig \
	"$image" /bin/sh -c '
		set -e
		dpkg --add-architecture amd64
		apt-get update -qq
		apt-get install -y -qq gcc-x86-64-linux-gnu libgl-dev:amd64 libx11-dev:amd64 \
			libxrandr-dev:amd64 libxcursor-dev:amd64 libxinerama-dev:amd64 \
			libxi-dev:amd64 libxxf86vm-dev:amd64 libxkbcommon-dev:amd64 >/dev/null
		cp -r /src /work
		cd /work
		go build -trimpath -ldflags "-s -w" -o /out/tomorrow-drop ./cmd/tomorrow-drop
	' || fail "the Linux build failed"

pkg="$stage/deb"
mkdir -p "$pkg/DEBIAN" "$pkg/usr/bin" "$pkg/usr/share/applications" "$pkg/usr/share/icons/hicolor/512x512/apps"
install -m 755 "$stage/out/tomorrow-drop" "$pkg/usr/bin/tomorrow-drop"
install -m 644 "$root/assets/icon.png" "$pkg/usr/share/icons/hicolor/512x512/apps/tomorrow-drop.png"

cat > "$pkg/usr/share/applications/tomorrow-drop.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=Tomorrow Drop
Comment=Transfer files and text on the local network
Exec=tomorrow-drop
Icon=tomorrow-drop
Terminal=false
Categories=Network;FileTransfer;
DESKTOP

cat > "$pkg/DEBIAN/control" <<CONTROL
Package: tomorrow-drop
Version: $version
Section: net
Priority: optional
Architecture: amd64
Maintainer: Tomorrow Drop <tomorrow-drop@localhost>
Depends: libc6, libgl1, libx11-6, libxcursor1, libxrandr2, libxinerama1, libxi6, libxxf86vm1, libxkbcommon0
Description: Transfer files and text on the local network
 Tomorrow Drop sends and receives files and text between devices on the same
 network, with no accounts, no cloud and no history kept.
CONTROL

out="dist/TomorrowDrop-linux-amd64.deb"
dpkg-deb --root-owner-group --build "$pkg" "$out" >/dev/null
echo "built $out"
