#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Deploy the Ritme marketing site (laravel-site) to https://ritme.app (+ www.ritme.app; the old beta hostname
# betatest.ritme.app stays served and is 301ed to ritme.app once the main domain answers publicly)
# on the PRODUCTION server (root@89.251.8.115), as its own compose project.
#
#     cd laravel-site && ./deploy-betatest.sh
#
#   source   /opt/ritme-site              (prod app: /opt/ritme, staging: /opt/ritme-stage)
#   project  ritme-site                   containers ritme-site-app-1 / -scheduler-1 / -db-1
#   runtime  Apache + PHP 8.2 + MariaDB 11.4 (cPanel-like), .htaccess honoured
#   door     production's nginx proxy only (no host ports) via the `ritme-edge` network
#
# Steps:
#   1. build the release package locally (deploy/build-cpanel.sh: no-dev vendor, Vite build, service worker,
#      critical CSS — needs php, composer, node, Chrome here, nothing of that on the server)
#   2. ship package + docker files to /opt/ritme-site, create the server .env once (generated secrets)
#   3. hook ritme.app / www / betatest into production's proxy (validated with a throw-away `nginx -t` before the
#      real proxy is touched; only recreated when its mounts change); keep APP_ENV/APP_URL on the main domain and
#      turn on the host redirect (APP_CANONICAL_REDIRECT) once https://ritme.app is reachable through the CDN
#   4. build the image on the server, `up -d`, then RUN THE MIGRATIONS: `php artisan app:upgrade` (maintenance
#      mode → migrate --force → insert-missing seeders → caches → cache namespaces bumped → up)
#   5. first deploy only: FAQ + demo seeders (SEED_DEMO=1) and a super-admin with a generated password
#      (written to /root/ritme-site-credentials.txt on the server)
#   6. HTTPS certificate for the origin (once), smoke tests (site + production hosts untouched)
#
# Switches:
#   (default)                              package the working tree as it is now (uncommitted changes included)
#   FROM_WORKTREE=0 ./deploy-betatest.sh   package the last commit (HEAD) instead
#   SKIP_BUILD=1    ./deploy-betatest.sh   reuse the newest dist/*.zip
#   NO_PROXY=1      ./deploy-betatest.sh   skip the proxy step (site-ssl.conf unchanged)
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail
cd "$(dirname "$0")"
SITE_DIR="$(pwd)"
REPO_DIR="$(cd .. && pwd)"

SERVER="${SERVER:-root@89.251.8.115}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_ed25519}"
REMOTE_DIR="/opt/ritme-site"
PROD_DIR="/opt/ritme"
DOMAIN="${DOMAIN:-ritme.app}"
WWW_DOMAIN="${WWW_DOMAIN:-www.ritme.app}"
LEGACY_DOMAIN="${LEGACY_DOMAIN:-betatest.ritme.app}"
SSH_OPTS=(-i "$SSH_KEY" -o ConnectTimeout=20 -o ServerAliveInterval=15 -o ServerAliveCountMax=8)
SITE_COMPOSE="docker compose -p ritme-site -f ${REMOTE_DIR}/docker-compose.yml --project-directory ${REMOTE_DIR}"
PROD_COMPOSE="docker compose -f ${PROD_DIR}/docker-compose.yml -f ${PROD_DIR}/docker-compose.prod.yml --project-directory ${PROD_DIR}"

ssh_run() { ssh "${SSH_OPTS[@]}" "$SERVER" "$@"; }
step() { printf '\n\033[1;35m==> %s\033[0m\n' "$*"; }
die() { printf '\033[1;31m!! %s\033[0m\n' "$*" >&2; exit 1; }

# ── 1. Release package ──────────────────────────────────────────────────────
if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
  # Default: ship exactly what is on disk now (uncommitted edits included) — the latest version is what the user
  # expects to see after a deploy. FROM_WORKTREE=0 packages the last commit (HEAD) instead.
  if [[ "${FROM_WORKTREE:-1}" == "1" ]]; then
    step "Building the release package from the WORKING TREE (latest local version)"
    bash deploy/build-cpanel.sh --layout=docroot --worktree
  else
    [[ -z "$(git status --porcelain -- . 2>/dev/null)" ]] \
      || echo "   (note) FROM_WORKTREE=0: uncommitted changes in laravel-site are NOT shipped — HEAD is packaged"
    step "Building the release package from HEAD ($(git rev-parse --short HEAD))"
    bash deploy/build-cpanel.sh --layout=docroot
  fi
