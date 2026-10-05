#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Issue a Let's Encrypt certificate for betatest.ritme.app and install it as ssl/betatest-*.pem for the production
# proxy. Run ON THE SERVER from /opt/ritme-site (deploy-betatest.sh calls it once the vhost answers):
#
#   bash deploy/betatest/enable-https.sh
#
# Until a certificate exists the vhost serves a copy of production's certificate (ArvanCloud terminates the public
# TLS in front, so the origin certificate only secures the CDN → origin hop). The challenge is answered over
# HTTP-01 from production's certbot webroot; ArvanCloud may redirect http → https first, which Let's Encrypt
# follows — acme.inc is included on the :443 block too.
# Separate files + a separate renewal hook: a failed betatest certificate can never touch production's.
# Idempotent; a failed issuance keeps the placeholder and exits 0 with a warning.
# -----------------------------------------------------------------------------
set -uo pipefail

DOMAIN="${DOMAIN:-betatest.ritme.app}"
PROD_DIR="${PROD_DIR:-/opt/ritme}"
EMAIL="${CERTBOT_EMAIL:-admin@ritme.app}"
PROD_COMPOSE="docker compose -f ${PROD_DIR}/docker-compose.yml -f ${PROD_DIR}/docker-compose.prod.yml --project-directory ${PROD_DIR}"

command -v certbot >/dev/null || { apt-get update -qq && apt-get install -y -qq certbot; }

if ! certbot certonly --webroot -w "${PROD_DIR}/certbot-www" --non-interactive --agree-tos -m "$EMAIL" \
        --keep-until-expiring -d "$DOMAIN"; then
    echo "⚠ certbot could not issue ${DOMAIN} (CDN challenge path?) — keeping the placeholder certificate." >&2
    exit 0
fi

install -m 644 "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" "${PROD_DIR}/ssl/betatest-fullchain.pem"
install -m 600 "/etc/letsencrypt/live/${DOMAIN}/privkey.pem"   "${PROD_DIR}/ssl/betatest-privkey.pem"

mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/ritme-site-proxy.sh <<HOOK
#!/usr/bin/env bash
set -e
[ -f /etc/letsencrypt/live/${DOMAIN}/fullchain.pem ] || exit 0
install -m 644 /etc/letsencrypt/live/${DOMAIN}/fullchain.pem ${PROD_DIR}/ssl/betatest-fullchain.pem
install -m 600 /etc/letsencrypt/live/${DOMAIN}/privkey.pem   ${PROD_DIR}/ssl/betatest-privkey.pem
${PROD_COMPOSE} exec -T proxy nginx -s reload
HOOK
chmod +x /etc/letsencrypt/renewal-hooks/deploy/ritme-site-proxy.sh
systemctl enable --now certbot.timer 2>/dev/null || true

$PROD_COMPOSE exec -T proxy nginx -t && $PROD_COMPOSE exec -T proxy nginx -s reload
echo "✅ ${DOMAIN}: own certificate installed"
