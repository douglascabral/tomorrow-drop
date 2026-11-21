#!/usr/bin/env bash
# Builds Tomorrow Drop for Android, keeps the APK in dist/ and installs it on
# the connected phone (make install-android).
#
# Settings are read from the environment when present:
#   ANDROID_SERIAL    the device to use, when more than one is connected
#   ANDROID_HOME      the Android SDK (default: the usual folder of each system)
#   ANDROID_NDK_HOME  the NDK (default: the newest one inside the SDK)
#   ADB               the adb binary (default: the SDK's platform-tools)
set -euo pipefail

cd "$(dirname "$0")/.."
root="$PWD"

fail() {
	echo "install-android: $*" >&2
	exit 1
}

# android_sdk prints the first SDK folder that exists.
android_sdk() {
	local c
	for c in "${ANDROID_HOME:-}" "${ANDROID_SDK_ROOT:-}" "$HOME/Library/Android/sdk" "$HOME/Android/Sdk"; do
		if [ -n "$c" ] && [ -d "$c" ]; then
			echo "$c"
			return 0
		fi
	done
	return 1
}

sdk="$(android_sdk || true)"
adb="${ADB:-}"
if [ -z "$adb" ]; then
	if [ -n "$sdk" ] && [ -x "$sdk/platform-tools/adb" ]; then
		adb="$sdk/platform-tools/adb"
	elif command -v adb >/dev/null 2>&1; then
		adb="$(command -v adb)"
	else
		fail "adb not found: install the Android platform-tools or set ADB"
	fi
fi

# Only devices in the "device" state can receive an install: unauthorized and
# offline devices are listed by adb but are not usable.
connected="$("$adb" devices | awk 'NR > 1 && $2 == "device" {print $1}')"
if [ -z "$connected" ]; then
	fail "no Android device connected. Connect the phone by USB, turn on USB debugging and accept the prompt on the phone"
fi

if [ -n "${ANDROID_SERIAL:-}" ]; then
	if ! printf '%s\n' "$connected" | grep -qx "$ANDROID_SERIAL"; then
		fail "ANDROID_SERIAL=$ANDROID_SERIAL is not a connected device. Connected: $(echo $connected)"
	fi
	serial="$ANDROID_SERIAL"
else
	count="$(printf '%s\n' "$connected" | wc -l | tr -d ' ')"
	if [ "$count" -ne 1 ]; then
		fail "more than one Android device connected. Set ANDROID_SERIAL to one of: $(echo $connected)"
	fi
	serial="$connected"
fi

if [ -z "${ANDROID_NDK_HOME:-}" ] && [ -n "$sdk" ] && [ -d "$sdk/ndk" ]; then
	ANDROID_NDK_HOME="$sdk/ndk/$(ls -1 "$sdk/ndk" | sort -V | tail -1)"
fi

tool="$root/build/tools/fyne-ios15"
if [ ! -x "$tool" ]; then
	scripts/fyne-tools/build.sh
fi

# The version and the application ID come from FyneApp.toml, the single source.
app_id="$(sed -n 's/^ *ID = "\(.*\)"/\1/p' FyneApp.toml)"
version="$(sed -n 's/^ *Version = "\(.*\)"/\1/p' FyneApp.toml)"
build="$(sed -n 's/^ *Build = \([0-9][0-9]*\)/\1/p' FyneApp.toml)"
[ -n "$app_id" ] && [ -n "$version" ] && [ -n "$build" ] || fail "FyneApp.toml lacks ID, Version or Build"

echo "==> packaging $version ($build) for $serial"
(
	cd cmd/tomorrow-drop
	ANDROID_HOME="$sdk" ANDROID_NDK_HOME="${ANDROID_NDK_HOME:-}" "$tool" package \
		-os android -appID "$app_id" -name "Tomorrow Drop" \
		-icon "$root/assets/icon.png" -appVersion "$version" -appBuild "$build"
)

mkdir -p dist
mv cmd/tomorrow-drop/Tomorrow_Drop.apk dist/Tomorrow_Drop.apk

echo "==> installing on $serial"
"$adb" -s "$serial" install -r dist/Tomorrow_Drop.apk
echo "installed: dist/Tomorrow_Drop.apk"
