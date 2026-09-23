#!/usr/bin/env bash
# Records laravel.jsonl: runs steps.json (period log writes) against the Laravel contract stack
# (docker-compose.contract.yml, docs/go-migration/contract.md) starting from the contract
# fixtures, then snapshots the rows snapshot.sql selects. sequence_int_test.go replays the same
# steps through the Go period service and must leave the same rows.
#
#   CONTRACT_PORT=18090 backend-go/internal/cycle/periods/testdata/sequence/capture_laravel.sh
#
# Holds contract/.work/record.lock (like `make contract-record`) and restores the fixture dump
# afterwards, so recordings are unaffected.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../../../../../.." && pwd)"
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
  user=$(jq -r .user <<<"$step")
  [[ -s "$tokdir/$mobile" ]] || login "$mobile" > "$tokdir/$mobile"
  path=$(jq -r .path <<<"$step")
  target=$(jq -r '.target // empty' <<<"$step")
  if [[ -n "$target" ]]; then
    id=$(sql -e "select id from cycle_histories where user_id=$user and period_start_date='$target'")
    path=${path/\{id\}/$id}
  fi
  resp=$(curl -s -w '\n%{http_code}' -X "$(jq -r .method <<<"$step")" "$base$path" \
    -H 'Accept: application/json' -H 'Content-Type: application/json' \
    -H "Accept-Language: $(jq -r .locale <<<"$step")" -H "X-Test-Now: $(jq -r .now <<<"$step")" \
    -H "Authorization: Bearer $(cat "$tokdir/$mobile")" -d "$(jq -c '.body // {}' <<<"$step")")
  status=$(tail -n1 <<<"$resp")
  jq -c --argjson i "$i" --argjson status "$status" \
    '{step: $i, status: $status, code: (.code // null), warnings: ((.data.warnings // []) | map(.type))}' \
    <<<"$(sed '$d' <<<"$resp")" >> "$out.tmp"
done
sql < "$here/snapshot.sql" >> "$out.tmp"
mv "$out.tmp" "$out"
echo "wrote $out"
