#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Issue a Let's Encrypt certificate for a beta host and install it as ssl/<prefix>-*.pem for the production proxy.
# Run ON THE SERVER from /opt/ritme (deploy-beta.sh calls it once per host):
#
#   bash deploy/enable-https-beta.sh                                  # beta.ritme.app       -> ssl/beta-*.pem
#   bash deploy/enable-https-beta.sh beta-admin.ritme.app beta-admin  # beta-admin.ritme.app -> ssl/beta-admin-*.pem
#
# Until then ssl/beta-*.pem is a copy of production's certificate (ArvanCloud terminates the public TLS, so the
# origin certificate only secures the CDN → origin hop). HTTP-01 from production's certbot webroot; acme.inc is on
# the :443 block too, in case ArvanCloud redirects the challenge to https first.
# Separate files + a separate renewal hook: a failed beta certificate can never touch production's.
# Idempotent; a failed issuance keeps the placeholder and exits 0 with a warning.
# -----------------------------------------------------------------------------
set -uo pipefail

DOMAIN="${1:-beta.ritme.app}"
PREFIX="${2:-beta}"
PROD_DIR="${PROD_DIR:-/opt/ritme}"
EMAIL="${CERTBOT_EMAIL:-admin@ritme.app}"
PROD_COMPOSE="docker compose -f ${PROD_DIR}/docker-compose.yml -f ${PROD_DIR}/docker-compose.prod.yml --project-directory ${PROD_DIR}"

command -v certbot >/dev/null || { apt-get update -qq && apt-get install -y -qq certbot; }

if ! certbot certonly --webroot -w "${PROD_DIR}/certbot-www" --non-interactive --agree-tos -m "$EMAIL" \
        --keep-until-expiring -d "$DOMAIN"; then
    echo "⚠ certbot could not issue ${DOMAIN} (CDN challenge path?) — keeping the placeholder certificate." >&2
    exit 0
fi

install -m 644 "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" "${PROD_DIR}/ssl/${PREFIX}-fullchain.pem"
install -m 600 "/etc/letsencrypt/live/${DOMAIN}/privkey.pem"   "${PROD_DIR}/ssl/${PREFIX}-privkey.pem"

mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/ritme-${PREFIX}-proxy.sh <<HOOK
#!/usr/bin/env bash
set -e
[ -f /etc/letsencrypt/live/${DOMAIN}/fullchain.pem ] || exit 0
install -m 644 /etc/letsencrypt/live/${DOMAIN}/fullchain.pem ${PROD_DIR}/ssl/${PREFIX}-fullchain.pem
install -m 600 /etc/letsencrypt/live/${DOMAIN}/privkey.pem   ${PROD_DIR}/ssl/${PREFIX}-privkey.pem
${PROD_COMPOSE} exec -T proxy nginx -s reload
HOOK
chmod +x /etc/letsencrypt/renewal-hooks/deploy/ritme-${PREFIX}-proxy.sh
systemctl enable --now certbot.timer 2>/dev/null || true

$PROD_COMPOSE exec -T proxy nginx -t && $PROD_COMPOSE exec -T proxy nginx -s reload
echo "✅ ${DOMAIN}: own certificate installed"
