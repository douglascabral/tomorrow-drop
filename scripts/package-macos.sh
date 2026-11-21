#!/usr/bin/env bash
# Builds the unsigned macOS application bundles into dist/ (make package-macos):
# one for Apple silicon (arm64) and one for Intel (amd64), by default. Set
# MACOS_ARCHES to build a subset, for example MACOS_ARCHES=arm64.
# Uses the project's Fyne packaging tool (build/tools/fyne-ios15), built by
# scripts/fyne-tools/build.sh when it is missing. Signing and notarization are
# not done here (see docs/PACKAGING.md).
set -euo pipefail

cd "$(dirname "$0")/.."
root="$PWD"
version="$(cat "$root/VERSION")"
arches="${MACOS_ARCHES:-arm64 amd64}"

tool="$root/build/tools/fyne-ios15"
if [ ! -x "$tool" ]; then
	scripts/fyne-tools/build.sh
fi

mkdir -p dist
for arch in $arches; do
	# lipo names the Intel architecture x86_64; Go calls it amd64.
	case "$arch" in
	arm64) want=arm64 ;;
	amd64) want=x86_64 ;;
	*) echo "unsupported macOS architecture: $arch" >&2; exit 2 ;;
	esac

	work="$root/build/macos/$arch"
	rm -rf "$work"
	mkdir -p "$work"

	# The graphical binary needs cgo, so it is compiled with the Apple toolchain
	# for the target architecture.
	GOOS=darwin GOARCH="$arch" CGO_ENABLED=1 go build -trimpath \
		-ldflags "-X main.version=$version" -o "$work/tomorrow-drop" ./cmd/tomorrow-drop
	if [ "$(lipo -archs "$work/tomorrow-drop")" != "$want" ]; then
		echo "the binary for $arch was not built for $arch" >&2
		exit 1
	fi

	(
		cd "$work"
		"$tool" package \
			--target darwin \
			--src "$root/cmd/tomorrow-drop" \
			--executable "$work/tomorrow-drop" \
			--icon "$root/assets/icon.png" \
			--name "Tomorrow Drop" \
			--app-id org.tomorrowdrop.app \
			--app-version "$version" \
			--app-build 1 \
			--release
	)

	app="dist/TomorrowDrop-macos-$arch.app"
	rm -rf "$app"
	mv "$work/Tomorrow Drop.app" "$app"

	# macOS asks for Local Network access only when the bundle declares why it
	# needs it; without the key, discovery is blocked without any prompt
	# (docs/MOBILE.md).
	/usr/libexec/PlistBuddy -c "Add :NSLocalNetworkUsageDescription string Tomorrow Drop finds and transfers files with other devices on your local network." "$app/Contents/Info.plist"

	# The bundle changed, so it is signed again (ad hoc), with the application
	# identifier: macOS keys the permissions of an app by that identifier.
	codesign --force --sign - --identifier org.tomorrowdrop.app "$app"
	echo "Built $app ($arch, unsigned: ad-hoc signature only)."
done
