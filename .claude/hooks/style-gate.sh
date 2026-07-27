#!/bin/bash
# PostToolUse hook: enforce the frontend styling rules (CLAUDE.md §10.1, §12)
# on every .ts/.tsx file written under frontend/src.
#
# Checks for static `style={{ … }}` props, hex colour literals, and var(--x)
# references to variables that globals.css never declares.
#
# Exit 2 feeds stderr back to Claude as a correction, so the violation gets
# fixed in the same turn rather than surviving to review.

INPUT=$(cat)
FILE=$(echo "$INPUT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('tool_input',{}).get('file_path',''))" 2>/dev/null)

[ -z "$FILE" ] && exit 0
[ ! -f "$FILE" ] && exit 0

ROOT="/Users/rezataheri/PhpstormProjects/ritme"

case "$FILE" in
  "$ROOT"/frontend/src/*.ts|"$ROOT"/frontend/src/*.tsx) ;;
  *) exit 0 ;;
esac

# Skip test files — they assert on behaviour, not presentation.
case "$FILE" in
  *.test.ts|*.test.tsx|*.spec.ts|*.spec.tsx) exit 0 ;;
esac

OUT=$(cd "$ROOT/frontend" && node scripts/check-styles.mjs "$FILE" 2>&1)
if [ $? -ne 0 ]; then
  echo "$OUT" >&2
  exit 2
fi

exit 0
