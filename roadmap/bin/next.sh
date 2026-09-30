#!/usr/bin/env bash
# Ritme canvas-build queue (roadmap/). Status lives ONLY in each task file's frontmatter.
# Usage:
#   roadmap/bin/next.sh                       runnable tasks (todo + every depends_on done), in order
#   roadmap/bin/next.sh --all                 every task with status
#   roadmap/bin/next.sh --status ID STATE     todo|in_progress|done|blocked (appends to LOG.md)
#   roadmap/bin/next.sh --show ID             print the task file path
#   roadmap/bin/next.sh --conflicts ID ID...  exit 1 if any two tasks' `touches` overlap
#   roadmap/bin/next.sh --check               validate ids, deps and statuses
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
files() { ls "$ROOT"/E*/CB-*.md 2>/dev/null | sort -V; }
field() { grep -m1 "^$2:" "$1" | sed -E "s/^$2: *//"; }
list() { field "$1" "$2" | sed -E "s/^\[//; s/\]$//" | tr -d " " | tr "," " "; }
BLOOM="$ROOT/../bloom"   # cross-queue deps: B-N* ids resolve in bloom/ (read-only from here)
file_of() { case "$1" in B-*) ls "$BLOOM"/N*/"$1"-*.md 2>/dev/null | head -1;; *) ls "$ROOT"/E*/"$1"-*.md 2>/dev/null | head -1;; esac; }
STATES="todo in_progress done blocked"

# Two paths overlap when one is a prefix of the other on a `/` boundary.
overlaps() { case "$1/" in "$2"/*) return 0;; esac; case "$2/" in "$1"/*) return 0;; esac; return 1; }

case "${1:-}" in
  --show) file_of "$2"; exit;;
  --status)
    case "$2" in B-*) echo "B-* tasks belong to bloom/ — change them with bloom/bin/next.sh" >&2; exit 1;; esac
    f=$(file_of "$2"); [ -n "$f" ] || { echo "no task $2" >&2; exit 1; }
    echo " $STATES " | grep -q " $3 " || { echo "bad state $3 ($STATES)" >&2; exit 1; }
    perl -pi -e "s/^status: .*/status: $3/ if \$. < 20" "$f"
    echo "$(date -u +%Y-%m-%dT%H:%MZ) $2 -> $3" >> "$ROOT/LOG.md"
    echo "$2 -> $3"; exit;;
  --all)
    for f in $(files); do printf "%-12s %-12s %-9s %-10s %s\n" "$(field "$f" id)" "$(field "$f" status)" "$(field "$f" parallel_group)" "$(field "$f" type)" "$(field "$f" title)"; done; exit;;
  --conflicts)
    shift; ids=("$@"); bad=0
    for ((i=0; i<${#ids[@]}; i++)); do for ((j=i+1; j<${#ids[@]}; j++)); do
      a=$(file_of "${ids[$i]}"); b=$(file_of "${ids[$j]}")
      for p in $(list "$a" touches); do for q in $(list "$b" touches); do
        if overlaps "$p" "$q"; then echo "CONFLICT ${ids[$i]} <-> ${ids[$j]}: $p ~ $q"; bad=1; fi
      done; done
    done; done
    [ $bad = 0 ] && echo "no conflicts"; exit $bad;;
  --check)
    ids=" "; for f in $(files) $(ls "$BLOOM"/N*/B-*.md 2>/dev/null); do ids="$ids$(field "$f" id) "; done; bad=0
    for f in $(files); do
      id=$(field "$f" id); st=$(field "$f" status)
      echo " $STATES " | grep -q " $st " || { echo "$id: bad status '$st'"; bad=1; }
      for d in $(list "$f" depends_on); do echo "$ids" | grep -q " $d " || { echo "$id: unknown dep $d"; bad=1; }; done
    done
    [ $bad = 0 ] && echo "ok"; exit $bad;;
esac

STATUS=""; for f in $(files); do STATUS="$STATUS $(field "$f" id)=$(field "$f" status)"; done
stat_of() { case "$1" in B-*) f=$(file_of "$1"); [ -n "$f" ] && field "$f" status;; *) echo "$STATUS" | tr ' ' '\n' | grep -m1 "^$1=" | cut -d= -f2;; esac; }
echo "runnable:"
for f in $(files); do
  id=$(field "$f" id); [ "$(stat_of "$id")" = todo ] || continue
  ok=1; for d in $(list "$f" depends_on); do [ "$(stat_of "$d")" = done ] || ok=0; done
  [ $ok = 1 ] && printf "  %-12s group=%-9s type=%-10s touches=%s\n    %s\n    file: %s\n" "$id" "$(field "$f" parallel_group)" "$(field "$f" type)" "$(field "$f" touches)" "$(field "$f" title)" "roadmap/${f#$ROOT/}"
done
echo; echo "in_progress:"; for f in $(files); do [ "$(field "$f" status)" = in_progress ] && echo "  $(field "$f" id) $(field "$f" title)"; done
echo "blocked:"; for f in $(files); do [ "$(field "$f" status)" = blocked ] && echo "  $(field "$f" id) $(field "$f" title)"; done
echo "waiting on bloom (todo, only B-N* deps missing):"
for f in $(files); do
  id=$(field "$f" id); [ "$(stat_of "$id")" = todo ] || continue
  own=1; miss=""; for d in $(list "$f" depends_on); do [ "$(stat_of "$d")" = done ] && continue; case "$d" in B-*) miss="$miss $d";; *) own=0;; esac; done
  [ $own = 1 ] && [ -n "$miss" ] && echo "  $id ←$miss"
done; true
