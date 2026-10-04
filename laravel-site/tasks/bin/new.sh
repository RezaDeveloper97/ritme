#!/usr/bin/env bash
# Create a task file with a valid frontmatter skeleton (run from laravel-site/).
# Usage: tasks/bin/new.sh ID "Title" TYPE "deps,comma" GROUP "touches,comma" "skills,comma" "verify command"
#   e.g. tasks/bin/new.sh L3-12 "Fix X" frontend "L3-01" L3-A "resources/views/pages/x" "" "composer verify"
# TYPE: setup|backend|frontend|fullstack|admin|investigate|quality|release. Never overwrites an existing file.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
[ $# -ge 8 ] || { sed -n 2,5p "$0"; exit 1; }
id="$1" title="$2" type="$3" deps="$4" group="$5" touches="$6" skills="$7" verify="$8"
ms=$(echo "$id" | cut -d- -f1)
slug=$(echo "$title" | tr 'A-Z' 'a-z' | sed -E 's/[^a-z0-9]+/-/g; s/^-|-$//g' | cut -c1-45 | sed -E 's/-+$//')
mkdir -p "$ROOT/$ms"; f="$ROOT/$ms/$id-$slug.md"
[ -e "$f" ] && { echo "exists: $f" >&2; exit 1; }
{
cat <<MD
---
id: $id
title: $title
milestone: $ms
type: $type
status: todo
depends_on: [$deps]
parallel_group: $group
touches: [$touches]
skills: [$skills]
verify: $verify
---

# $id — $title

MD
if [ ! -t 0 ]; then cat; else printf '## Why\n\n## Scope\n- \n\n## Out of scope\n- \n\n## Acceptance\n- \n'; fi
} > "$f"
echo "$f"
