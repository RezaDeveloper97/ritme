#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Auto-deploy staging: runs ON the server every 2 minutes (systemd timer
# ritme-stage-autodeploy.timer). When the `stage` branch on GitHub has a commit
# we have not tried yet, it ships it with `LOCAL=1 ./deploy-stage.sh`.
#
# The chain this closes:
#   Claude Code (web) pushes claude/<x>  →  .github/workflows/claude-to-stage.yml
#   runs the checks and merges into stage  →  this script sees the new commit
#   and rebuilds stage.ritmeapp.ir.
#
# Pull, not push: GitHub's runners cannot be relied on to reach this server,
# but the server reaches github.com fine, and nothing listens for inbound calls.
#
# Files on the server:
#   /opt/ritme-stage-src                    a git clone used as the build source
#   /var/lib/ritme-stage-autodeploy/last    the last commit we TRIED (ok or not)
#   /var/log/ritme-stage-autodeploy.log     every run's output
#
# A failed deploy is NOT retried every tick (a broken commit would rebuild the
# images on the production machine forever). Push a new commit, or retry by hand:
#   rm /var/lib/ritme-stage-autodeploy/last && systemctl start ritme-stage-autodeploy
# Pause it:   systemctl stop ritme-stage-autodeploy.timer
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/RezaDeveloper97/ritme.git}"
BRANCH="${BRANCH:-stage}"
SRC_DIR="${SRC_DIR:-/opt/ritme-stage-src}"
STATE_DIR="/var/lib/ritme-stage-autodeploy"
LOCK="/run/ritme-stage-autodeploy.lock"

log() { echo "[$(date -u +%FT%TZ)] $*"; }

# One run at a time; a tick that finds a deploy in progress just leaves.
exec 9>"$LOCK"
flock -n 9 || { log "a deploy is already running — skipping this tick"; exit 0; }

mkdir -p "$STATE_DIR"
if [[ ! -d "$SRC_DIR/.git" ]]; then
  log "cloning $REPO_URL into $SRC_DIR"
  git clone --quiet --branch "$BRANCH" "$REPO_URL" "$SRC_DIR"
fi

cd "$SRC_DIR"
timeout 60 git fetch --quiet --prune origin "+refs/heads/${BRANCH}:refs/remotes/origin/${BRANCH}"
new="$(git rev-parse "origin/${BRANCH}")"
last="$(cat "$STATE_DIR/last" 2>/dev/null || true)"
[[ "$new" == "$last" ]] && exit 0

log "stage moved ${last:0:8} → ${new:0:8}: $(git log -1 --format='%s' "$new")"
# Record the attempt first, so a failure is not retried on the next tick.
echo "$new" > "$STATE_DIR/last"

# Check the commit out so deploy-stage.sh itself is the version being shipped.
git checkout --quiet --force --detach "$new"
git clean -fdq

if LOCAL=1 BRANCH="origin/${BRANCH}" ./deploy-stage.sh; then
  log "✅ deployed ${new:0:8}"
  echo "$new" > "$STATE_DIR/ok"
else
  rc=$?
  log "❌ deploy of ${new:0:8} failed (exit $rc) — not retried until stage moves again"
  exit "$rc"
fi
