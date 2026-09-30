#!/usr/bin/env bash
# Per-epic progress board: done/total, in progress, blocked, next runnable.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
field() { grep -m1 "^$2:" "$1" | sed -E "s/^$2: *//"; }
printf "%-16s %-22s %5s %5s %5s %5s\n" EPIC PROGRESS done todo wip block
for ep in $(ls -d "$ROOT"/E*/ | xargs -n1 basename | sort -V); do
  t=0 d=0 w=0 b=0 o=0
  for f in "$ROOT/$ep"/CB-*.md; do [ -e "$f" ] || continue; t=$((t+1))
    case "$(field "$f" status)" in done) d=$((d+1));; in_progress) w=$((w+1));; blocked) b=$((b+1));; *) o=$((o+1));; esac; done
  [ $t = 0 ] && continue
  n=$((d*20/t)); bar=$(printf "%${n}s" | tr ' ' '#')$(printf "%$((20-n))s" | tr ' ' '.')
  printf "%-16s [%s] %5s %5s %5s %5s\n" "$ep" "$bar" "$d/$t" "$o" "$w" "$b"
done
