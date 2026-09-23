#!/bin/bash
# PostToolUse hook: auto-format files after Edit/Write
# Reads the tool input JSON from stdin, extracts file_path, runs the right formatter.

INPUT=$(cat)
FILE=$(echo "$INPUT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('tool_input',{}).get('file_path',''))" 2>/dev/null)

[ -z "$FILE" ] && exit 0
[ ! -f "$FILE" ] && exit 0

ROOT="/Users/rezataheri/PhpstormProjects/ritme"

case "$FILE" in
  "$ROOT"/backend/*.php)
    "$ROOT/backend/vendor/bin/pint" "$FILE" --quiet 2>/dev/null
    ;;
  "$ROOT"/frontend/*.ts|"$ROOT"/frontend/*.tsx|"$ROOT"/frontend/*.js|"$ROOT"/frontend/*.jsx)
    cd "$ROOT/frontend" && npx eslint --fix "$FILE" --no-warn-ignored >/dev/null 2>&1
    ;;
  "$ROOT"/backend-go/*.go)
    # gofmt + goimports (local-prefixes from .golangci.yml); plain gofmt if golangci-lint is missing.
    if command -v golangci-lint >/dev/null 2>&1; then
      (cd "$ROOT/backend-go" && golangci-lint fmt "$FILE" 2>/dev/null) || gofmt -w "$FILE" 2>/dev/null
    else
      gofmt -w "$FILE" 2>/dev/null
    fi
    # Lint gate: exit 2 with the file's golangci-lint issues (see go-lint.sh).
    "$ROOT/.claude/hooks/go-lint.sh" "$FILE"
    exit $?
    ;;
esac

exit 0
