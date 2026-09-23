#!/usr/bin/env bash
# Records laravel.jsonl: runs steps.json against the Laravel contract stack
# (docker-compose.contract.yml, docs/go-migration/contract.md) starting from the contract
# fixtures, then snapshots the rows snapshot.sql selects. side_effects_int_test.go replays the
# same steps through the Go service and must produce the same rows.
#
#   CONTRACT_PORT=18090 backend-go/internal/healthlog/testdata/sideeffects/capture_laravel.sh
#
# Holds contract/.work/record.lock (like `make contract-record`) and restores the fixture dump
# afterwards, so recordings are unaffected.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../../../../.." && pwd)"
base="http://127.0.0.1:${CONTRACT_PORT:-8090}/api/v1"
compose=(docker compose -f "$repo/docker-compose.contract.yml")
dump="$repo/backend-go/contract/fixtures/dump.sql"
sql() { "${compose[@]}" exec -T contract-mariadb mariadb -uritme -pcontract ritme_contract -N -B -r "$@"; }

mkdir -p "$repo/backend-go/contract/.work"
exec 9>"$repo/backend-go/contract/.work/record.lock"
flock 9 2>/dev/null || python3 -c 'import fcntl,sys; fcntl.flock(9, fcntl.LOCK_EX)'

tokdir=$(mktemp -d)
sql < "$dump"
trap 'sql < "$dump"; rm -rf "$tokdir"' EXIT

tokdir=$(mktemp -d)
login() {
  local mobile=$1 code
  curl -sf -X POST "$base/auth/send-otp" -H 'Accept: application/json' -H 'Content-Type: application/json' \
    -d "{\"mobile\":\"$mobile\"}" >/dev/null
  code=$(sql -e "select code from otp_verifications where mobile='$mobile' order by id desc limit 1")
  curl -sf -X POST "$base/auth/verify-otp" -H 'Accept: application/json' -H 'Content-Type: application/json' \
    -d "{\"mobile\":\"$mobile\",\"code\":\"$code\"}" | jq -r .data.access_token
}

out="$here/laravel.jsonl"
: > "$out.tmp"
n=$(jq length "$here/steps.json")
for ((i = 0; i < n; i++)); do
  step=$(jq -c ".[$i]" "$here/steps.json")
  mobile=$(jq -r .mobile <<<"$step")
  [[ -s "$tokdir/$mobile" ]] || login "$mobile" > "$tokdir/$mobile"
  resp=$(curl -s -w '\n%{http_code}' -X POST "$base/health-logs" \
    -H 'Accept: application/json' -H 'Content-Type: application/json' \
    -H "Accept-Language: $(jq -r .locale <<<"$step")" -H "X-Test-Now: $(jq -r .now <<<"$step")" \
    -H "Authorization: Bearer $(cat "$tokdir/$mobile")" -d "$(jq -c .body <<<"$step")")
  status=$(tail -n1 <<<"$resp")
  jq -c --argjson i "$i" --argjson status "$status" \
    '{step: $i, status: $status, warning: (.warning // null)}' <<<"$(sed '$d' <<<"$resp")" >> "$out.tmp"
done
sql < "$here/snapshot.sql" >> "$out.tmp"
mv "$out.tmp" "$out"
echo "wrote $out"
