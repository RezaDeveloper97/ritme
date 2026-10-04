#!/usr/bin/env bash
# Ritme Laravel site queue (laravel-site/tasks/). Status lives ONLY in each task file's frontmatter.
# Usage (from laravel-site/):
#   tasks/bin/next.sh                       runnable tasks (todo + every depends_on done), in order
#   tasks/bin/next.sh --all                 every task with status
#   tasks/bin/next.sh --status ID STATE     todo|in_progress|done|blocked (appends to LOG.md)
#   tasks/bin/next.sh --show ID             print the task file path
#   tasks/bin/next.sh --conflicts ID ID...  exit 1 if any two tasks' `touches` overlap
#   tasks/bin/next.sh --check               validate ids, deps and statuses
#   tasks/bin/next.sh --summary             done/total per milestone
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
files() { ls "$ROOT"/L*/L*.md 2>/dev/null | awk -F/ '{print $NF"\t"$0}' | sort -V | cut -f2; }
field() { grep -m1 "^$2:" "$1" | sed -E "s/^$2: *//"; }
list() { field "$1" "$2" | sed -E "s/^\[//; s/\]$//" | tr -d " " | tr "," " "; }
file_of() { ls "$ROOT"/L*/"$1"-*.md 2>/dev/null | head -1; }
STATES="todo in_progress done blocked"
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
    for f in $(files); do printf "%-8s %-12s %-6s %-11s %s\n" "$(field "$f" id)" "$(field "$f" status)" "$(field "$f" parallel_group)" "$(field "$f" type)" "$(field "$f" title)"; done; exit;;
  --summary)
    for ms in $(ls -d "$ROOT"/L*/ | xargs -n1 basename | sort -V); do
      t=$(ls "$ROOT/$ms"/L*.md | wc -l | tr -d ' '); d=$(grep -l '^status: done' "$ROOT/$ms"/L*.md 2>/dev/null | wc -l || true | tr -d ' ')
      printf "%-4s %3s/%-3s done\n" "$ms" "$d" "$t"; done; exit;;
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
  [ $ok = 1 ] && printf "  %-8s group=%-6s type=%-11s touches=%s\n    %s\n    file: %s\n" "$id" "$(field "$f" parallel_group)" "$(field "$f" type)" "$(field "$f" touches)" "$(field "$f" title)" "tasks/${f#$ROOT/}"
done
echo; echo "in_progress:"; for f in $(files); do [ "$(field "$f" status)" = in_progress ] && echo "  $(field "$f" id) $(field "$f" title)"; done
echo "blocked:"; for f in $(files); do [ "$(field "$f" status)" = blocked ] && echo "  $(field "$f" id) $(field "$f" title)"; done; true
