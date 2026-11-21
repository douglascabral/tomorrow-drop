#!/usr/bin/env bash
# Builds the native Tomorrow Drop app for iPhone and iPad and installs it on the
# connected device (make install-ios).
#
#   1. the Go core is built as a static library for the iPhone SDK (build/ios);
#   2. the Xcode project is generated from ios/project.yml (XcodeGen);
#   3. the app is built and signed with the team of the Apple Development
#      certificate (automatic signing creates the provisioning profile);
#   4. the app is copied to dist/ and installed on the device.
#
# Settings, from the environment when present:
#   IOS_UDID   the device, when more than one is connected
#   IOS_CERT   the signing certificate (default: the first Apple Development one)
set -euo pipefail

cd "$(dirname "$0")/.."
root="$PWD"

fail() {
	echo "install-ios: $*" >&2
	exit 1
}

command -v xcodegen >/dev/null 2>&1 || fail "xcodegen not found: brew install xcodegen"

# The device: a physical iPhone or iPad paired with this Mac.
if [ -n "${IOS_UDID:-}" ]; then
	udid="$IOS_UDID"
else
	devices="$(xcrun devicectl list devices 2>/dev/null | grep -i ' physical' | grep -oE '[0-9A-F]{8}-[0-9A-F]{16}' | sort -u || true)"
	[ -n "$devices" ] || fail "no iPhone or iPad connected. Connect it by USB, unlock it, and trust this Mac"
	[ "$(printf '%s\n' "$devices" | wc -l | tr -d ' ')" -eq 1 ] || fail "more than one iPhone or iPad connected. Set IOS_UDID to one of: $(echo $devices)"
	udid="$devices"
fi

cert="${IOS_CERT:-}"
if [ -z "$cert" ]; then
	cert="$(security find-identity -v -p codesigning | grep 'Apple Development' | head -1 | sed 's/.*"\(.*\)".*/\1/' || true)"
fi
[ -n "$cert" ] || fail "no Apple Development certificate in the keychain. Sign in to Xcode with your Apple account"

# The team is the organizational unit of the certificate.
team="$(security find-certificate -c "$cert" -p | openssl x509 -noout -subject | sed -n 's/.*OU *= *\([A-Z0-9]*\).*/\1/p' | head -1)"
[ -n "$team" ] || fail "cannot read the team of the certificate"

version="$(sed -n 's/^ *Version = "\(.*\)"/\1/p' FyneApp.toml)"
build="$(sed -n 's/^ *Build = \([0-9][0-9]*\)/\1/p' FyneApp.toml)"

echo "==> building the Go core for the iPhone SDK"
mkdir -p build/ios
sdk="$(xcrun --sdk iphoneos --show-sdk-path)"
cc="$(xcrun --sdk iphoneos -f clang) -isysroot $sdk -arch arm64 -miphoneos-version-min=16.0"
GOOS=ios GOARCH=arm64 CGO_ENABLED=1 CC="$cc" go build -buildmode=c-archive -o build/ios/libtdrop.a ./cmd/tomorrow-drop-ios

echo "==> generating the Xcode project"
(cd ios && xcodegen generate >/dev/null)

echo "==> building $version ($build) for $udid, team $team"
derived="$root/build/ios/derived"
xcodebuild -project ios/TomorrowDrop.xcodeproj -scheme TomorrowDrop -configuration Debug \
	-destination "id=$udid" -derivedDataPath "$derived" -allowProvisioningUpdates \
	DEVELOPMENT_TEAM="$team" CODE_SIGN_STYLE=Automatic \
	MARKETING_VERSION="$version" CURRENT_PROJECT_VERSION="$build" build >build/ios/xcodebuild.log 2>&1 \
	|| { grep -E "error:" build/ios/xcodebuild.log | head -5 >&2; fail "build failed, see build/ios/xcodebuild.log"; }

mkdir -p dist
rm -rf dist/TomorrowDrop-ios-arm64.app
cp -R "$derived/Build/Products/Debug-iphoneos/TomorrowDrop.app" dist/TomorrowDrop-ios-arm64.app

echo "==> installing on $udid"
xcrun devicectl device install app --device "$udid" dist/TomorrowDrop-ios-arm64.app
echo "installed: dist/TomorrowDrop-ios-arm64.app"
echo "Unlock the device and open Tomorrow Drop. The first time, trust the developer in Settings > General > VPN & Device Management."
