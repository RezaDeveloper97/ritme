#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Deploy ritme to the production server (ritmeapp.ir / 62.60.198.240).
#
# The server has no access to GitHub/Packagist/npm (Iranian network filtering),
# so images CANNOT be built there. This script builds them locally for
# linux/amd64, streams them to the server with docker save/load, then restarts
# the stack. Run from the repo root:  ./deploy.sh
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail
cd "$(dirname "$0")"

SERVER="root@62.60.198.240"
SSH_KEY="$HOME/.ssh/id_ed25519"
API_BASE_URL="http://ritmeapp.ir/api/v1"
OTP_TEST_MODE="false"

echo "==> Building backend (linux/amd64)…"
docker build --platform linux/amd64 -t ritme-backend:latest ./backend

echo "==> Building frontend (linux/amd64)…"
docker build --platform linux/amd64 \
  --build-arg NEXT_PUBLIC_API_BASE_URL="$API_BASE_URL" \
  --build-arg NEXT_PUBLIC_OTP_TEST_MODE="$OTP_TEST_MODE" \
  -t ritme-frontend:latest ./frontend

echo "==> Transferring images to $SERVER…"
docker save ritme-backend:latest ritme-frontend:latest \
  | gzip \
  | ssh -i "$SSH_KEY" "$SERVER" 'gunzip | docker load'

echo "==> Updating compose files and restarting stack…"
ssh -i "$SSH_KEY" "$SERVER" \
  'cd /opt/ritme && git pull --ff-only && docker compose up -d --no-build'

echo "==> Verifying…"
sleep 5
ssh -i "$SSH_KEY" "$SERVER" 'cd /opt/ritme && docker compose ps --format "{{.Service}}: {{.Status}}"'
curl -s -o /dev/null -m 15 -w "http://ritmeapp.ir -> %{http_code}\n" http://ritmeapp.ir/ || true

echo "✅ Deploy done."
