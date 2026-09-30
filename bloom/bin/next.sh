#!/usr/bin/env bash
# Ritme Night & Bloom queue (bloom/). Status lives ONLY in each task file's frontmatter.
# Usage:
#   bloom/bin/next.sh                       runnable tasks (todo + every depends_on done), in order
#   bloom/bin/next.sh --all                 every task with status
#   bloom/bin/next.sh --status ID STATE     todo|in_progress|done|blocked (appends to LOG.md)
#   bloom/bin/next.sh --show ID             print the task file path
#   bloom/bin/next.sh --conflicts ID ID...  exit 1 if any two tasks' `touches` overlap
#   bloom/bin/next.sh --check               validate ids, deps and statuses
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
files() { ls "$ROOT"/N*/B-*.md 2>/dev/null | sort -t- -k2,2V -k3,3V; }
field() { grep -m1 "^$2:" "$1" | sed -E "s/^$2: *//"; }
list() { field "$1" "$2" | sed -E "s/^\[//; s/\]$//" | tr -d " " | tr "," " "; }
file_of() { ls "$ROOT"/N*/"$1"-*.md 2>/dev/null | head -1; }
STATES="todo in_progress done blocked"

# Two paths overlap when one is a prefix of the other on a `/` boundary.
overlaps() { case "$1/" in "$2"/*) return 0;; esac; case "$2/" in "$1"/*) return 0;; esac; return 1; }

case "${1:-}" in
  --show) file_of "$2"; exit;;
  --status)
    f=$(file_of "$2"); [ -n "$f" ] || { echo "no task $2" >&2; exit 1; }
    echo " $STATES " | grep -q " $3 " || { echo "bad state $3 ($STATES)" >&2; exit 1; }
    perl -pi -e "s/^status: .*/status: $3/ if \$. < 20" "$f"
    echo "$(date -u +%Y-%m-%dT%H:%MZ) $2 -> $3" >> "$ROOT/LOG.md"
    echo "$2 -> $3"; exit;;
  --all)
    for f in $(files); do printf "%-9s %-12s %-6s %-10s %s\n" "$(field "$f" id)" "$(field "$f" status)" "$(field "$f" parallel_group)" "$(field "$f" type)" "$(field "$f" title)"; done; exit;;
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
    ids=" "; for f in $(files); do ids="$ids$(field "$f" id) "; done; bad=0
    for f in $(files); do
      id=$(field "$f" id); st=$(field "$f" status)
      echo " $STATES " | grep -q " $st " || { echo "$id: bad status '$st'"; bad=1; }
      for d in $(list "$f" depends_on); do echo "$ids" | grep -q " $d " || { echo "$id: unknown dep $d"; bad=1; }; done
    done
    [ $bad = 0 ] && echo "ok"; exit $bad;;
esac

STATUS=""; for f in $(files); do STATUS="$STATUS $(field "$f" id)=$(field "$f" status)"; done
stat_of() { echo "$STATUS" | tr ' ' '\n' | grep -m1 "^$1=" | cut -d= -f2; }
echo "runnable:"
for f in $(files); do
  id=$(field "$f" id); [ "$(stat_of "$id")" = todo ] || continue
  ok=1; for d in $(list "$f" depends_on); do [ "$(stat_of "$d")" = done ] || ok=0; done
  [ $ok = 1 ] && printf "  %-9s group=%-6s type=%-10s touches=%s\n    %s\n    file: %s\n" "$id" "$(field "$f" parallel_group)" "$(field "$f" type)" "$(field "$f" touches)" "$(field "$f" title)" "bloom/${f#$ROOT/}"
done
echo; echo "in_progress:"; for f in $(files); do [ "$(field "$f" status)" = in_progress ] && echo "  $(field "$f" id) $(field "$f" title)"; done
echo "blocked:"; for f in $(files); do [ "$(field "$f" status)" = blocked ] && echo "  $(field "$f" id) $(field "$f" title)"; done; true
