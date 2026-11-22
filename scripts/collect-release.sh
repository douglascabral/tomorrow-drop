#!/usr/bin/env bash
# Copies built files into a release directory, with the version in each name.
# Usage: scripts/collect-release.sh <directory> <file>...
#
# The version is the one in the VERSION file. It goes after the program name,
# before the system: tdrop-linux-amd64 becomes tdrop-1.2.0-linux-amd64, and
# TomorrowDrop-macos-arm64.dmg becomes TomorrowDrop-1.2.0-macos-arm64.dmg. The
# built files are copied, not renamed, so the names used by the other scripts
# and by make do not change.
#
# When RELEASE_TAG is set (the workflow sets it for a tag), it must be
# v<VERSION>: a tag that does not match the VERSION file would publish files
# with the wrong version in their names, so nothing is written.
set -euo pipefail

if [ "$#" -lt 2 ]; then
	echo "usage: $0 <directory> <file>..." >&2
	exit 2
fi

cd_root="$(cd "$(dirname "$0")/.." && pwd)"
version="$(tr -d '[:space:]' < "$cd_root/VERSION")"
dest="$1"
shift

if [ -n "${RELEASE_TAG:-}" ] && [ "$RELEASE_TAG" != "v$version" ]; then
	echo "the tag $RELEASE_TAG is not v$version, the version in the VERSION file" >&2
	exit 1
fi

# Check every file before writing anything.
for file in "$@"; do
	if [ ! -f "$file" ]; then
		echo "no such file: $file" >&2
		exit 1
	fi
	case "$(basename "$file")" in
	*-*) ;;
	*)
		echo "cannot put a version in the name of $file: it has no system part (name-system-arch)" >&2
		exit 1
		;;
	esac
done

mkdir -p "$dest"
for file in "$@"; do
	name="$(basename "$file")"
	versioned="${name%%-*}-$version-${name#*-}"
	cp "$file" "$dest/$versioned"
	echo "$dest/$versioned"
done
