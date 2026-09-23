#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Flip ONE strangler route group between Laravel and Go in ONE environment.
#
#   deploy/switch-go-route.sh <stage|prod> <group> <on|off> [--dry-run|--local]
#   deploy/switch-go-route.sh --status [stage|prod] [--local]
#
#   on        route the group to Go       (stage-backend-go:80 / ritme-backend-go-1:80)
#   off       route the group to Laravel  (stage-backend:80    / ritme-backend-1:80)
#   --status  print the current group -> backend map (both envs if none given)
#   --dry-run show what would change, computed on the REPO copy; no ssh at all
#   --local   rewrite the REPO copy only (makes a flip permanent: the next
#             deploy rsyncs it). No ssh.
#
# The route map itself is deploy/go-routes.inc (shared by both vhosts). What a
# group points at is one line per group in the environment's own config:
#
#   stage  -> STAGE_CONF (deploy/stage-ssl.conf)  upstream stage_route_<group> { server …; }
#   prod   -> PROXY_CONF (deploy/proxy-ssl.conf)  upstream prod_route_<group>  { server …; }
#
# Live mode (default) runs against the shared proxy on the server:
#   1. reads the env's LIVE config file (the one .env's STAGE_CONF/PROXY_CONF
#      mounts into the proxy) and rewrites that single upstream line locally;
#   2. on the server: backs the file up, rewrites it IN PLACE (same inode — it
#      is a single-file bind mount; a new inode would be invisible to the
#      container), checks the container sees the new line, `nginx -t`, reload;
#   3. if anything in step 2 fails the backup is restored and `nginx -t` re-run
#      — production's vhosts share this nginx, so a bad file must never stay.
# `nginx -s reload` is graceful: in-flight requests finish on the old workers,
# so a flip drops nothing.
#
# The other environment's file and upstream lines are never read for writing:
# the env picks the file AND the upstream-name prefix, and the rewrite refuses
# to change anything but exactly one `<env>_route_<group>` line.
#
# A live flip is lost on the next deploy (rsync ships the repo copy); make it
# permanent with --local and commit — see docs/go-migration/cutover.md.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail
cd "$(dirname "$0")/.."

SERVER="${SERVER:-root@89.251.8.115}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_ed25519}"
PROD_DIR="${PROD_DIR:-/opt/ritme}"   # the shared proxy and its config live here
PROD_COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

# Must match the locations in deploy/go-routes.inc.
ROUTE_GROUPS=(content reminders healthlog profile pregnancy cycle messages home auth)

usage() {
  sed -n '4,13p' "$0" | sed 's/^# \{0,1\}//' >&2
  exit 2
}
die() { echo "!! $*" >&2; exit 1; }

# ── per-environment facts ───────────────────────────────────────────────────
laravel_target() { case "$1" in stage) echo "stage-backend:80" ;; prod) echo "ritme-backend-1:80" ;; esac; }
go_target()      { case "$1" in stage) echo "stage-backend-go:80" ;; prod) echo "ritme-backend-go-1:80" ;; esac; }
repo_conf()      { case "$1" in stage) echo "deploy/stage-ssl.conf" ;; prod) echo "deploy/proxy-ssl.conf" ;; esac; }
# .env key that selects the live file, its default, and where the proxy mounts it.
env_key()        { case "$1" in stage) echo "STAGE_CONF" ;; prod) echo "PROXY_CONF" ;; esac; }
env_default()    { case "$1" in stage) echo "./deploy/stage-off.conf" ;; prod) echo "./deploy/proxy.conf" ;; esac; }
mount_path()     { case "$1" in stage) echo "/etc/nginx/conf.d/zz-stage.conf" ;; prod) echo "/etc/nginx/conf.d/default.conf" ;; esac; }

