#!/bin/bash
# Go lint gate for backend-go, called by format.sh after it has formatted a .go file.
#
#   go-lint.sh <absolute path to a .go file under backend-go/>
#
# Runs golangci-lint (backend-go/.golangci.yml) on the file's package and reports only the
# issues in that file. Exit 2 feeds them back to Claude so they get fixed in the same turn
# (same contract as style-gate.sh). It never blocks on:
#   - compile/typecheck errors: the package is often half-written between edits, and
#     `go vet` / `go test` catch those at verify time;
#   - a missing golangci-lint binary (`brew install golangci-lint`, v2).
# Parallel agents share one tree, hence --allow-parallel-runners.

FILE="$1"
ROOT="/Users/rezataheri/PhpstormProjects/ritme"
GO_ROOT="${RITME_GO_ROOT:-$ROOT/backend-go}"  # override only to test the hook on a copy

[ -f "$FILE" ] || exit 0
command -v golangci-lint >/dev/null 2>&1 || exit 0

PKG_DIR="./$(dirname "${FILE#"$GO_ROOT"/}")"

OUT=$(cd "$GO_ROOT" && golangci-lint run --allow-parallel-runners --timeout 45s \
  --show-stats=false --output.text.colors=false --output.text.print-issued-lines=false \
  --path-mode=abs "$PKG_DIR" 2>&1)

# Half-written package: leave it to vet/test.
echo "$OUT" | grep -qE '\(typecheck\)|^# github\.com/' && exit 0

ISSUES=$(echo "$OUT" | grep -F "$FILE:")
if [ -n "$ISSUES" ]; then
  echo "golangci-lint (backend-go/.golangci.yml) — fix these in $FILE:" >&2
  echo "$ISSUES" >&2
  exit 2
fi

exit 0
