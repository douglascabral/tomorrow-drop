#!/usr/bin/env bash
# Builds the macOS application bundles and disk images into dist/ (make
# package-macos): one for Apple silicon (arm64) and one for Intel (amd64), by
# default. Set MACOS_ARCHES to build a subset, for example MACOS_ARCHES=arm64.
# Each architecture gives dist/TomorrowDrop-macos-<arch>.app and
# dist/TomorrowDrop-macos-<arch>.dmg.
# Uses the project's Fyne packaging tool (build/tools/fyne-ios15), built by
# scripts/fyne-tools/build.sh when it is missing.
#
# Signing and notarization (docs/PACKAGING.md):
#   MACOS_SIGN_IDENTITY    a "Developer ID Application" identity: the bundle and the
#                          image are signed with it (hardened runtime, timestamp).
#                          Without it they are signed ad hoc and macOS warns.
#   MACOS_NOTARY_KEY, MACOS_NOTARY_KEY_ID, MACOS_NOTARY_ISSUER
#                          an App Store Connect API key. When MACOS_NOTARY_KEY_ID is
#                          set the application and the image are notarized and
#                          stapled; this needs MACOS_SIGN_IDENTITY.
set -euo pipefail

cd "$(dirname "$0")/.."
root="$PWD"
version="$(cat "$root/VERSION")"
arches="${MACOS_ARCHES:-arm64 amd64}"

# Check the signing setup before building anything.
notarize=false
if [ -n "${MACOS_NOTARY_KEY_ID:-}" ]; then
	notarize=true
	if [ -z "${MACOS_SIGN_IDENTITY:-}" ] || [ "$MACOS_SIGN_IDENTITY" = "-" ]; then
		echo "notarization needs a Developer ID Application identity: set MACOS_SIGN_IDENTITY" >&2
		exit 1
	fi
fi

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

	# The bundle changed, so it is signed again, with the application identifier:
	# macOS keys the permissions of an app by that identifier.
	scripts/sign-macos.sh "$app" org.tomorrowdrop.app

	if $notarize; then
		# The application is notarized and stapled first, so the copy that people
		# drag out of the image carries its own ticket.
		zip="$work/Tomorrow Drop.zip"
		rm -f "$zip"
		ditto -c -k --keepParent "$app" "$zip"
		scripts/notarize-macos.sh "$zip" "$app"
	fi

	dmg="dist/TomorrowDrop-macos-$arch.dmg"
	scripts/dmg-macos.sh "$app" "$dmg"
	if $notarize; then
		scripts/notarize-macos.sh "$dmg" "$dmg"
	fi
	echo "Built $app and $dmg ($arch)."
done