# ── pure text helpers (shared by live, --dry-run and --local) ───────────────
# print_map <env>  (config on stdin)  ->  "group  backend  target" lines
print_map() {
  local env="$1" conf g line target who
  conf="$(cat)"
  for g in "${ROUTE_GROUPS[@]}"; do
    line="$(grep -E "^[[:space:]]*upstream[[:space:]]+${env}_route_${g}[[:space:]]*\{" <<<"$conf" || true)"
    target="$(sed -nE 's/.*server[[:space:]]+([^;[:space:]]+);.*/\1/p' <<<"$line")"
    if   [[ -z "$line" ]];                          then who="MISSING"; target="-"
    elif [[ "$target" == "$(go_target "$env")" ]];      then who="go"
    elif [[ "$target" == "$(laravel_target "$env")" ]]; then who="laravel"
    else who="UNKNOWN"; fi
    printf '  %-6s %-10s %-8s %s\n' "$env" "$g" "$who" "$target"
  done
}

# rewrite <env> <group> <target>  (config on stdin -> new config on stdout)
# Refuses unless exactly one line is that group's upstream.
rewrite() {
  local env="$1" group="$2" target="$3" conf n
  conf="$(cat)"
  n="$(grep -cE "^[[:space:]]*upstream[[:space:]]+${env}_route_${group}[[:space:]]*\{[[:space:]]*server[[:space:]]+[^;[:space:]]+;[[:space:]]*\}" <<<"$conf" || true)"
  [[ "$n" == "1" ]] || die "expected exactly 1 'upstream ${env}_route_${group} { server …; }' line, found ${n}"
  sed -E "s/^([[:space:]]*upstream[[:space:]]+${env}_route_${group}[[:space:]]*\{[[:space:]]*server[[:space:]]+)[^;[:space:]]+;/\1${target};/" <<<"$conf"
}

ssh_run() { ssh -i "$SSH_KEY" -o ConnectTimeout=20 "$SERVER" "$@"; }

# live_conf_path <env>  ->  path of the live file relative to PROD_DIR (--status)
live_conf_path() {
  local env="$1" key def
  key="$(env_key "$env")"; def="$(env_default "$env")"
  ssh_run "cd ${PROD_DIR} && v=\$(grep -E '^${key}=' .env | tail -n1 | cut -d= -f2-); echo \"\${v:-${def}}\""
}

# ── argument parsing ────────────────────────────────────────────────────────
MODE=live; STATUS=0; POS=()
for a in "$@"; do
  case "$a" in
    --dry-run) MODE=dry ;;
    --local)   MODE=local ;;
    --status)  STATUS=1 ;;
    -h|--help) usage ;;
    --*)       die "unknown flag $a" ;;
    *)         POS+=("$a") ;;
  esac
done

valid_env()   { [[ "$1" == stage || "$1" == prod ]]; }
valid_group() { local g; for g in "${ROUTE_GROUPS[@]}"; do [[ "$g" == "$1" ]] && return 0; done; return 1; }