fi
PACKAGE="$(ls -t dist/ritme-site-*.zip 2>/dev/null | grep -v -- '-public_html' | head -1 || true)"
[[ -n "$PACKAGE" ]] || die "no release package in dist/ — run without SKIP_BUILD=1"
echo "   package: $PACKAGE ($(du -h "$PACKAGE" | cut -f1))"

# ── 2. Ship ─────────────────────────────────────────────────────────────────
step "Shipping package + docker files to ${SERVER}:${REMOTE_DIR}"
ssh_run "mkdir -p ${REMOTE_DIR}/deploy/betatest ${REMOTE_DIR}/incoming"
# The ~20 MB zip: no -z (already compressed), progress shown, resumable (--partial) and a 60 s stall timeout with
# retries — slow or flaky uploads used to hang here silently while interactive ssh still worked.
shipped=0
for attempt in 1 2 3 4 5; do
  if rsync -a --partial --timeout=60 --info=progress2 -e "ssh ${SSH_OPTS[*]}" "$PACKAGE" "${SERVER}:${REMOTE_DIR}/incoming/release.zip"; then
    shipped=1; break
  fi
  echo "   upload stalled or failed (attempt ${attempt}/5) — resuming in 5 s…"; sleep 5
done
[[ "$shipped" == "1" ]] || die "could not upload ${PACKAGE} — check the connection (try another network / VPN) and re-run with SKIP_BUILD=1"
rsync -az --delete -e "ssh ${SSH_OPTS[*]}" deploy/betatest/ "${SERVER}:${REMOTE_DIR}/deploy/betatest/"
rsync -az -e "ssh ${SSH_OPTS[*]}" deploy/betatest/docker-compose.yml "${SERVER}:${REMOTE_DIR}/docker-compose.yml"
ssh_run "set -e; command -v unzip >/dev/null || (apt-get update -qq && apt-get install -y -qq unzip);
  rm -rf ${REMOTE_DIR}/release.new && mkdir -p ${REMOTE_DIR}/release.new
  unzip -q ${REMOTE_DIR}/incoming/release.zip -d ${REMOTE_DIR}/release.new
  rm -rf ${REMOTE_DIR}/release && mv ${REMOTE_DIR}/release.new ${REMOTE_DIR}/release"

# Server-only .env, created once with generated secrets; never overwritten.
ssh_run "test -f ${REMOTE_DIR}/.env" || {
  step "Creating ${REMOTE_DIR}/.env (first deploy)"
  ssh_run "set -e; cd ${REMOTE_DIR}
    key=\"base64:\$(openssl rand -base64 32)\"; dbp=\"\$(openssl rand -hex 24)\"; rootp=\"\$(openssl rand -hex 24)\"
    sed -e \"s|^APP_KEY=.*|APP_KEY=\${key}|\" -e \"s|^DB_PASSWORD=.*|DB_PASSWORD=\${dbp}|\" \
        -e \"s|^DB_ROOT_PASSWORD=.*|DB_ROOT_PASSWORD=\${rootp}|\" deploy/betatest/env.example > .env
    chmod 600 .env"
}

# Main-domain settings, kept in line on every deploy (the rest of .env is never touched).
step "Pointing the app at https://${DOMAIN} (APP_ENV=production)"
ssh_run "set -e; cd ${REMOTE_DIR}
  sed -i -e 's|^APP_ENV=.*|APP_ENV=production|' -e 's|^APP_URL=.*|APP_URL=https://${DOMAIN}|' .env
  grep -q '^APP_CANONICAL_REDIRECT=' .env || echo 'APP_CANONICAL_REDIRECT=false' >> .env"
# www / betatest → ritme.app only once the main domain really answers through the CDN; before that a redirect
# would send every visitor of the still-working beta hostname into the void.
if curl -s -o /dev/null -w '%{http_code}' --max-time 20 "https://${DOMAIN}/up" 2>/dev/null | grep -q '^200$'; then
  ssh_run "sed -i 's|^APP_CANONICAL_REDIRECT=.*|APP_CANONICAL_REDIRECT=true|' ${REMOTE_DIR}/.env"
  echo "   https://${DOMAIN} is public → APP_CANONICAL_REDIRECT=true (www/${LEGACY_DOMAIN} 301 to ${DOMAIN})"
else
  ssh_run "sed -i 's|^APP_CANONICAL_REDIRECT=.*|APP_CANONICAL_REDIRECT=false|' ${REMOTE_DIR}/.env"
  echo "   https://${DOMAIN} not public yet (DNS?) → APP_CANONICAL_REDIRECT=false, ${LEGACY_DOMAIN} keeps serving"
