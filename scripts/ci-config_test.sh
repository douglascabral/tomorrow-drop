#!/usr/bin/env bash
# Tests that the macOS packaging tests have a place to run. They need macOS
# (codesign, hdiutil), so the Linux "quality" job cannot run them: scripts/check.sh
# runs them only on macOS and reports it otherwise, and the CI workflow runs them
# on the macOS runner of the "test" job. Without this, a check that cannot run is
# either a red Linux job or a test that nobody runs.
set -euo pipefail

cd "$(dirname "$0")/.."

fail() { echo "FAIL: $*" >&2; exit 1; }

check=scripts/check.sh
workflow=.github/workflows/ci.yml

# check.sh: the call is guarded by the operating system and the other case is
# reported, as it is for ruff.
guard_line="$(grep -n 'uname -s' "$check" | head -n1 | cut -d: -f1 || true)"
call_line="$(grep -n 'scripts/package-macos_test.sh' "$check" | head -n1 | cut -d: -f1 || true)"
[ -n "$call_line" ] || fail "$check must run scripts/package-macos_test.sh"
[ -n "$guard_line" ] && [ "$guard_line" -lt "$call_line" ] || fail "$check must run scripts/package-macos_test.sh only on macOS (uname -s)"
grep -q 'not applicable on' "$check" || fail "$check must say that the macOS packaging tests are not applicable on other systems"

# ci.yml: a step runs the macOS packaging tests on the macOS runner only.
step="$(grep -n 'scripts/package-macos_test.sh' "$workflow" | head -n1 | cut -d: -f1 || true)"
[ -n "$step" ] || fail "$workflow must run scripts/package-macos_test.sh"
sed -n "$((step > 4 ? step - 4 : 1)),${step}p" "$workflow" | grep -q "runner.os == 'macOS'" \
	|| fail "$workflow must run scripts/package-macos_test.sh only when runner.os is macOS"

echo "ci config: ok"
