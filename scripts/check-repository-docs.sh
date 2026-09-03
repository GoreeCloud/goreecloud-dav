#!/bin/sh
set -eu

required="
README.md
SPECIFICATIONS.md
FEATURES.md
BENEFITS.md
COMPETITIVE-OBJECTIVES.md
BRANDING.md
"

missing=0
for file in $required; do
  if [ ! -s "$file" ]; then
    echo "missing or empty required repository document: $file" >&2
    missing=1
  fi
done

exit "$missing"
