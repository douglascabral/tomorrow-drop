#!/usr/bin/env bash
# Tests scripts/publish-release.sh with a fake gh that records its calls.
# A release that already exists must be updated, not created again.
set -euo pipefail

cd "$(dirname "$0")/.."
script="$PWD/scripts/publish-release.sh"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }

# make_gh writes a fake gh. "view" succeeds only when $1 is "yes".
make_gh() {
	mkdir -p "$work/bin"
	cat > "$work/bin/gh" <<GH
#!/usr/bin/env bash
echo "\$*" >> "$work/calls"
if [ "\$1" = "release" ] && [ "\$2" = "view" ]; then
	exit $([ "$1" = "yes" ] && echo 0 || echo 1)
fi
exit 0
GH
	chmod +x "$work/bin/gh"
	rm -f "$work/calls"
}

# The release files are a directory with two assets.
mkdir -p "$work/release"
echo a > "$work/release/TomorrowDrop-linux-amd64.deb"
echo b > "$work/release/SHA256SUMS"

# A new tag: the release is created with its notes file.
make_gh no
PATH="$work/bin:$PATH" "$script" v9.9.9 "$work/release" >/dev/null 2>&1 \
	|| fail "creating a new release must succeed"
grep -q '^release create v9.9.9 ' "$work/calls" || fail "new tag must call gh release create: $(cat "$work/calls")"
grep -q -- '--generate-notes' "$work/calls" || fail "a tag without docs/releases/<tag>.md must generate its notes"
grep -q 'release upload' "$work/calls" && fail "new release must not upload separately"

# An existing release (v1.0.0 published before a re-run): assets are replaced
# and the notes are updated; gh release create must not run again.
make_gh yes
PATH="$work/bin:$PATH" "$script" v1.0.0 "$work/release" >/dev/null 2>&1 \
	|| fail "an existing release must be updated, not fail"
grep -q '^release create' "$work/calls" && fail "existing release must not call gh release create"
grep -q '^release upload v1.0.0 .*--clobber' "$work/calls" || fail "existing release must upload assets with --clobber: $(cat "$work/calls")"
grep -q '^release edit v1.0.0 .*--notes-file docs/releases/v1.0.0.md' "$work/calls" || fail "existing release must update its notes from docs/releases/<tag>.md"

echo "publish-release tests passed"
