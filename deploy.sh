#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Deploy Ritme to production.  Run from the repo root:  ./deploy.sh
#
# Server: root@89.251.8.115 (Ubuntu 24.04, Docker, /opt/ritme).
#
# Unlike the previous host, this server has full international connectivity, so
# images are BUILT ON THE SERVER. The deploy is therefore just:
#     rsync the working tree  ->  docker compose build  ->  up -d
# No local cross-compilation, no `docker save` streaming of multi-hundred-MB
# tarballs. Whatever is in your working tree is what ships — committing is not
# required (but is a good idea).
#
# Server-only state that rsync must never touch: .env (prod secrets), ssl/,
# certbot-www/ — all excluded below.
#
# Useful switches:
#   SERVICES="frontend" ./deploy.sh   rebuild/restart only some services
#   NO_BUILD=1 ./deploy.sh            ship files + restart, skip image builds
#   SKIP_SYNC=1 ./deploy.sh           rebuild from what's already on the server
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail
cd "$(dirname "$0")"

SERVER="${SERVER:-root@89.251.8.115}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_ed25519}"
REMOTE_DIR="/opt/ritme"
SERVICES="${SERVICES:-}"

SSH_OPTS=(-i "$SSH_KEY" -o ConnectTimeout=20 -o ServerAliveInterval=15 -o ServerAliveCountMax=8)
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

ssh_run() { ssh "${SSH_OPTS[@]}" "$SERVER" "$@"; }

if [[ "${SKIP_SYNC:-0}" != "1" ]]; then
  echo "==> Syncing source to ${SERVER}:${REMOTE_DIR} ..."
  # --delete keeps the server tree an exact mirror, so deleted files really go
  # away; every server-only path is excluded so it survives that.
  rsync -az --delete --stats \
    --exclude '.git/' \
    --exclude 'node_modules/' \
    --exclude 'vendor/' \
    --exclude '.next/' \
    --exclude 'application/' \
    --exclude '*.tar.gz' \
    --exclude '*.fig' \
    --exclude '*.log' \
    --exclude '.env' \
    --exclude '.env.local' \
    --exclude 'ssl/' \
    --exclude 'certbot-www/' \
    --exclude 'backend/storage/logs/*' \
    --exclude 'backend/storage/framework/cache/*' \
    --exclude 'backend/database/*.sqlite' \
    -e "ssh ${SSH_OPTS[*]}" \
    ./ "${SERVER}:${REMOTE_DIR}/" | tail -4
fi

if [[ "${NO_BUILD:-0}" != "1" ]]; then
  echo "==> Building images on the server (this is the slow part)..."
  # Build args (NEXT_PUBLIC_API_BASE_URL, NEXT_PUBLIC_OTP_TEST_MODE) come from
  # the server's .env, which compose loads automatically — so the public API
  # URL baked into the browser bundle is configured once, on the server.
  ssh_run "cd ${REMOTE_DIR} && ${COMPOSE} build ${SERVICES}"
fi

echo "==> Starting stack..."
ssh_run "cd ${REMOTE_DIR} && ${COMPOSE} up -d ${SERVICES}"

echo "==> Pruning dangling images..."
# Each rebuild orphans the previous image layer set; without this the 24 GB
# disk fills after a handful of deploys.
ssh_run "docker image prune -f" >/dev/null || true

echo "==> Verifying..."
sleep 8
ssh_run "cd ${REMOTE_DIR} && ${COMPOSE} ps --format '{{.Service}}: {{.Status}}'"

# Each check ASSERTS an exact status. Printing codes without comparing them is
# not enough: a PHP fatal in bootstrap/app.php still answers 200 (with an error
# page as the body), so a "200 everywhere" run once hid a completely dead API.
# Hence the protected route below must be 401 — that proves the framework
# actually booted and ran the auth middleware, not merely that PHP replied.
failures=0
check() { # check <expected> <label> [extra curl args...]
  local expected="$1" url="$2"; shift 2
  local got
  got="$(curl -s -o /dev/null -m 25 -w '%{http_code}' "$@" "$url" || echo 000)"
  if [[ "$got" == "$expected" ]]; then
    printf '  ok    %s  %s\n' "$got" "$url"
  else
    printf '  FAIL  %s (expected %s)  %s\n' "$got" "$expected" "$url"
    failures=$((failures + 1))
  fi
}

check 200 "https://api.ritme.app/up"
check 401 "https://api.ritme.app/api/v1/banners" -H 'Accept: application/json'
check 404 "https://api.ritme.app/admin"
check 200 "https://adpanell.ritme.app/admin/login"
check 404 "https://adpanell.ritme.app/api/v1/banners"
check 307 "https://web.ritme.app/"

if (( failures > 0 )); then
  echo "❌ Deploy finished but ${failures} check(s) failed — see above." >&2
  exit 1
fi

echo "✅ Deploy done."
