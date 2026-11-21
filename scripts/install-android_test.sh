#!/usr/bin/env bash
# Tests the device selection of scripts/install-android.sh with a fake adb.
# The build is never reached in these cases: each one fails before packaging.
set -euo pipefail

cd "$(dirname "$0")/.."
script="$PWD/scripts/install-android.sh"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# fake_adb prints the device list given as its first argument.
make_adb() {
	cat > "$work/adb" <<ADB
#!/usr/bin/env bash
if [ "\$1" = "devices" ]; then
	printf 'List of devices attached\n%b' '$1'
	exit 0
fi
echo "unexpected adb call: \$*" >&2
exit 99
ADB
	chmod +x "$work/adb"
}

fail() { echo "FAIL: $*" >&2; exit 1; }

# No device connected: exit 1 with a message that names the cause.
make_adb ""
out="$(ADB="$work/adb" "$script" 2>&1)" && fail "no device must fail"
case "$out" in *"no Android device"*) ;; *) fail "no-device message missing: $out" ;; esac

# Unauthorized or offline devices do not count as connected.
make_adb 'ABC\tunauthorized\nDEF\toffline\n'
out="$(ADB="$work/adb" "$script" 2>&1)" && fail "unauthorized device must not be used"
case "$out" in *"no Android device"*) ;; *) fail "unauthorized message missing: $out" ;; esac

# Two devices: refuse to guess, unless ANDROID_SERIAL names one.
make_adb 'ABC\tdevice\nDEF\tdevice\n'
out="$(ADB="$work/adb" "$script" 2>&1)" && fail "two devices must fail without ANDROID_SERIAL"
case "$out" in *"ANDROID_SERIAL"*) ;; *) fail "two-device message must name ANDROID_SERIAL: $out" ;; esac

echo "install-android tests passed"