# ── --status ────────────────────────────────────────────────────────────────
if (( STATUS )); then
  envs=(stage prod)
  if (( ${#POS[@]} > 0 )); then valid_env "${POS[0]}" || usage; envs=("${POS[0]}"); fi
  for env in "${envs[@]}"; do
    if [[ "$MODE" == live ]]; then
      path="$(live_conf_path "$env")"
      echo "${env}: live ${PROD_DIR}/${path#./}"
      ssh_run "cat ${PROD_DIR}/${path#./}" | print_map "$env"
    else
      echo "${env}: repo $(repo_conf "$env")"
      print_map "$env" < "$(repo_conf "$env")"
    fi
  done
  exit 0
fi

(( ${#POS[@]} == 3 )) || usage
ENV_="${POS[0]}"; GROUP="${POS[1]}"; STATE="${POS[2]}"
valid_env "$ENV_"     || die "environment must be stage or prod, got '${ENV_}'"
valid_group "$GROUP"  || die "unknown group '${GROUP}' (groups: ${ROUTE_GROUPS[*]})"
case "$STATE" in
  on)  TARGET="$(go_target "$ENV_")" ;;
  off) TARGET="$(laravel_target "$ENV_")" ;;
  *)   die "state must be on or off, got '${STATE}'" ;;
esac

# ── --dry-run / --local: repo copy only ─────────────────────────────────────
if [[ "$MODE" != live ]]; then
  file="$(repo_conf "$ENV_")"
  new="$(rewrite "$ENV_" "$GROUP" "$TARGET" < "$file")"
  echo "==> ${ENV_} ${GROUP} -> ${STATE} ($TARGET) in ${file}"
  if diff -u --label "${file} (current)" --label "${file} (after)" "$file" <(printf '%s\n' "$new"); then
    echo "  (no change: ${GROUP} already ${STATE})"
  fi
  if [[ "$MODE" == local ]]; then
    printf '%s\n' "$new" > "$file"
    echo "==> wrote ${file}"
  else
    echo "==> dry run: nothing written, nothing sent to ${SERVER}"
  fi
  echo "${ENV_} map after:"
  printf '%s\n' "$new" | print_map "$ENV_"
  exit 0
fi

# ── live ────────────────────────────────────────────────────────────────────
SECONDS=0
# One round trip: the live file's path (line 1) and its content (the rest).
key="$(env_key "$ENV_")"; def="$(env_default "$ENV_")"
live="$(ssh_run "cd ${PROD_DIR} && v=\$(grep -E '^${key}=' .env | tail -n1 | cut -d= -f2-); p=\${v:-${def}}; echo \"\$p\"; cat \"\$p\"")"
path="$(head -n1 <<<"$live")"; path="${path#./}"
current="$(tail -n +2 <<<"$live")"
[[ "$path" == deploy/* && "$path" != *..* ]] || die "unexpected live config path '${path}'"
grep -qE "upstream[[:space:]]+${ENV_}_route_" <<<"$current" \
  || die "live ${path} has no ${ENV_}_route_* upstreams — deploy the T-M2-09 proxy config first"
new="$(rewrite "$ENV_" "$GROUP" "$TARGET" <<<"$current")"
if [[ "$new" == "$current" ]]; then
  echo "==> ${ENV_} ${GROUP} already ${STATE} (${TARGET}); nothing to do."
  printf '%s\n' "$current" | print_map "$ENV_"
  exit 0
fi
changed="$(diff <(printf '%s\n' "$current") <(printf '%s\n' "$new") | grep -c '^>' || true)"
[[ "$changed" == "1" ]] || die "refusing: rewrite would change ${changed} lines, expected 1"

echo "==> ${ENV_}: ${GROUP} -> ${STATE} (${TARGET}) in ${PROD_DIR}/${path}"
want="upstream ${ENV_}_route_${GROUP}"
# New content goes over stdin; the rest runs on the server in ONE session and
# ONE container exec (check the container sees the line, nginx -t, reload), so
# the flip itself takes about a second. Any failure restores the backup.
printf '%s\n' "$new" | ssh_run "set -u
cd ${PROD_DIR}
F='${path}'
C='${PROD_COMPOSE}'
BK=/root/\$(basename \"\$F\").bak-\$(date +%Y%m%d-%H%M%S)
cp -p \"\$F\" \"\$BK\" || { echo '!! backup failed — nothing changed' >&2; exit 1; }
cat > \"\$F\"                      # in place: keeps the bind-mounted inode
\$C exec -T proxy sh -c \"grep -qE '${want}[[:space:]]*\\{[[:space:]]*server[[:space:]]+${TARGET};' '$(mount_path "$ENV_")' || exit 10; nginx -t 2>&1 || exit 11; nginx -s reload 2>&1 || exit 12\"
rc=\$?
if [ \$rc -ne 0 ]; then
  case \$rc in
    10) why='the proxy container does not see the new line (stale bind mount? run: up -d proxy)' ;;
    11) why='nginx -t failed (is the target container running?)' ;;
    *)  why=\"nginx reload failed (rc \$rc)\" ;;
  esac
  echo \"!! \$why -> restoring \$BK\" >&2
  cat \"\$BK\" > \"\$F\"
  \$C exec -T proxy nginx -t >&2 || echo '!! nginx -t STILL failing after restore — investigate NOW' >&2
  exit 1
fi
echo \"backup: \$BK\""
echo "==> reloaded in ${SECONDS}s (graceful; new connections use it within ~1s). ${ENV_} map now:"
printf '%s\n' "$new" | print_map "$ENV_"
