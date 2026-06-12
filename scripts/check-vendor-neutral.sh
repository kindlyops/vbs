#!/usr/bin/env bash
set -euo pipefail

# Vendor-neutrality guard.
#
# The originating app's publisher is referred to only in generic terms in this
# repository ("the source app", "the publisher's media API"). The publisher's
# two-letter abbreviation, full name, app name, domains, and file extension must
# never appear in files we author.
#
# The forbidden abbreviation is expressed below as a character class so this
# script does not itself contain the literal text; case-insensitive matching
# covers upper/lower case.
#
# Out of scope: third-party code (vendor/), Go dependency manifests
# (go.mod, go.sum) and the Bazel lock file, which legitimately reference an
# unrelated token-handling module whose name happens to contain the same two
# letters, and binary assets.

pattern='j[w]'

matches=$(
	git grep -i -I -nE "$pattern" -- \
		':!vendor/' \
		':!go.mod' \
		':!go.sum' \
		':!MODULE.bazel.lock' \
		':!*.png' ':!*.jpg' ':!*.jpeg' ':!*.ico' ':!*.svg' \
		':!*.woff' ':!*.woff2' ':!*.ttf' || true
)

if [[ -n "$matches" ]]; then
	echo "Vendor-neutrality guard FAILED: forbidden publisher string found in committed files:" >&2
	echo "$matches" >&2
	echo >&2
	echo "Use the generic terms instead (see scripts/check-vendor-neutral.sh)." >&2
	exit 1
fi

echo "Vendor-neutrality guard passed: no forbidden publisher string in authored files."
