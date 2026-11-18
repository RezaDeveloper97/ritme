#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# T-M12-02 smoke: a forged X-Forwarded-For must not move a client to another
# rate-limit bucket, and (on stage) the password gate must hold for external
# IPv4 / IPv6 clients without the cookie — whatever "internal" address they
# claim.
#
#   bash deploy/smoke/forged-xff.sh https://stage.ritmeapp.ir
#   bash deploy/smoke/forged-xff.sh https://beta.ritme.app
#
# Environment (all optional; secrets are never printed):
#   STAGE_BASIC_AUTH  user:password for the stage gate (from the server-only
#                     /root/ritme-stage-credentials.txt). Needed on stage for the
#                     rate-limit part; passed to curl on stdin, not argv.
#   ORIGIN_IP         the origin's IPv4 (e.g. 89.251.8.115): ALSO run the
#                     rate-limit check straight to the origin, bypassing the CDN —
#                     the S-03 attack path. Recommended for beta.
#   ORIGIN_IP6        the origin's IPv6, for the IPv6 gate check when the host
#                     name has no AAAA record.
#   PROBE_PATH        a guest, IP-throttled GET that returns X-RateLimit-* headers
#                     (default /api/v1/city-services/options, 120/min).
#   GATE              1 | 0 | auto (default: auto = host starts with "stage.").
#   STRICT_IPV6       1 = an IPv6 check that cannot run is a FAIL, not a SKIP.
#
# Exit 0 when every check passed (SKIPs allowed unless STRICT_IPV6=1), 1 otherwise.
# Spends ~8 requests of the probe's per-IP budget per path tested.
# ─────────────────────────────────────────────────────────────────────────────
set -o pipefail

BASE="${1:-}"
if [[ -z "$BASE" || ! "$BASE" =~ ^https?:// ]]; then
  echo "usage: $0 <base-url>   e.g. $0 https://stage.ritmeapp.ir" >&2
  exit 2
fi
BASE="${BASE%/}"
HOST="${BASE#*://}"; HOST="${HOST%%/*}"; HOST="${HOST%%:*}"
case "$BASE" in https://*) PORT=443 ;; *) PORT=80 ;; esac
PROBE_PATH="${PROBE_PATH:-/api/v1/city-services/options}"
GATE="${GATE:-auto}"
if [[ "$GATE" == auto ]]; then
  case "$HOST" in stage.*) GATE=1 ;; *) GATE=0 ;; esac
fi

fails=0
skips=0
pass() { printf '  PASS  %s\n' "$*"; }
fail() { printf '  FAIL  %s\n' "$*"; fails=$((fails + 1)); }
skip() { printf '  SKIP  %s\n' "$*"; skips=$((skips + 1)); }

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# curl with the stage credentials (if any) fed through a config on stdin, so they
# never appear in argv / `ps`. Extra args are passed through.
curl_auth() {
  if [[ -n "${STAGE_BASIC_AUTH:-}" ]]; then
    printf 'user = "%s"\n' "${STAGE_BASIC_AUTH//\"/\\\"}" | curl -K - "$@"
  else
    curl "$@"
  fi
}

# header <file> <name> → value (case-insensitive, last occurrence, CR stripped)
header() { grep -i "^$2:" "$1" | tail -n1 | cut -d: -f2- | tr -d ' \r'; }

# ── 1. Rate-limit key ignores forged X-Forwarded-For / X-Real-IP ────────────
# One honest request, then several that each claim a different client. Through
# the same real client they must all draw from ONE bucket: X-RateLimit-Remaining
# keeps falling (or hits 429). A forged header that changed the key would show
# a fresh bucket (Remaining back near the limit).
ratelimit_check() { # ratelimit_check <label> [curl args to reach the origin]
  local label="$1"; shift
  local url="${BASE}${PROBE_PATH}" code limit prev rem i xff
  local forged=(
    "203.0.113.7"
    "198.51.100.23, 203.0.113.99"
    "10.0.0.5"
    "172.18.0.5"
    "127.0.0.1"
    "2001:db8::beef"
  )
  echo "== rate-limit key (${label}): ${PROBE_PATH}"
  code="$(curl_auth -sk "$@" -o /dev/null -D "$TMP/h0" -w '%{http_code}' "$url")"
  limit="$(header "$TMP/h0" X-RateLimit-Limit)"
  prev="$(header "$TMP/h0" X-RateLimit-Remaining)"
  if [[ "$code" == 429 ]]; then
    fail "baseline already 429 — the bucket is spent; wait a minute and rerun"
    return
  fi
  if [[ "$code" != 200 || -z "$limit" || -z "$prev" ]]; then
    fail "baseline answered ${code} without X-RateLimit headers — pick another PROBE_PATH (guest, IP-throttled GET)"
    return
  fi
  i=0
  for xff in "${forged[@]}"; do
    i=$((i + 1))
    code="$(curl_auth -sk "$@" -o /dev/null -D "$TMP/h$i" -w '%{http_code}' \
      -H "X-Forwarded-For: ${xff}" -H "X-Real-IP: ${xff%%,*}" "$url")"
    rem="$(header "$TMP/h$i" X-RateLimit-Remaining)"
    if [[ "$code" == 429 ]]; then
      pass "forged '${xff}' → 429 from the same exhausted bucket"
      prev=0
      continue
    fi
    if [[ -z "$rem" ]]; then
      fail "forged '${xff}' → ${code}, no X-RateLimit-Remaining"
    elif (( rem < prev )); then
      pass "forged '${xff}' → same bucket (remaining ${prev} → ${rem} of ${limit})"
    else
      fail "forged '${xff}' → NEW bucket (remaining ${prev} → ${rem} of ${limit}): the header chose the key"
    fi
    prev="$rem"
  done
}

