#!/usr/bin/env bash
# Tests the macOS signing, notarization and disk image scripts. It needs macOS
# (codesign, hdiutil, plutil). The Developer ID paths are exercised with fake
# codesign and xcrun programs that record their arguments, since this project's
# tests have no certificate or Apple account.
set -euo pipefail

cd "$(dirname "$0")/.."
root="$PWD"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }

[ "$(uname -s)" = "Darwin" ] || fail "these tests need macOS (codesign, hdiutil)"

# A minimal application bundle.
app="$work/TomorrowDrop-macos-arm64.app"
mkdir -p "$app/Contents/MacOS"
printf '#!/bin/sh\nexit 0\n' > "$app/Contents/MacOS/fake"
chmod +x "$app/Contents/MacOS/fake"
cat > "$app/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>fake</string>
<key>CFBundleIdentifier</key><string>org.tomorrowdrop.app</string>
<key>CFBundleName</key><string>Tomorrow Drop</string>
<key>CFBundlePackageType</key><string>APPL</string>
</dict></plist>
PLIST

# 1. Without an identity the bundle is signed ad hoc, with the app identifier,
#    and says so.
out="$(env -u MACOS_SIGN_IDENTITY scripts/sign-macos.sh "$app" org.tomorrowdrop.app 2>&1)"
grep -qi 'ad hoc' <<<"$out" || fail "an ad-hoc signature must be reported as such: $out"
codesign --verify --strict "$app" || fail "the bundle does not verify after signing"
details="$(codesign -dv "$app" 2>&1)"
grep -q 'Identifier=org.tomorrowdrop.app' <<<"$details" || fail "the signature must carry the application identifier"

# 2. With an identity the signature uses the hardened runtime and a secure
#    timestamp (both are required for notarization), and is checked after.
mkdir -p "$work/fakebin"
cat > "$work/fakebin/codesign" <<FAKE
#!/usr/bin/env bash
echo "\$*" >> "$work/codesign.calls"
exit 0
FAKE
chmod +x "$work/fakebin/codesign"
rm -f "$work/codesign.calls"
PATH="$work/fakebin:$PATH" MACOS_SIGN_IDENTITY="Developer ID Application: Example (TEAM123)" \
	scripts/sign-macos.sh "$app" org.tomorrowdrop.app >/dev/null
sign="$(grep -- '--sign' "$work/codesign.calls")"
for want in '--options runtime' '--timestamp' 'Developer ID Application: Example (TEAM123)' '--identifier org.tomorrowdrop.app'; do
	grep -q -- "$want" <<<"$sign" || fail "a Developer ID signature needs '$want': $sign"
done
grep -q -- '--verify' "$work/codesign.calls" || fail "the signature must be verified after signing"

# 3. The disk image holds the application under its own name, next to a link
#    to Applications, and the application inside still verifies.
dmg="$work/TomorrowDrop-macos-arm64.dmg"
env -u MACOS_SIGN_IDENTITY scripts/dmg-macos.sh "$app" "$dmg" >/dev/null
[ -f "$dmg" ] || fail "no disk image was written"
hdiutil verify "$dmg" >/dev/null || fail "the disk image is not valid"
mnt="$work/mnt"
mkdir -p "$mnt"
hdiutil attach "$dmg" -nobrowse -readonly -mountpoint "$mnt" >/dev/null
detach() { hdiutil detach "$mnt" -force >/dev/null 2>&1 || true; }
trap 'detach; rm -rf "$work"' EXIT
[ -d "$mnt/Tomorrow Drop.app" ] || fail "the disk image must hold 'Tomorrow Drop.app'; has: $(ls "$mnt")"
[ -L "$mnt/Applications" ] || fail "the disk image must link to /Applications"
codesign --verify --strict "$mnt/Tomorrow Drop.app" || fail "the application inside the disk image does not verify"
detach

# 4. With an identity the disk image is signed too.
rm -f "$work/codesign.calls"
PATH="$work/fakebin:$PATH" MACOS_SIGN_IDENTITY="Developer ID Application: Example (TEAM123)" \
	scripts/dmg-macos.sh "$app" "$work/signed.dmg" >/dev/null
grep -q 'signed.dmg' "$work/codesign.calls" || fail "the disk image must be signed when there is an identity"

# 5. Notarization: only an accepted result is good; the ticket is stapled.
make_xcrun() {
	cat > "$work/fakebin/xcrun" <<FAKE
#!/usr/bin/env bash
echo "\$*" >> "$work/xcrun.calls"
if [ "\$1" = "notarytool" ] && [ "\$2" = "submit" ]; then
	echo '{"id":"abc","status":"$1"}'
fi
exit 0
FAKE
	chmod +x "$work/fakebin/xcrun"
	rm -f "$work/xcrun.calls"
}
notary_env=(MACOS_NOTARY_KEY="$work/key.p8" MACOS_NOTARY_KEY_ID=KEYID MACOS_NOTARY_ISSUER=ISSUER)

make_xcrun Accepted
env PATH="$work/fakebin:$PATH" "${notary_env[@]}" scripts/notarize-macos.sh "$dmg" "$dmg" >/dev/null \
	|| fail "an accepted notarization must succeed"
grep -q 'stapler staple' "$work/xcrun.calls" || fail "the ticket must be stapled after an accepted notarization"
grep -q -- '--wait' "$work/xcrun.calls" || fail "notarization must wait for the result"

make_xcrun Invalid
if env PATH="$work/fakebin:$PATH" "${notary_env[@]}" scripts/notarize-macos.sh "$dmg" "$dmg" >/dev/null 2>&1; then
	fail "a notarization that is not accepted must fail"
fi
grep -q 'stapler staple' "$work/xcrun.calls" && fail "nothing is stapled when notarization fails"

if env -u MACOS_NOTARY_KEY -u MACOS_NOTARY_KEY_ID -u MACOS_NOTARY_ISSUER scripts/notarize-macos.sh "$dmg" "$dmg" >/dev/null 2>&1; then
	fail "notarization without credentials must fail"
fi

# 6. Notarizing needs a Developer ID identity, and the package script says so
#    before it builds anything.
if err="$(env -u MACOS_SIGN_IDENTITY MACOS_NOTARY_KEY_ID=KEYID scripts/package-macos.sh 2>&1)"; then
	fail "notarization with an ad-hoc signature must fail"
fi
grep -qi 'developer id' <<<"$err" || fail "the error must name the Developer ID identity: $err"

echo "package-macos: ok"
