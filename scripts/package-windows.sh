#!/usr/bin/env bash
# Builds the Windows graphical application as dist/TomorrowDrop-windows-amd64.exe
# (make package-windows), with the Tomorrow Drop icon embedded in the file. The
# program is cross-compiled inside the fyne-cross Linux image with the MinGW
# toolchain, so Docker must be running.
set -euo pipefail

cd "$(dirname "$0")/.."
root="$PWD"
image="fyneio/fyne-cross-images:linux"
stage="$root/build/windows"

fail() {
	echo "package-windows: $*" >&2
	exit 1
}

command -v docker >/dev/null 2>&1 || fail "docker not found (on macOS, start Colima or Docker Desktop)"
docker info >/dev/null 2>&1 || fail "the Docker daemon is not running"

rm -rf "$stage"
mkdir -p "$stage/out" dist

# Windows icons hold PNG images of up to 256 pixels. assets/icon-256.png is the
# project icon at that size, so the executable shows the same icon as the other
# platforms, on any build machine.
python3 - "$root/assets/icon-256.png" "$stage/icon.ico" <<'PY'
import struct, sys
png = open(sys.argv[1], "rb").read()
header = struct.pack("<HHH", 0, 1, 1)
entry = struct.pack("<BBBBHHII", 0, 0, 0, 0, 1, 32, len(png), 6 + 16)
open(sys.argv[2], "wb").write(header + entry + png)
PY

docker run --rm \
	-v "$root":/src:ro \
	-v "$stage":/stage \
	-v tomorrow-drop-gomod:/go/pkg/mod \
	-v tomorrow-drop-gobuild:/root/.cache/go-build \
	-v "$stage/out":/out \
	-e CGO_ENABLED=1 -e GOOS=windows -e GOARCH=amd64 -e CC=x86_64-w64-mingw32-gcc \
	"$image" /bin/sh -c '
		set -e
		apt-get update -qq
		apt-get install -y -qq gcc-mingw-w64-x86-64 >/dev/null
		cp -r /src /work
		cd /work
		# The icon resource is compiled into the package, next to its sources.
		# The tool runs on this Linux image, so it is built for the image, not for the target.
		GOOS= GOARCH= CGO_ENABLED=0 go install github.com/akavel/rsrc@v0.10.2
		"$(go env GOPATH)/bin/rsrc" -ico /stage/icon.ico -arch amd64 -o cmd/tomorrow-drop/rsrc_windows_amd64.syso
		go build -trimpath -ldflags "-s -w -H=windowsgui" -o /out/TomorrowDrop-windows-amd64.exe ./cmd/tomorrow-drop
	' || fail "the Windows build failed"

mv "$stage/out/TomorrowDrop-windows-amd64.exe" dist/TomorrowDrop-windows-amd64.exe
echo "built dist/TomorrowDrop-windows-amd64.exe"
