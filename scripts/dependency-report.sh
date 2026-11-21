#!/usr/bin/env bash
# Lists every module in the build graph with its version and the license it
# declares in its LICENSE file (task 067). Heuristic: unknown licenses are
# reported as UNKNOWN and must be reviewed by hand before release.
set -euo pipefail

cd "$(dirname "$0")/.."

go list -m -f '{{.Path}} {{.Version}} {{.Dir}}' all | while read -r path version dir; do
	[ "$path" = "tomorrow-drop" ] && continue
	license="UNKNOWN"
	if [ -n "$dir" ] && [ -d "$dir" ]; then
		file="$(find "$dir" -maxdepth 1 \( -iname 'LICENSE*' -o -iname 'COPYING*' \) | head -1)"
		if [ -n "$file" ]; then
			text="$(cat "$file")"
			case "$text" in
			*"Apache License"*"Version 2.0"*) license="Apache-2.0" ;;
			*"MIT License"*|*"Permission is hereby granted, free of charge"*) license="MIT" ;;
			*"BSD 3-Clause"*|*"Redistribution and use in source and binary forms"*"Neither the name"*) license="BSD-3-Clause" ;;
			*"BSD 2-Clause"*) license="BSD-2-Clause" ;;
			*"ISC License"*|*"Permission to use, copy, modify, and/or distribute this software for any"*) license="ISC" ;;
			*"Mozilla Public License Version 2.0"*) license="MPL-2.0" ;;
			*"GNU LESSER GENERAL PUBLIC LICENSE"*) license="LGPL" ;;
			*"GNU GENERAL PUBLIC LICENSE"*) license="GPL" ;;
			*) license="UNKNOWN" ;;
			esac
		fi
	fi
	printf '%s\t%s\t%s\n' "$path" "$version" "$license"
done
