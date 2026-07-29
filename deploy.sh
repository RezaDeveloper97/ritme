#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Deploy ritme to the production server (ritmeapp.ir / 62.60.198.240).
#
# The server has no access to GitHub/Packagist/npm (Iranian network filtering),
# so images CANNOT be built there. This script builds them locally for
# linux/amd64, then ships them over a slow/flaky link using rsync with
# --partial --append-verify so an interrupted upload RESUMES instead of
# restarting from 0%. Run from the repo root:  ./deploy.sh
#
# If the transfer dies (network drop, Ctrl+C), just run ./deploy.sh again —
# rsync continues from where it stopped.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail
cd "$(dirname "$0")"

SERVER="root@62.60.198.240"
SSH_KEY="$HOME/.ssh/id_ed25519"
API_BASE_URL="http://ritmeapp.ir/api/v1"
# Test mode ON for now: SMS.ir gateway is failing, so login accepts 1111
# instead of sending a real OTP. Flip back to "false" once SMS works.
OTP_TEST_MODE="true"

# Keepalive so a long, slow upload isn't dropped mid-stream; retry-friendly.
SSH_OPTS=(-i "$SSH_KEY" -o ConnectTimeout=20 -o ServerAliveInterval=15 -o ServerAliveCountMax=8)

REMOTE_DIR="/opt/ritme"
LOCAL_TAR="./.deploy-images.tar.gz"       # local staging file (gitignored)
REMOTE_TAR="${REMOTE_DIR}/.deploy-images.tar.gz"

# Resume works only if the staged tar is byte-identical across runs, so we do
# NOT rebuild when a staged tar already exists — a fresh build would change the
# bytes and make the partial remote file useless. To force a clean rebuild
# (new code), delete the tar or run:  FRESH=1 ./deploy.sh
if [[ "${FRESH:-0}" == "1" ]]; then
  rm -f "$LOCAL_TAR"
fi

if [[ -f "$LOCAL_TAR" ]]; then
  echo "==> Reusing existing ${LOCAL_TAR} ($(( $(wc -c < "$LOCAL_TAR") / 1024 / 1024 )) MB) — resuming upload."
  echo "    (delete it or run 'FRESH=1 ./deploy.sh' to rebuild from current code)"
else
  echo "==> Building backend (linux/amd64)..."
  docker build --platform linux/amd64 -t ritme-backend:latest ./backend

  echo "==> Building frontend (linux/amd64)..."
  docker build --platform linux/amd64 \
    --build-arg NEXT_PUBLIC_API_BASE_URL="$API_BASE_URL" \
    --build-arg NEXT_PUBLIC_OTP_TEST_MODE="$OTP_TEST_MODE" \
    -t ritme-frontend:latest ./frontend

  echo "==> Saving images to ${LOCAL_TAR} ..."
  # Stage to a single compressed file so rsync can resume a partial transfer.
  docker save ritme-backend:latest ritme-frontend:latest | gzip > "$LOCAL_TAR"
  echo "    staged $(( $(wc -c < "$LOCAL_TAR") / 1024 / 1024 )) MB (compressed)"
fi

echo "==> Uploading to ${SERVER} (resumable — safe to re-run if it drops)..."
# --partial  : keep the partial file on interrupt
# --append   : on rerun, send only the bytes past what's already on the server
#              (macOS stock rsync has no --append-verify; safe here because the
#               reuse-tar logic above guarantees the source is unchanged)
# --inplace  : write straight into the target (no full-size temp copy)
# --progress : live speed / ETA / percentage
rsync --partial --append --inplace --progress \
  -e "ssh ${SSH_OPTS[*]}" \
  "$LOCAL_TAR" "${SERVER}:${REMOTE_TAR}"

echo "==> Loading images on the server..."
ssh "${SSH_OPTS[@]}" "${SERVER}" "gunzip -c '${REMOTE_TAR}' | docker load"

echo "==> Updating compose files and restarting stack..."
ssh "${SSH_OPTS[@]}" "${SERVER}" \
  "cd ${REMOTE_DIR} && docker compose up -d --no-build"
#  "cd ${REMOTE_DIR} && git pull --ff-only && docker compose up -d --no-build"

echo "==> Cleaning up staged image files..."
rm -f "$LOCAL_TAR"
ssh "${SSH_OPTS[@]}" "${SERVER}" "rm -f '${REMOTE_TAR}'" || true

# Each docker load leaves the previous image untagged on disk; without this the
# 18 GB root disk fills after a few deploys and the backend dies with ENOSPC.
echo "==> Pruning old (dangling) images on the server..."
ssh "${SSH_OPTS[@]}" "${SERVER}" "docker image prune -f" || true

echo "==> Verifying..."
sleep 5
ssh "${SSH_OPTS[@]}" "${SERVER}" "cd ${REMOTE_DIR} && docker compose ps --format '{{.Service}}: {{.Status}}'"
curl -s -o /dev/null -m 15 -w "http://ritmeapp.ir -> %{http_code}\n" http://ritmeapp.ir/ || true

echo "✅ Deploy done."
