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
#   MERGE_STAGE=1 ./deploy.sh         merge `stage` without asking (0 = never ask)
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail
cd "$(dirname "$0")"

SERVER="${SERVER:-root@89.251.8.115}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_ed25519}"
REMOTE_DIR="/opt/ritme"
# curl --resolve only accepts a literal address, so a SERVER override that
# names a host (or omits `user@`) must be resolved before we pin to it.
ORIGIN_IP="${ORIGIN_IP:-${SERVER##*@}}"
if [[ ! "$ORIGIN_IP" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  # `|| true`: a missing dig must not abort the deploy under `set -e`; the
  # empty result is reported below instead.
  ORIGIN_IP="$(dig +short A "$ORIGIN_IP" 2>/dev/null | grep -m1 -E '^[0-9.]+$' || true)"
  [[ -n "$ORIGIN_IP" ]] || echo "!! could not resolve ${SERVER##*@} — the origin-pinned check will be skipped" >&2
fi
SERVICES="${SERVICES:-}"

SSH_OPTS=(-i "$SSH_KEY" -o ConnectTimeout=20 -o ServerAliveInterval=15 -o ServerAliveCountMax=8)
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

ssh_run() { ssh "${SSH_OPTS[@]}" "$SERVER" "$@"; }

# ── Offer to merge `stage` first ─────────────────────────────────────────────
# Day-to-day work happens on the `stage` branch (that's what deploy-stage.sh
# ships). A production deploy run from another branch would therefore quietly
# ship an older tree, so ask before that happens. Skipped entirely when there is
# nothing to merge, when SKIP_SYNC=1 (the working tree isn't shipped at all), or
# when stdin isn't a terminal — set MERGE_STAGE=1 to merge unattended, or
# MERGE_STAGE=0 to never ask.
STAGE_BRANCH="${STAGE_BRANCH:-stage}"
maybe_merge_stage() {
  [[ "${MERGE_STAGE:-}" == "0" ]] && return 0
  [[ "${SKIP_SYNC:-0}" == "1" ]] && return 0
  git rev-parse --git-dir >/dev/null 2>&1 || return 0

  local current ahead
  current="$(git rev-parse --abbrev-ref HEAD)"
  [[ "$current" == "$STAGE_BRANCH" ]] && return 0
  git rev-parse --verify "${STAGE_BRANCH}^{commit}" >/dev/null 2>&1 || return 0

  # Commits on stage that this branch doesn't have yet. Zero => nothing to do.
  ahead="$(git rev-list --count "HEAD..${STAGE_BRANCH}")"
  [[ "$ahead" == "0" ]] && return 0

  echo
  echo "!! '${STAGE_BRANCH}' has ${ahead} commit(s) that '${current}' does not:" >&2
  git --no-pager log --oneline --no-decorate "HEAD..${STAGE_BRANCH}" | sed 's/^/     /' >&2
  echo "   Deploying now would ship the OLDER tree." >&2

  if [[ -n "$(git status --porcelain)" ]]; then
    echo "!! Working tree is dirty — cannot merge automatically." >&2
    echo "   Commit or stash, then re-run (or continue and ship as-is)." >&2
    return 0
  fi

  if [[ "${MERGE_STAGE:-}" != "1" ]]; then
    if [[ ! -t 0 ]]; then
      echo "   (non-interactive: not merging; use MERGE_STAGE=1 to merge)" >&2
      return 0
    fi
    local reply
    read -r -p "   Merge '${STAGE_BRANCH}' into '${current}' before deploying? [y/N] " reply
    [[ "$reply" =~ ^[Yy]$ ]] || { echo "   Skipping merge — shipping '${current}' as-is."; return 0; }
  fi

  echo "==> Merging '${STAGE_BRANCH}' into '${current}' ..."
  if ! git merge --no-edit "$STAGE_BRANCH"; then
    git merge --abort || true
    echo "!! Merge conflicted and was aborted. Resolve it by hand, then re-run." >&2
    exit 1
  fi
  echo "   Now at $(git rev-parse --short HEAD)."
}
maybe_merge_stage

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
    --exclude 'stage.htpasswd' \
    --exclude 'stage-gate.conf' \
    --exclude 'backend/storage/logs/*' \
    --exclude 'backend/storage/framework/cache/*' \
    --exclude 'backend/database/*.sqlite' \
    -e "ssh ${SSH_OPTS[*]}" \
    --progress \
    ./ "${SERVER}:${REMOTE_DIR}/"
fi

# The proxy joins the external `ritme-edge` bridge (shared with the separate
# `ritme-stage` compose project) and bind-mounts ./stage.htpasswd. Neither is
# rsynced, so create them if absent — otherwise compose errors on the missing
# network, or Docker silently materialises a DIRECTORY where nginx wants a file.
echo "==> Ensuring shared edge network and staging htpasswd exist..."
ssh_run "docker network inspect ritme-edge >/dev/null 2>&1 || docker network create ritme-edge" >/dev/null
ssh_run "test -e ${REMOTE_DIR}/stage.htpasswd || : > ${REMOTE_DIR}/stage.htpasswd"
# stage-gate.conf is mounted into conf.d, so an EMPTY one is not good enough:
# whenever the staging vhost is switched on it reads $stage_auth_realm from it
# and nginx refuses to start on an unknown variable. Generate a valid file.
# shellcheck source=deploy/stage-gate.sh
source ./deploy/stage-gate.sh
ensure_stage_gate "${REMOTE_DIR}"

if [[ "${NO_BUILD:-0}" != "1" ]]; then
  echo "==> Building images on the server (this is the slow part)..."
  # The build arg (NEXT_PUBLIC_API_BASE_URL) comes from the server's .env,
  # which compose loads automatically — so the public API URL baked into the
  # browser bundle is configured once, on the server.
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
  got="$(curl -s -o /dev/null -m 25 -w '%{http_code}' "$@" "$url")" || got=000
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
# web.ritme.app resolves to a CDN edge, not this server. A deploy check must
# test what the deploy actually changed, so pin the request to the origin IP.
if [[ -n "$ORIGIN_IP" ]]; then
  check 307 "https://web.ritme.app/" --resolve "web.ritme.app:443:${ORIGIN_IP}"
fi

if (( failures > 0 )); then
  echo "❌ Deploy finished but ${failures} check(s) failed — see above." >&2
  exit 1
fi

echo "✅ Deploy done."
