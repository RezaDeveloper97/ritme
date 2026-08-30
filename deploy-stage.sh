#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Deploy Ritme STAGING to stage.ritmeapp.ir.  Run from the repo root:
#     ./deploy-stage.sh
#
# Same server as production (root@89.251.8.115), different everything else:
#
#   source   /opt/ritme-stage          (prod: /opt/ritme)
#   project  ritme-stage               (prod: ritme)
#   volumes  ritme-stage_mysql-data …  (prod: ritme_mysql-data …)
#   URL      https://stage.ritmeapp.ir (prod: api/adpanell/web.ritme.app)
#
# Unlike deploy.sh, which ships your working tree, this ships the tip of the
# `stage` BRANCH via `git archive` — staging is meant to reflect a reviewable
# commit, not whatever happens to be on your disk. Override with:
#
#   BRANCH=my-feature ./deploy-stage.sh    ship a different branch
#   FROM_WORKTREE=1   ./deploy-stage.sh    ship the working tree instead
#
# The staging stack publishes NO host ports. Its only door is production's
# nginx, which gates the whole hostname behind HTTP Basic auth — so the
# password cannot be sidestepped by hitting a port directly.
#
# Other switches (same meaning as deploy.sh):
#   SERVICES="frontend" ./deploy-stage.sh
#   NO_BUILD=1          ./deploy-stage.sh
#   SKIP_SYNC=1         ./deploy-stage.sh
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail
cd "$(dirname "$0")"

SERVER="${SERVER:-root@89.251.8.115}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_ed25519}"
REMOTE_DIR="/opt/ritme-stage"
PROD_DIR="/opt/ritme"          # where the shared proxy and its config live
STAGE_HOST="${STAGE_HOST:-stage.ritmeapp.ir}"
BRANCH="${BRANCH:-stage}"
SERVICES="${SERVICES:-}"

SSH_OPTS=(-i "$SSH_KEY" -o ConnectTimeout=20 -o ServerAliveInterval=15 -o ServerAliveCountMax=8)
COMPOSE="docker compose -p ritme-stage -f docker-compose.yml -f docker-compose.stage.yml"
PROD_COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

ssh_run() { ssh "${SSH_OPTS[@]}" "$SERVER" "$@"; }

# ── 1. Ship the source ──────────────────────────────────────────────────────
if [[ "${SKIP_SYNC:-0}" != "1" ]]; then
  if [[ "${FROM_WORKTREE:-0}" == "1" ]]; then
    SRC="./"
    echo "==> Syncing WORKING TREE to ${SERVER}:${REMOTE_DIR} ..."
  else
    git rev-parse --verify "${BRANCH}^{commit}" >/dev/null 2>&1 \
      || { echo "!! branch '${BRANCH}' does not exist locally" >&2; exit 1; }
    # `git archive` into a temp dir gives an exact, reproducible snapshot of the
    # branch — no untracked scratch files, no uncommitted edits leaking into an
    # environment whose whole point is "this is what the commit does".
    SRC="$(mktemp -d)/"
    trap 'rm -rf "${SRC%/}"' EXIT
    git archive "$BRANCH" | tar -x -C "$SRC"
    echo "==> Syncing branch '${BRANCH}' ($(git rev-parse --short "$BRANCH")) to ${SERVER}:${REMOTE_DIR} ..."
  fi

  ssh_run "mkdir -p ${REMOTE_DIR}"
  rsync -az --delete --stats \
    --exclude '.git/' \
    --exclude 'node_modules/' \
    --exclude 'vendor/' \
    --exclude '.next/' \
    --exclude 'application/' \
    --exclude 'twa/' \
    --exclude 'android-shell/' \
    --exclude '*.tar.gz' \
    --exclude '*.fig' \
    --exclude '*.log' \
    --exclude '.playwright-mcp/' \
    --exclude '.env' \
    --exclude '.env.local' \
    --exclude 'ssl/' \
    --exclude 'certbot-www/' \
    --exclude 'backend/storage/logs/*' \
    --exclude 'backend/storage/framework/cache/*' \
    --exclude 'backend/database/*.sqlite' \
    -e "ssh ${SSH_OPTS[*]}" \
    --progress \
    "$SRC" "${SERVER}:${REMOTE_DIR}/"
fi

