#!/usr/bin/env bash
# Builds the unsigned macOS application bundle into dist/ (make package-macos).
# Uses the project's Fyne packaging tool (build/tools/fyne-ios15), built by
# scripts/fyne-tools/build.sh when it is missing. Signing and notarization are
# not done here (see docs/PACKAGING.md).
set -euo pipefail

cd "$(dirname "$0")/.."
root="$PWD"

tool="$root/build/tools/fyne-ios15"
if [ ! -x "$tool" ]; then
	scripts/fyne-tools/build.sh
fi

rm -rf "Tomorrow Drop.app"
"$tool" package \
	--target darwin \
	--src "$root/cmd/tomorrow-drop" \
	--icon "$root/assets/icon.png" \
	--name "Tomorrow Drop" \
	--app-id org.tomorrowdrop.app \
	--app-version 0.1.0 \
	--app-build 1 \
	--release

mkdir -p dist
rm -rf "dist/Tomorrow Drop.app"
mv "Tomorrow Drop.app" dist/
app="dist/Tomorrow Drop.app"

# macOS asks for Local Network access only when the bundle declares why it needs
# it; without the key, discovery is blocked without any prompt (docs/MOBILE.md).
/usr/libexec/PlistBuddy -c "Add :NSLocalNetworkUsageDescription string Tomorrow Drop finds and transfers files with other devices on your local network." "$app/Contents/Info.plist"

# The bundle changed, so it is signed again (ad hoc), with the application
# identifier: macOS keys the permissions of an app by that identifier.
codesign --force --sign - --identifier org.tomorrowdrop.app "$app"
echo "Built $app (unsigned: ad-hoc signature only)."
