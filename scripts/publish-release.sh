#!/usr/bin/env bash
# Publishes the files of a version tag as a GitHub release.
# Usage: scripts/publish-release.sh <tag> <directory-with-the-files>
#
# The release text is docs/releases/<tag>.md when it exists; otherwise GitHub
# generates it from the commits since the previous tag. An existing release
# (for example, a tag whose workflow runs again) is updated in place: its
# files are replaced and its title and notes are set again.
set -euo pipefail

if [ "$#" -ne 2 ]; then
	echo "usage: $0 <tag> <directory-with-the-files>" >&2
	exit 2
fi

tag="$1"
dir="$(cd "$2" && pwd)"

cd "$(dirname "$0")/.."

notes="docs/releases/${tag}.md"
title="Tomorrow Drop ${tag}"

if ! gh release view "$tag" >/dev/null 2>&1; then
	if [ -f "$notes" ]; then
		gh release create "$tag" "$dir"/* --title "$title" --notes-file "$notes"
	else
		gh release create "$tag" "$dir"/* --title "$title" --generate-notes
	fi
	exit 0
fi

echo "release ${tag} already exists: replacing its files and notes" >&2
gh release upload "$tag" "$dir"/* --clobber
if [ -f "$notes" ]; then
	gh release edit "$tag" --title "$title" --notes-file "$notes"
else
	gh release edit "$tag" --title "$title"
fi