# ── 2. Preconditions on the server ──────────────────────────────────────────
# The shared edge network must exist before either project starts, and the
# staging .env must exist before compose can interpolate the required vars
# (APP_KEY, DB_PASSWORD, ADMIN_SEED_PASSWORD) — compose fails loudly if not.
ssh_run "docker network inspect ritme-edge >/dev/null 2>&1 || docker network create ritme-edge" >/dev/null
ssh_run "test -f ${REMOTE_DIR}/.env" || {
  echo "!! ${REMOTE_DIR}/.env is missing on the server." >&2
  echo "   Create it from .env.stage.example (server-only; never rsynced)." >&2
  exit 1
}
ssh_run "test -s ${PROD_DIR}/stage.htpasswd" || {
  echo "!! ${PROD_DIR}/stage.htpasswd is missing or empty — the staging vhost" >&2
  echo "   would deny everyone. Create it with:" >&2
  echo "   ssh ${SERVER} \"htpasswd -Bbc ${PROD_DIR}/stage.htpasswd <user> '<password>'\"" >&2
  exit 1
}

# ── 3. Build and start staging ──────────────────────────────────────────────
if [[ "${NO_BUILD:-0}" != "1" ]]; then
  echo "==> Building staging images on the server..."
  ssh_run "cd ${REMOTE_DIR} && ${COMPOSE} build ${SERVICES}"
fi

echo "==> Starting staging stack..."
ssh_run "cd ${REMOTE_DIR} && ${COMPOSE} up -d ${SERVICES}"

echo "==> Pruning dangling images..."
# Two stacks on one 24 GB disk orphan layers twice as fast as one did.
ssh_run "docker image prune -f" >/dev/null || true

# ── 4. Make sure the shared proxy is serving the staging vhost ──────────────
# STAGE_CONF starts unset (staging off). Turn it on at :80; running
# deploy/enable-https-stage.sh later flips it to the TLS variant, and this
# never downgrades that.
echo "==> Ensuring the proxy serves ${STAGE_HOST}..."
ssh_run "cd ${PROD_DIR} && grep -q '^STAGE_CONF=' .env || echo 'STAGE_CONF=./deploy/stage-http.conf' >> .env"
# `up -d proxy` is a no-op when the mount sources are unchanged, and recreates
# the container when STAGE_CONF was just added or flipped.
ssh_run "cd ${PROD_DIR} && ${PROD_COMPOSE} up -d proxy && ${PROD_COMPOSE} exec -T proxy nginx -t"

# ── 5. Verify ───────────────────────────────────────────────────────────────
echo "==> Verifying..."
sleep 8
ssh_run "cd ${REMOTE_DIR} && ${COMPOSE} ps --format '{{.Service}}: {{.Status}}'"

# Every check asserts an exact status — see the long comment in deploy.sh for
# why printing codes without comparing them once hid a completely dead API.
failures=0
check() { # check <expected> <url> [extra curl args...]
  local expected="$1" url="$2"; shift 2
  local got
  got="$(curl -s -o /dev/null -m 25 -w '%{http_code}' "$@" "$url")" || got=000
  if [[ "$got" == "$expected" ]]; then
    printf '  ok    %s  %s\n' "$got" "$url"
  else
    printf '  FAIL  %s (expected %s)  %s\n' "$got" "$expected" "$url"
    failures=$((failures + 1))
  fi
}

SCHEME="$(ssh_run "cd ${PROD_DIR} && grep -q 'stage-ssl.conf' .env && echo https || echo http")"
BASE="${SCHEME}://${STAGE_HOST}"

# /up is deliberately exempt from Basic auth so probes keep working.
check 200 "${BASE}/up"
# No credentials anywhere else => 401. This is the assertion that proves the
# password gate is actually on; a 200 here means staging is world-readable.
check 401 "${BASE}/"
check 401 "${BASE}/api/v1/banners" -H 'Accept: application/json'
check 401 "${BASE}/admin/login"

if [[ -n "${STAGE_BASIC_AUTH:-}" ]]; then
  # STAGE_BASIC_AUTH="user:password" — checks what's BEHIND the gate too.
  check 200 "${BASE}/" -u "$STAGE_BASIC_AUTH"
  check 200 "${BASE}/admin/login" -u "$STAGE_BASIC_AUTH"
  # 401 from Laravel (not nginx): proves the framework booted and the auth
  # middleware ran, rather than PHP merely answering.
  check 401 "${BASE}/api/v1/banners" -u "$STAGE_BASIC_AUTH" -H 'Accept: application/json'
else
  echo "  note  set STAGE_BASIC_AUTH='user:pass' to also verify behind the gate"
fi

if (( failures > 0 )); then
  echo "❌ Staging deploy finished but ${failures} check(s) failed — see above." >&2
  exit 1
fi

echo "✅ Staging deploy done — ${BASE}"