if [[ "$GATE" == 1 && -z "${STAGE_BASIC_AUTH:-}" ]]; then
  echo "== rate-limit key"
  skip "STAGE_BASIC_AUTH not set — the gated API cannot be probed"
else
  ratelimit_check "via ${HOST}"
  if [[ -n "${ORIGIN_IP:-}" ]]; then
    # Straight to the origin, the CDN bypassed (TLS still for HOST).
    ratelimit_check "direct to origin ${ORIGIN_IP}" --connect-to "${HOST}:${PORT}:${ORIGIN_IP}:${PORT}"
  fi
fi

# ── 2. Stage gate: external clients without the cookie get 401 ──────────────
gate_check() { # gate_check <family flag -4|-6> <label> [extra curl args]
  local fam="$1" label="$2"; shift 2
  local path code xff
  for path in / /api/v1/languages /api/v1/auth/me; do
    code="$(curl -sk "$fam" "$@" -o /dev/null -w '%{http_code}' --max-time 15 "${BASE}${path}")"
    if [[ "$code" == 000 ]]; then
      return 3 # no route / no record for this family
    fi
    if [[ "$code" == 401 ]]; then pass "${label} ${path} without cookie → 401"
    else fail "${label} ${path} without cookie → ${code} (expected 401)"; fi
  done
  # Claiming to be a container (the old 10/8 + 172.16/12 exemption) or loopback.
  for xff in 172.18.0.5 10.0.0.2 127.0.0.1 "203.0.113.1, 172.18.0.1"; do
    code="$(curl -sk "$fam" "$@" -o /dev/null -w '%{http_code}' --max-time 15 \
      -H "X-Forwarded-For: ${xff}" -H "X-Real-IP: ${xff%%,*}" "${BASE}/api/v1/languages")"
    if [[ "$code" == 401 ]]; then pass "${label} forged '${xff}' without cookie → 401"
    else fail "${label} forged '${xff}' without cookie → ${code} (expected 401)"; fi
  done
  code="$(curl -sk "$fam" "$@" -o /dev/null -w '%{http_code}' --max-time 15 \
    -H "Cookie: ritme_stage=000000000000000000000000000000000000000000000000" "${BASE}/")"
  if [[ "$code" == 401 ]]; then pass "${label} wrong ritme_stage cookie → 401"
  else fail "${label} wrong ritme_stage cookie → ${code} (expected 401)"; fi
  return 0
}

if [[ "$GATE" == 1 ]]; then
  echo "== stage gate (no cookie)"
  gate_check -4 "IPv4" || fail "IPv4: ${HOST} unreachable over IPv4"
  if ! gate_check -6 "IPv6"; then
    if [[ -n "${ORIGIN_IP6:-}" ]]; then
      gate_check -6 "IPv6 (origin ${ORIGIN_IP6})" --connect-to "${HOST}:${PORT}:[${ORIGIN_IP6}]:${PORT}" \
        || { [[ "${STRICT_IPV6:-0}" == 1 ]] && fail "IPv6: origin ${ORIGIN_IP6} unreachable" || skip "IPv6: origin ${ORIGIN_IP6} unreachable from here"; }
    elif [[ "${STRICT_IPV6:-0}" == 1 ]]; then
      fail "IPv6: ${HOST} unreachable over IPv6 (no AAAA or no IPv6 route here; set ORIGIN_IP6)"
    else
      skip "IPv6: ${HOST} unreachable over IPv6 (no AAAA or no IPv6 route here; set ORIGIN_IP6)"
    fi
  fi

  # S-05: the cookie handed out after the password carries Secure.
  if [[ -n "${STAGE_BASIC_AUTH:-}" ]]; then
    curl_auth -sk -o /dev/null -D "$TMP/hc" "${BASE}/" || true
    if grep -i '^set-cookie: *ritme_stage=' "$TMP/hc" | grep -qi '; *secure'; then
      pass "ritme_stage cookie is Secure"
    elif grep -qi '^set-cookie: *ritme_stage=' "$TMP/hc"; then
      fail "ritme_stage cookie lacks Secure"
    else
      fail "no ritme_stage cookie after a password login"
    fi
  else
    skip "cookie Secure flag: STAGE_BASIC_AUTH not set"
  fi
fi

echo
if (( fails > 0 )); then
  echo "forged-xff smoke: ${fails} FAIL, ${skips} SKIP — ${BASE}"
  exit 1
fi
echo "forged-xff smoke: OK (${skips} SKIP) — ${BASE}"
