#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Auto-deploy staging: runs ON the server every 2 minutes (systemd timer
# ritme-stage-autodeploy.timer). When the `stage` branch on GitHub has a commit
# we have not tried yet, it VERIFIES its signatures and then ships it with
# `LOCAL=1 ./deploy-stage.sh`.
#
# The chain this closes:
#   Claude Code pushes claude/<x>  →  .github/workflows/claude-to-stage.yml
#   runs the checks and merges into stage  →  this script sees the new commit,
#   verifies it, and rebuilds stage.ritmeapp.ir.
#
# Supply-chain guard (T-M12-03, decision (a) — docs/quality-2026-11/autodeploy.md):
#   * This file runs from an INSTALLED copy, /usr/local/sbin/ritme-stage-autodeploy,
#     updated only by hand (install -m 0755 …). It never runs or re-execs the copy
#     in the checkout, so a pushed commit cannot change the gate that checks it.
#   * Every commit that stage gained since the last verified commit must be
#     acceptable (verify_range below), or nothing is checked out or built:
#       - signed with an SSH key listed in /etc/ritme/allowed_signers
#         (git verify-commit, gpg.minTrustLevel=fully → an unknown key fails), or
#       - an unsigned MERGE whose tree is exactly the clean merge of its parents
#         (the merge commit claude-to-stage.yml makes — it adds no content of
#         its own; its parents are checked like every other commit), or
#       - reachable only through a non-first parent of a merge that is itself
#         signed by an allowed key (a signed merge on the Mac = "I approve this
#         branch", e.g. a branch Claude Code on the web committed unsigned).
#   * Fails closed: no allowed_signers file, a writable one, no trust anchor,
#     an unsigned/unknown-key commit, an "evil" merge, an old git → exit 1, logged.
#
# Files on the server:
#   /usr/local/sbin/ritme-stage-autodeploy  this script (installed copy)
#   /etc/ritme/allowed_signers              pinned signing keys (root:root 0644)
#   /opt/ritme-stage-src                    a git clone used as the build source
#   /var/lib/ritme-stage-autodeploy/verified  last commit that passed verification
#                                           (trust anchor; set once by hand at install)
#   /var/lib/ritme-stage-autodeploy/last    the last commit we TRIED (ok or not)
#   /var/log/ritme-stage-autodeploy.log     every run's output
#
# A refused or failed commit is NOT retried every tick. Push a new commit, or
# retry by hand:
#   rm /var/lib/ritme-stage-autodeploy/last && systemctl start ritme-stage-autodeploy
# Pause it:     systemctl stop ritme-stage-autodeploy.timer
# Self-test:    ritme-stage-autodeploy --self-test   (temp repo + temp keys, no deploy)
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/RezaDeveloper97/ritme.git}"
BRANCH="${BRANCH:-stage}"
SRC_DIR="${SRC_DIR:-/opt/ritme-stage-src}"
STATE_DIR="/var/lib/ritme-stage-autodeploy"
LOCK="/run/ritme-stage-autodeploy.lock"
# Pinned outside the checkout. Only the self-test overrides it (with a temp file).
ALLOWED_SIGNERS="/etc/ritme/allowed_signers"

log() { echo "[$(date -u +%FT%TZ)] $*"; }

# git with signature verification pinned to $ALLOWED_SIGNERS, whatever the clone's config says.
vgit() {
  git -c gpg.format=ssh \
      -c gpg.ssh.allowedSignersFile="$ALLOWED_SIGNERS" \
      -c gpg.minTrustLevel=fully \
      "$@"
}

