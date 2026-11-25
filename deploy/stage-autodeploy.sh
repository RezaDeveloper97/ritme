#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Auto-deploy staging: runs ON the server every 2 minutes (systemd timer
# ritme-stage-autodeploy.timer). When the `stage` branch on GitHub has a commit
# we have not tried yet, it ships it with `LOCAL=1 ./deploy-stage.sh`.
#
# The chain this closes:
#   a branch is pushed  →  .github/workflows/claude-to-stage.yml runs the checks
#   and merges it into stage  →  this script sees the new commit and rebuilds
#   stage.ritmeapp.ir.
#
# Trust (user decision 2026-11-25, replaces the signed-commit gate of T-M12-03):
#   whatever is on `stage` on GitHub is deployed. The gate is GitHub itself —
#   push access to the repository plus the workflow's checks. There is no
#   signature check here any more, so anyone who can push to the repository can
#   run code as root on this host; keep push access and tokens tight.
#   docs/quality-2026-11/autodeploy.md has the history and how to bring the
#   signature gate back (git history of this file: the version before 2026-11-25).
#
# This file runs from an INSTALLED copy, /usr/local/sbin/ritme-stage-autodeploy,
# so a half-written checkout can never be the thing systemd executes. Update it
# by hand after changing it:
#   install -o root -g root -m 0755 deploy/stage-autodeploy.sh /usr/local/sbin/ritme-stage-autodeploy
#
# Files on the server:
#   /usr/local/sbin/ritme-stage-autodeploy  this script (installed copy)
#   /opt/ritme-stage-src                    a git clone used as the build source
#   /var/lib/ritme-stage-autodeploy/last    the last commit we TRIED (ok or not)
#   /var/lib/ritme-stage-autodeploy/ok      the last commit that deployed
#   /var/log/ritme-stage-autodeploy.log     every run's output
#
# A failed commit is NOT retried every tick. Push a new commit, or retry by hand:
#   rm /var/lib/ritme-stage-autodeploy/last && systemctl start ritme-stage-autodeploy
# Pause it:     systemctl stop ritme-stage-autodeploy.timer
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/RezaDeveloper97/ritme.git}"
BRANCH="${BRANCH:-stage}"
SRC_DIR="${SRC_DIR:-/opt/ritme-stage-src}"
STATE_DIR="/var/lib/ritme-stage-autodeploy"
LOCK="/run/ritme-stage-autodeploy.lock"

log() { echo "[$(date -u +%FT%TZ)] $*"; }

main() {
  # One run at a time; a tick that finds a deploy in progress just leaves.
  exec 9>"$LOCK"
  flock -n 9 || { log "a deploy is already running — skipping this tick"; exit 0; }

  mkdir -p "$STATE_DIR"
  if [[ ! -d "$SRC_DIR/.git" ]]; then
    log "cloning $REPO_URL into $SRC_DIR"
    git clone --quiet --no-checkout --branch "$BRANCH" "$REPO_URL" "$SRC_DIR"
  fi

  cd "$SRC_DIR"
  timeout 60 git fetch --quiet --prune origin "+refs/heads/${BRANCH}:refs/remotes/origin/${BRANCH}"
  new="$(git rev-parse "origin/${BRANCH}")"
  last="$(cat "$STATE_DIR/last" 2>/dev/null || true)"
  [[ "$new" == "$last" ]] && exit 0

  log "stage moved ${last:0:8} → ${new:0:8}: $(git log -1 --format='%s' "$new")"
  # Record the attempt first, so a failure is not retried on the next tick.
  echo "$new" > "$STATE_DIR/last"

  git checkout --quiet --force --detach "$new"
  git clean -fdq

  # Build the SHA we just checked out, never the ref: a fetch during the build must not swap the tip.
  # SERIAL_BUILD=1: one image at a time — parallel `next build`s swap this 2-core / 4 GB machine to a halt.
  started="$(date +%s)"
  if LOCAL=1 SERIAL_BUILD=1 BRANCH="$new" ./deploy-stage.sh; then
    log "✅ deployed ${new:0:8} in $(( $(date +%s) - started ))s"
    echo "$new" > "$STATE_DIR/ok"
  else
    rc=$?
    log "❌ deploy of ${new:0:8} failed after $(( $(date +%s) - started ))s (exit $rc) — not retried until stage moves again"
    exit "$rc"
  fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  case "${1:-}" in
    "") main ;;
    *) echo "usage: $0" >&2; exit 2 ;;
  esac
fi
