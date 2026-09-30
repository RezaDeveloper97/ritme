#!/usr/bin/env bash
# Create a canvas-build task file with a valid frontmatter skeleton.
# Usage: roadmap/bin/new.sh ID "Title" TYPE "deps,comma" GROUP "touches,comma" "skills,comma" "boards,comma" "verify command"
#   e.g. roadmap/bin/new.sh CB-MENO-07b "Hot flash widget" frontend "CB-MENO-07" MENO-C "frontend/src/widgets/hot-flash" "new-fsd-slice" "nbl_Meno_HotFlash.dc.html" "cd frontend && npm run test"
# The epic folder is picked from the ID's middle part (CB-<EPIC>-NN). Never overwrites an existing file.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
[ $# -ge 9 ] || { sed -n 2,5p "$0"; exit 1; }
id="$1" title="$2" type="$3" deps="$4" group="$5" touches="$6" skills="$7" boards="$8" verify="$9"
epic=$(echo "$id" | cut -d- -f2 | tr 'A-Z' 'a-z')
dir=$(ls -d "$ROOT"/E*-"$epic" 2>/dev/null | head -1); [ -n "$dir" ] || { echo "no epic folder for $epic" >&2; exit 1; }
slug=$(echo "$title" | tr 'A-Z' 'a-z' | sed -E 's/[^a-z0-9]+/-/g; s/^-|-$//g' | cut -c1-45)
f="$dir/$id-$slug.md"; [ -e "$f" ] && { echo "exists: $f" >&2; exit 1; }
cat > "$f" <<MD
---
id: $id
title: $title
epic: $(echo "$epic" | tr 'a-z' 'A-Z')
type: $type
status: todo
depends_on: [$deps]
parallel_group: $group
touches: [$touches]
skills: [$skills]
boards: [$boards]
verify: $verify
---

# $id — $title

## Why

## Boards

## Scope
1.

## Out of scope

## Acceptance
-
MD
echo "$f"