fi

# ── 3. Production proxy: site vhosts ────────────────────────────────────────
if [[ "${NO_PROXY:-0}" != "1" ]]; then
  step "Hooking ${DOMAIN}, ${WWW_DOMAIN} and ${LEGACY_DOMAIN} into the production proxy"
  ssh_run "docker network inspect ritme-edge >/dev/null 2>&1 || docker network create ritme-edge" >/dev/null

  # Only these three files of the production tree are touched (never a full rsync of the monorepo).
  changed=0
  for f in docker-compose.prod.yml deploy/site-ssl.conf deploy/site-app.inc deploy/site-off.conf; do
    if ! ssh_run "cat ${PROD_DIR}/${f} 2>/dev/null" | cmp -s - "${REPO_DIR}/${f}"; then
      ssh_run "test -f ${PROD_DIR}/${f} && cp ${PROD_DIR}/${f} ${PROD_DIR}/${f}.bak-betatest || true"
      rsync -az -e "ssh ${SSH_OPTS[*]}" "${REPO_DIR}/${f}" "${SERVER}:${PROD_DIR}/${f}"
      changed=1
      echo "   updated ${PROD_DIR}/${f}"
    fi
  done

  # Origin certificate placeholders until enable-https.sh installs the site's own.
  ssh_run "cd ${PROD_DIR} && test -s ssl/betatest-fullchain.pem || { install -m 644 ssl/fullchain.pem ssl/betatest-fullchain.pem && install -m 600 ssl/privkey.pem ssl/betatest-privkey.pem; }"
  ssh_run "cd ${PROD_DIR} && test -s ssl/site-fullchain.pem || { install -m 644 ssl/betatest-fullchain.pem ssl/site-fullchain.pem && install -m 600 ssl/betatest-privkey.pem ssl/site-privkey.pem; }"

  if ! ssh_run "grep -qx 'SITE_CONF=./deploy/site-ssl.conf' ${PROD_DIR}/.env"; then
    ssh_run "sed -i '/^SITE_CONF=/d' ${PROD_DIR}/.env && echo 'SITE_CONF=./deploy/site-ssl.conf' >> ${PROD_DIR}/.env"
    changed=1
  fi

  if [[ "$changed" == "1" ]]; then
    # Dry run with the NEW mounts in a throw-away container before the live proxy is recreated.
    ssh_run "${PROD_COMPOSE} run --rm --no-deps -T proxy nginx -t" \
      || die "nginx -t failed with the site vhosts — live proxy NOT touched (restore *.bak-betatest if needed)"
    ssh_run "${PROD_COMPOSE} up -d --no-deps proxy && sleep 3 && ${PROD_COMPOSE} exec -T proxy nginx -t"
  else
    echo "   proxy config unchanged"
  fi
fi

# ── 4. Build, start, migrate ────────────────────────────────────────────────
step "Building the site image on the server"
ssh_run "${SITE_COMPOSE} build app"
step "Starting ritme-site (db, app, scheduler)"
ssh_run "${SITE_COMPOSE} up -d --remove-orphans"
ssh_run "for i in \$(seq 1 60); do s=\$(docker inspect -f '{{.State.Health.Status}}' ritme-site-app-1 2>/dev/null || echo none); [ \"\$s\" = healthy ] && exit 0; sleep 3; done; docker logs --tail 50 ritme-site-app-1; exit 1" \
  || die "app container did not become healthy"

ARTISAN="${SITE_COMPOSE} exec -T -u www-data app php artisan"
step "Running migrations + upgrade steps (php artisan app:upgrade)"
ssh_run "${ARTISAN} app:upgrade --skip-checks --no-interaction" || die "app:upgrade failed — the site stays in maintenance mode"

# ── 5. First deploy: content + admin ────────────────────────────────────────
if ! ssh_run "test -f ${REMOTE_DIR}/.installed"; then
  step "First deploy: FAQ + demo content + super-admin"
  ssh_run "${ARTISAN} db:seed --class='Database\\Seeders\\FaqSeeder' --force --no-interaction"
  if ssh_run "grep -qx 'SEED_DEMO=1' ${REMOTE_DIR}/.env"; then
    for seeder in BlogSeeder DirectorySeeder ShopSeeder; do
      ssh_run "${ARTISAN} db:seed --class='Database\\Seeders\\${seeder}' --force --no-interaction"
    done
  fi
  ssh_run "set -e; pw=\"Rt\$(openssl rand -hex 10)9\"
    printf '%s\n%s\n' \"\$pw\" \"\$pw\" | ${ARTISAN} admin:create --name='Ritme Admin' --email='admin@ritme.app' --role=super-admin
    umask 077; printf 'site admin (https://${DOMAIN}/admin)\nemail: admin@ritme.app\npassword: %s\n(super-admin: enrol TOTP on first login)\n' \"\$pw\" > /root/ritme-site-credentials.txt
    touch ${REMOTE_DIR}/.installed"
  ssh_run "${ARTISAN} cache:ns bump-all --no-interaction"
