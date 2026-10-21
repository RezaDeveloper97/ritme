#!/usr/bin/env bash
# loadtest/run.sh <scenario> | --inspect — runs one k6 scenario against the LOCAL API and summarises the server
# side (/metrics scrape before/after → queries per request, server percentiles, 5xx). Called by `make loadtest`.
#
# Recipe (all local, nothing leaves the machine):
#   make test-db-up && make loadtest-db            # scratch DB ritme_loadtest on the test stack (never ritme_test)
#   make loadtest-seed                              # 2 000 users + loadtest/.tokens.json (RESET=1 to start over)
#   make loadtest-api                               # other terminal: API :8020, /metrics 127.0.0.1:9090
#   make loadtest SCENARIO=app_open                 # docker k6 (K6_RUNNER=local for a k6 on PATH)
#   make loadtest-drop                              # when done
#
# Env: K6_RUNNER=docker|local, K6_IMAGE, BASE_URL (loopback only), METRICS_URL, RPS, DURATION, COLD_WAIT, SOAK_*.
set -euo pipefail
cd "$(dirname "$0")/.."

scenario="${1:-}"
runner="${K6_RUNNER:-docker}"
image="${K6_IMAGE:-grafana/k6:1.3.0}"
metrics="${METRICS_URL:-http://127.0.0.1:9090/metrics}"
base="${BASE_URL:-http://127.0.0.1:8020}"
scenarios=(app_open calendar_browse log_burst analysis_tab fertility_tab soak)

k6run() { # k6run <k6 args...>; scripts live in loadtest/, mounted at /loadtest in docker
  if [[ "$runner" == local ]]; then
    (cd loadtest && k6 "$@")
  else
    # host.docker.internal reaches the API on the host; lib.js only accepts loopback / host.docker.internal.
    docker run --rm -i -v "$PWD/loadtest:/loadtest" -w /loadtest \
      -e BASE_URL="${base/127.0.0.1/host.docker.internal}" -e RPS -e DURATION -e COLD_WAIT -e SOAK_RATE -e SOAK_DURATION \
      "$image" "$@"
  fi
}

if [[ "$scenario" == --inspect ]]; then
  TOKENS_FILE=./.tokens.json
  if [[ ! -f loadtest/.tokens.json ]]; then # lint needs only a parseable file
    echo '[]' >loadtest/.tokens.lint.json
    TOKENS_FILE=./.tokens.lint.json
  fi
  for s in "${scenarios[@]}"; do
    echo "== k6 inspect $s"
    if [[ "$runner" == local ]]; then (cd loadtest && k6 inspect -e TOKENS_FILE="$TOKENS_FILE" "$s.js" >/dev/null); \
    else docker run --rm -v "$PWD/loadtest:/loadtest" -w /loadtest "$image" inspect -e TOKENS_FILE="$TOKENS_FILE" "$s.js" >/dev/null; fi
  done
  rm -f loadtest/.tokens.lint.json
  echo "inspect ok"
  exit 0
fi

if [[ ! " ${scenarios[*]} " =~ " ${scenario} " ]]; then
  echo "usage: loadtest/run.sh <${scenarios[*]// /|}>  (or --inspect)" >&2
  exit 2
fi
[[ -f loadtest/.tokens.json ]] || { echo "loadtest/.tokens.json missing: run make loadtest-seed" >&2; exit 1; }
case "$base" in http://127.0.0.1:*|http://localhost:*) ;; *) echo "refusing BASE_URL=$base: local APIs only" >&2; exit 1 ;; esac
curl -fsS "$base/up" >/dev/null || { echo "API not answering on $base/up (make loadtest-api)" >&2; exit 1; }

out="loadtest/results"; mkdir -p "$out"
stamp="$(date +%Y%m%d-%H%M%S)"
run="$out/$scenario-$stamp"
go run ./loadtest/metricsdiff scrape -url "$metrics" -out "$run.before.prom"
set +e
k6run run --summary-export="results/$scenario-$stamp.summary.json" "$scenario.js" | tee "$run.k6.txt"
rc=${PIPESTATUS[0]}
set -e
go run ./loadtest/metricsdiff scrape -url "$metrics" -out "$run.after.prom"
{
  echo "## $scenario ($stamp) — server side, from $metrics"
  go run ./loadtest/metricsdiff diff -before "$run.before.prom" -after "$run.after.prom"
} | tee "$run.server.md"
echo "k6 exit code $rc (99 = a threshold failed). Files: $run.*"
exit "$rc"
