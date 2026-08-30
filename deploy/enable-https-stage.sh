#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Issue a Let's Encrypt certificate for the STAGING hostname and switch the
# staging vhost to HTTPS. Run ON THE SERVER, from /opt/ritme, after
# deploy-stage.sh has brought the stage stack up on :80.
#
#   ssh root@<server> 'cd /opt/ritme && ./deploy/enable-https-stage.sh'
#
# Deliberately separate from enable-https.sh: stage.ritmeapp.ir is on a
# different registrable domain than the production hostnames, so it needs its
# own certificate. Writing it to ssl/stage-*.pem keeps prod's ssl/fullchain.pem
# untouched — a botched staging cert can never take production offline.
#
# Idempotent: re-running just re-copies the cert and reloads nginx.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail
cd "$(dirname "$0")/.."

DOMAIN="${1:-stage.ritmeapp.ir}"
EMAIL="${CERTBOT_EMAIL:-admin@${DOMAIN#*.}}"
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

command -v certbot >/dev/null || { apt-get update -qq && apt-get install -y -qq certbot; }

mkdir -p certbot-www ssl

# HTTP-01 over the proxy's webroot. acme.inc opts out of auth_basic explicitly,
# so the challenge stays reachable even though the rest of the host needs a
# password.
certbot certonly --webroot -w "$(pwd)/certbot-www" \
  --non-interactive --agree-tos -m "$EMAIL" --keep-until-expiring -d "$DOMAIN"

install -m 644 "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" ssl/stage-fullchain.pem
install -m 600 "/etc/letsencrypt/live/${DOMAIN}/privkey.pem"   ssl/stage-privkey.pem

# Point the staging vhost at the TLS config and keep it that way across deploys.
if grep -q '^STAGE_CONF=' .env 2>/dev/null; then
  sed -i 's|^STAGE_CONF=.*|STAGE_CONF=./deploy/stage-ssl.conf|' .env
else
  echo 'STAGE_CONF=./deploy/stage-ssl.conf' >> .env
fi

# Renewal hook, kept separate from the production one so the two never race on
# the same file names.
mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/ritme-stage-proxy.sh <<HOOK
#!/usr/bin/env bash
set -e
[ -f /etc/letsencrypt/live/${DOMAIN}/fullchain.pem ] || exit 0
install -m 644 /etc/letsencrypt/live/${DOMAIN}/fullchain.pem /opt/ritme/ssl/stage-fullchain.pem
install -m 600 /etc/letsencrypt/live/${DOMAIN}/privkey.pem   /opt/ritme/ssl/stage-privkey.pem
docker compose -f /opt/ritme/docker-compose.yml -f /opt/ritme/docker-compose.prod.yml \\
  --project-directory /opt/ritme exec -T proxy nginx -s reload
HOOK
chmod +x /etc/letsencrypt/renewal-hooks/deploy/ritme-stage-proxy.sh
systemctl enable --now certbot.timer 2>/dev/null || true

# Recreate (not just reload) the proxy: STAGE_CONF selects a bind-mount source,
# which only takes effect when the container is recreated.
$COMPOSE up -d proxy
sleep 3
$COMPOSE exec -T proxy nginx -t

echo "✅ HTTPS enabled for staging: ${DOMAIN}"