fi

# ── 6. HTTPS for the origin hop + smoke tests ───────────────────────────────
if ! ssh_run "test -f /etc/letsencrypt/live/${DOMAIN}/fullchain.pem && test -f /etc/letsencrypt/live/${LEGACY_DOMAIN}/fullchain.pem"; then
  step "Issuing the origin certificates (${DOMAIN}, ${WWW_DOMAIN}, ${LEGACY_DOMAIN})"
  ssh_run "cd ${REMOTE_DIR} && DOMAIN=${DOMAIN} WWW_DOMAIN=${WWW_DOMAIN} LEGACY_DOMAIN=${LEGACY_DOMAIN} bash deploy/betatest/enable-https.sh" || true
fi

step "Smoke tests"
fail=0
check() { # $1 label, $2 expected code(s) "200" or "403|404", $3.. curl args
  local label="$1" want="$2"; shift 2
  local got; got="$(curl -s -o /dev/null -w '%{http_code}' --max-time 30 "$@" || echo 000)"
  if [[ "|${want}|" == *"|${got}|"* ]]; then echo "   ✔ ${label} → ${got}"; else echo "   ✘ ${label} → ${got} (want ${want})"; fail=1; fi
}
origin=(--resolve "${DOMAIN}:443:89.251.8.115" -k)
check "origin /"             200 "${origin[@]}" "https://${DOMAIN}/"
check "origin /up"           200 "${origin[@]}" "https://${DOMAIN}/up"
check "origin /admin/login"  200 "${origin[@]}" "https://${DOMAIN}/admin/login"
check "origin /blog"         200 "${origin[@]}" "https://${DOMAIN}/blog"
check "origin /.env"         "403|404" "${origin[@]}" "https://${DOMAIN}/.env"
if curl -s --max-time 30 "${origin[@]}" "https://${DOMAIN}/robots.txt" | grep -q "^Sitemap: https://${DOMAIN}/sitemap.xml"; then
  echo "   ✔ robots.txt is the production one (sitemap on https://${DOMAIN})"; else echo "   ✘ robots.txt is not production (APP_ENV?)"; fail=1; fi
check "origin www /up"       "200|301" --resolve "${WWW_DOMAIN}:443:89.251.8.115" -k "https://${WWW_DOMAIN}/up"
check "origin ${LEGACY_DOMAIN} /up" "200|301" --resolve "${LEGACY_DOMAIN}:443:89.251.8.115" -k "https://${LEGACY_DOMAIN}/up"
if curl -sI --max-time 30 --resolve "${LEGACY_DOMAIN}:443:89.251.8.115" -k "https://${LEGACY_DOMAIN}/" | grep -qi '^x-robots-tag: noindex'; then
  echo "   ✔ ${LEGACY_DOMAIN} answers with X-Robots-Tag noindex"; else echo "   ✘ ${LEGACY_DOMAIN} lacks X-Robots-Tag noindex"; fail=1; fi
check "CDN https://${LEGACY_DOMAIN}/" "200|301" "https://${LEGACY_DOMAIN}/"
if dig +short "${DOMAIN}" A | grep -q .; then
  check "CDN https://${DOMAIN}/" 200 "https://${DOMAIN}/"
else
  echo "   ⚠ ${DOMAIN} has no DNS record yet — add it at ArvanCloud (A → 89.251.8.115 proxied, same for ${WWW_DOMAIN}), then re-run"
fi
# Production must be untouched.
check "prod api.ritme.app"   200 --resolve "api.ritme.app:443:89.251.8.115" "https://api.ritme.app/up"
check "prod web.ritme.app"   307 --resolve "web.ritme.app:443:89.251.8.115" "https://web.ritme.app/"
check "prod adpanell"        302 --resolve "adpanell.ritme.app:443:89.251.8.115" "https://adpanell.ritme.app/"

[[ "$fail" == "0" ]] && step "✅ ${DOMAIN} deployed" || die "smoke tests failed (see above)"
ssh_run "test -f /root/ritme-site-credentials.txt && echo '   admin credentials: /root/ritme-site-credentials.txt on the server' || true"