# The allowed_signers file must exist, be non-empty, and be writable only by root (or, in the self-test, by us).
check_allowed_signers() {
  local f="$ALLOWED_SIGNERS" owner mode
  [[ -f "$f" && -s "$f" ]] || { log "REFUSED: $f is missing or empty"; return 1; }
  owner="$(stat -c '%u' "$f" 2>/dev/null || stat -f '%u' "$f")"
  mode="$(stat -c '%a' "$f" 2>/dev/null || stat -f '%Lp' "$f")"
  if [[ "$owner" != "0" && "$owner" != "$(id -u)" ]]; then
    log "REFUSED: $f is owned by uid $owner"; return 1
  fi
  if (( (8#$mode & 8#022) != 0 )); then
    log "REFUSED: $f is group/world-writable (mode $mode)"; return 1
  fi
}

# 0 when $1 carries a good signature from a key in $ALLOWED_SIGNERS.
signed_ok() {
  vgit verify-commit "$1" >/dev/null 2>&1 || return 1
  # Belt and braces: %G? must be G (good + valid), not U (good, unknown key) or anything else.
  [[ "$(vgit log -1 --format='%G?' "$1" 2>/dev/null)" == "G" ]]
}

# The tree a clean, conflict-free merge of the given parents produces (empty output on conflict/error).
clean_merge_tree() {
  local p1="$1" p2="$2"
  git merge-tree --write-tree --no-messages "$p1" "$p2" 2>/dev/null | head -n1
}

# 0 when merge commit $1 (exactly two parents) is unsigned-but-harmless: its tree is the clean merge of its parents.
clean_merge_ok() {
  local c="$1" parents want have
  read -r -a parents <<<"$(git rev-list --parents -n1 "$c" | cut -d' ' -f2-)"
  (( ${#parents[@]} == 2 )) || return 1
  want="$(clean_merge_tree "${parents[0]}" "${parents[1]}")" || return 1
  have="$(git rev-parse "$c^{tree}")"
  [[ -n "$want" && "$want" == "$have" ]]
}

# verify_range ANCHOR NEW — every commit reachable from NEW but not from ANCHOR must be acceptable (see header).
# Prints one line per refused commit; returns 0 only when all are acceptable.
verify_range() {
  local anchor="$1" new="$2" c p v first rest bad=0
  local vouched=$'\n'   # newline-separated set (bash 3.2-safe; no associative arrays)
  git cat-file -e "$anchor^{commit}" 2>/dev/null || { log "REFUSED: trust anchor ${anchor:0:12} is not in the clone"; return 1; }
  git cat-file -e "$new^{commit}" 2>/dev/null || { log "REFUSED: ${new:0:12} is not a commit"; return 1; }

  local commits
  commits="$(git rev-list "$anchor..$new")" || return 1
  [[ -z "$commits" ]] && return 0

  # Pass 1: signed merges vouch for what only their non-first parents bring in.
  for c in $commits; do
    read -r first rest <<<"$(git rev-list --parents -n1 "$c" | cut -d' ' -f2-)"
    [[ -n "${rest:-}" ]] || continue
    signed_ok "$c" || continue
    for p in $rest; do
      while read -r v; do [[ -n "$v" ]] && vouched+="$v"$'\n'; done < <(git rev-list "$p" "^$first" "^$anchor")
    done
  done

  # Pass 2: every commit is signed, vouched for, or a clean unsigned merge.
  for c in $commits; do
    if signed_ok "$c"; then continue; fi
    if [[ "$vouched" == *$'\n'"$c"$'\n'* ]]; then continue; fi
    if [[ "$(git rev-list --parents -n1 "$c" | wc -w)" -ge 3 ]] && clean_merge_ok "$c"; then continue; fi
    log "REFUSED: ${c:0:12} $(git log -1 --format='%G? %an <%ae>: %s' "$c" 2>/dev/null) — not signed by an allowed key"
    bad=1
  done
  return "$bad"
}

preflight() {
  # SSH signing needs git ≥ 2.34; merge-tree --write-tree needs git ≥ 2.38.
  local v; v="$(git version | awk '{print $3}')"
  if ! printf '2.38\n%s\n' "$v" | sort -V -C; then
    log "REFUSED: git $v is too old (need ≥ 2.38 for SSH signature + merge checks)"; return 1
  fi
  command -v ssh-keygen >/dev/null || { log "REFUSED: ssh-keygen not found"; return 1; }
  check_allowed_signers
}

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
  # Record the attempt first, so neither a refusal nor a failure is retried on the next tick.
  echo "$new" > "$STATE_DIR/last"

  preflight || { log "❌ ${new:0:8} not deployed (preflight)"; exit 1; }
  anchor="$(cat "$STATE_DIR/verified" 2>/dev/null || true)"
  if [[ -z "$anchor" ]]; then
    log "REFUSED: no trust anchor in $STATE_DIR/verified — set it once by hand (docs/quality-2026-11/autodeploy.md)"
    log "❌ ${new:0:8} not deployed"; exit 1
  fi
  if ! verify_range "$anchor" "$new"; then
    log "❌ ${new:0:8} not deployed: unverified commits since ${anchor:0:8} (sign them, or approve with a signed merge)"
    exit 1
  fi
  log "🔏 ${anchor:0:8}..${new:0:8} verified"
  echo "$new" > "$STATE_DIR/verified"

  # Check the verified commit out; deploy-stage.sh is now content we just verified.
  git checkout --quiet --force --detach "$new"
  git clean -fdq

  # Build the verified SHA, never the ref: a fetch during the build must not swap in an unverified tip.
  if LOCAL=1 BRANCH="$new" ./deploy-stage.sh; then
    log "✅ deployed ${new:0:8}"
    echo "$new" > "$STATE_DIR/ok"
  else
    rc=$?
    log "❌ deploy of ${new:0:8} failed (exit $rc) — not retried until stage moves again"
    exit "$rc"
  fi
}

# ── self-test: temp repo, temp keys; proves accept/refuse without touching the server ──
self_test() {
  local tmp fails=0
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' RETURN
  ssh-keygen -q -t ed25519 -N '' -C ritme-test-allowed -f "$tmp/allowed" >/dev/null
  ssh-keygen -q -t ed25519 -N '' -C ritme-test-stranger -f "$tmp/stranger" >/dev/null
  printf 'dev@ritme.test namespaces="git" %s\n' "$(cat "$tmp/allowed.pub")" > "$tmp/allowed_signers"
  chmod 0644 "$tmp/allowed_signers"
  ALLOWED_SIGNERS="$tmp/allowed_signers"

  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
  git init -q -b stage "$tmp/r"; cd "$tmp/r"
  git config user.name t; git config user.email dev@ritme.test; git config gpg.format ssh
  cm()  { echo "$1" > "$1.txt"; git add -A; git commit -q -m "$1" "${@:2}"; git rev-parse HEAD; }
  ok()  { git commit --allow-empty -q -m "$1" -S"$tmp/allowed"; git rev-parse HEAD; }

  expect() { # expect accept|refuse NAME ANCHOR NEW
    local want="$1" name="$2" got
    if verify_range "$3" "$4" >/dev/null 2>&1; then got=accept; else got=refuse; fi
    if [[ "$got" == "$want" ]]; then echo "  ok    $name → $got"; else echo "  FAIL  $name → $got (want $want)"; fails=$((fails+1)); fi
  }

  local base s u x m e
  base="$(cm base)"                                       # anchor (deployed, trusted)
  s="$(cm signed -S"$tmp/allowed")"
  expect accept "commit signed by the allowed key"        "$base" "$s"
  u="$(cm unsigned)"
  expect refuse "unsigned commit on top"                  "$base" "$u"
  expect refuse "unsigned commit, anchor = parent"        "$s"    "$u"
  git reset -q --hard "$s"
  x="$(cm stranger -S"$tmp/stranger")"
  expect refuse "commit signed by an unknown key"         "$s"    "$x"
  git reset -q --hard "$s"
  expect accept "nothing new (anchor = tip)"              "$s"    "$s"

  # CI-style unsigned --no-ff merge of a signed branch into stage.
  git checkout -q -b feat "$s"; cm feat1 -S"$tmp/allowed" >/dev/null; cm feat2 -S"$tmp/allowed" >/dev/null
  git checkout -q stage; cm stagework -S"$tmp/allowed" >/dev/null
  git merge -q --no-ff --no-edit feat; m="$(git rev-parse HEAD)"
  expect accept "unsigned clean merge of signed commits"  "$s"    "$m"

  # Evil merge: same parents, extra content smuggled into the merge commit.
  git reset -q --hard HEAD^; git merge -q --no-ff --no-commit feat >/dev/null 2>&1; echo pwn > pwn.txt; git add pwn.txt
  git commit -q -m "Merge feat (auto)"; e="$(git rev-parse HEAD)"
  expect refuse "unsigned merge with extra content"       "$s"    "$e"

  # Clean unsigned merge of a branch that has an unsigned commit.
  git reset -q --hard HEAD^
  git checkout -q -b web "$s"; cm web1 >/dev/null
  git checkout -q stage; git merge -q --no-ff --no-edit web; m="$(git rev-parse HEAD)"
  expect refuse "clean merge bringing an unsigned commit" "$s"    "$m"

  # Signed merge approves the unsigned branch (the Mac-side approval of a web-made branch).
  git reset -q --hard HEAD^; git merge -q --no-ff --no-edit -S"$tmp/allowed" web; m="$(git rev-parse HEAD)"
  expect accept "signed merge vouching an unsigned branch" "$s"   "$m"
  # …but a signed merge does not vouch for unsigned commits on its first-parent line.
  git reset -q --hard HEAD^; cm stage-unsigned >/dev/null; git merge -q --no-ff --no-edit -S"$tmp/allowed" feat
  expect refuse "signed merge over an unsigned stage commit" "$s"  "$(git rev-parse HEAD)"

  expect refuse "unknown trust anchor"                    "0000000000000000000000000000000000000000" "$s"

  chmod 0666 "$ALLOWED_SIGNERS"
  if check_allowed_signers >/dev/null 2>&1; then echo "  FAIL  world-writable allowed_signers → accept"; fails=$((fails+1))
  else echo "  ok    world-writable allowed_signers → refuse"; fi
  ALLOWED_SIGNERS="$tmp/missing"
  if check_allowed_signers >/dev/null 2>&1; then echo "  FAIL  missing allowed_signers → accept"; fails=$((fails+1))
  else echo "  ok    missing allowed_signers → refuse"; fi
  expect refuse "signed commit, no allowed_signers file"  "$base" "$s"

  cd /
  if (( fails )); then echo "self-test: $fails FAILED"; return 1; fi
  echo "self-test: all passed"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  case "${1:-}" in
    --self-test) self_test ;;
    "") main ;;
    *) echo "usage: $0 [--self-test]" >&2; exit 2 ;;
  esac
fi
