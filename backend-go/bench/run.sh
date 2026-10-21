#!/usr/bin/env bash
# Runs the T-M9-08 benchmarks and writes them, with a header naming the machine and the Go toolchain, to $1
# (default bench/current.txt). Used by `make bench` / `make bench-baseline` and .github/workflows/bench.yml.
#
#   BENCH_PKGS   packages (default: the engine, fertility, analysis, rules and auth packages)
#   BENCH_COUNT  samples per benchmark (default 6: benchstat's minimum for a confidence interval)
#   BENCH_TIME   -benchtime (default 1s)
set -euo pipefail
cd "$(dirname "$0")/.."

out="${1:-bench/current.txt}"
pkgs="${BENCH_PKGS:-./internal/cycle/... ./internal/fertility/... ./internal/analysis/... ./internal/messages/rules/... ./internal/auth/passport/...}"
count="${BENCH_COUNT:-6}"
benchtime="${BENCH_TIME:-1s}"

case "$(uname -s)" in
Darwin)
  cpu="$(sysctl -n machdep.cpu.brand_string)"
  cores="$(sysctl -n hw.ncpu)"
  mem="$(( $(sysctl -n hw.memsize) / 1073741824 )) GB"
  load="$(sysctl -n vm.loadavg | tr -d '{}' | xargs)"
  ;;
*)
  cpu="$(grep -m1 'model name' /proc/cpuinfo 2>/dev/null | cut -d: -f2 | xargs || echo unknown)"
  cores="$(nproc 2>/dev/null || echo '?')"
  mem="$(awk '/MemTotal/ {printf "%d GB", $2/1048576}' /proc/meminfo 2>/dev/null || echo '?')"
  load="$(cut -d' ' -f1-3 /proc/loadavg 2>/dev/null || echo '?')"
  ;;
esac

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
{
  echo "# Ritme backend-go benchmarks (T-M9-08). Compare: make bench-compare. Re-record: make bench-baseline."
  echo "# recorded: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "# commit: $(git rev-parse --short HEAD 2>/dev/null || echo unknown)$(git diff --quiet HEAD -- . 2>/dev/null || echo ' (dirty tree)')"
  echo "# machine: ${cpu}, ${cores} cores, ${mem} ($(go env GOOS)/$(go env GOARCH))"
  echo "# go: $(go env GOVERSION)"
  echo "# load at start (1/5/15 min): ${load}"
  echo "# command: go test -run '^\$' -bench . -benchmem -count ${count} -benchtime ${benchtime} ${pkgs}"
} >"$tmp"
# shellcheck disable=SC2086 # pkgs is a word list on purpose
go test -run '^$' -bench . -benchmem -count "$count" -benchtime "$benchtime" $pkgs | tee -a "$tmp"
mv "$tmp" "$out"
trap - EXIT
echo "wrote $out"
