#!/usr/bin/env bash
# (Re)start the local stack used by bloom tasks, detached: backend-go API on :8020 (DB ritme_dev, SMS log) and the
# Next dev server on :3000 (distDir .next-dev so `npm run build` can run alongside). Usage: dev-up.sh [api|web|all]
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"; LOG="${TMPDIR:-/tmp}/ritme-dev"; mkdir -p "$LOG"; what="${1:-all}"
if [ "$what" != web ]; then
  lsof -ti tcp:8020 | xargs kill 2>/dev/null || true
  (cd "$ROOT/backend-go" && DB_DATABASE=ritme_dev SMS_PROVIDER=log nohup make run >"$LOG/api.log" 2>&1 &)
  until curl -s 127.0.0.1:8020/up >/dev/null; do sleep 2; done; echo "api up (log $LOG/api.log)"
fi
if [ "$what" != api ]; then
  pkill -f "next dev" 2>/dev/null || true
  (cd "$ROOT/frontend" && NEXT_DIST_DIR=.next-dev nohup npm run dev >"$LOG/web.log" 2>&1 &)
  until curl -s -o /dev/null localhost:3000/fa/splash; do sleep 2; done; echo "web up (log $LOG/web.log)"
fi
