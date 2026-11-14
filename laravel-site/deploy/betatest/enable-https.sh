#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Issue the Let's Encrypt origin certificates for the marketing site and install them for the production proxy:
#   ssl/site-*.pem      ritme.app + www.ritme.app   (DOMAIN / WWW_DOMAIN)
#   ssl/betatest-*.pem  betatest.ritme.app          (LEGACY_DOMAIN, the old beta hostname)
# Run ON THE SERVER from /opt/ritme-site (deploy-betatest.sh calls it while a certificate is missing):
#
#   bash deploy/betatest/enable-https.sh
#
# Until a certificate exists the vhost serves a placeholder copy (ArvanCloud terminates the public TLS in front, so
# the origin certificate only secures the CDN → origin hop). The challenge is answered over HTTP-01 from
# production's certbot webroot; ArvanCloud may redirect http → https first, which Let's Encrypt follows — acme.inc
# is included on the :443 blocks too. www is optional: when its DNS is missing the main name is issued alone.
# Separate files + a separate renewal hook: a failed site certificate can never touch production's.
# Idempotent; a failed issuance keeps the placeholder and exits 0 with a warning.
# -----------------------------------------------------------------------------
set -uo pipefail

DOMAIN="${DOMAIN:-ritme.app}"
WWW_DOMAIN="${WWW_DOMAIN:-www.ritme.app}"
LEGACY_DOMAIN="${LEGACY_DOMAIN:-betatest.ritme.app}"
PROD_DIR="${PROD_DIR:-/opt/ritme}"
EMAIL="${CERTBOT_EMAIL:-admin@ritme.app}"
PROD_COMPOSE="docker compose -f ${PROD_DIR}/docker-compose.yml -f ${PROD_DIR}/docker-compose.prod.yml --project-directory ${PROD_DIR}"

command -v certbot >/dev/null || { apt-get update -qq && apt-get install -y -qq certbot; }

issue() { # $1 lineage name, $2.. names on the certificate
    local lineage="$1"; shift
    local args=(); local n; for n in "$@"; do args+=(-d "$n"); done
    certbot certonly --webroot -w "${PROD_DIR}/certbot-www" --non-interactive --agree-tos -m "$EMAIL" \
        --keep-until-expiring --cert-name "$lineage" "${args[@]}"
}
install_pair() { # $1 lineage, $2 file prefix
    install -m 644 "/etc/letsencrypt/live/$1/fullchain.pem" "${PROD_DIR}/ssl/$2-fullchain.pem"
    install -m 600 "/etc/letsencrypt/live/$1/privkey.pem"   "${PROD_DIR}/ssl/$2-privkey.pem"
}

changed=0
# Main domain (+ www when it resolves).
if issue "$DOMAIN" "$DOMAIN" "$WWW_DOMAIN" || issue "$DOMAIN" "$DOMAIN"; then
    install_pair "$DOMAIN" site; changed=1
else
    echo "⚠ certbot could not issue ${DOMAIN} (DNS not pointing here yet? CDN challenge path?) — keeping the placeholder certificate." >&2
fi
# Old beta hostname.
if issue "$LEGACY_DOMAIN" "$LEGACY_DOMAIN"; then
    install_pair "$LEGACY_DOMAIN" betatest; changed=1
else
    echo "⚠ certbot could not issue ${LEGACY_DOMAIN} — keeping the current certificate." >&2
fi

mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/ritme-site-proxy.sh <<HOOK
#!/usr/bin/env bash
set -e
if [ -f /etc/letsencrypt/live/${DOMAIN}/fullchain.pem ]; then
    install -m 644 /etc/letsencrypt/live/${DOMAIN}/fullchain.pem ${PROD_DIR}/ssl/site-fullchain.pem
    install -m 600 /etc/letsencrypt/live/${DOMAIN}/privkey.pem   ${PROD_DIR}/ssl/site-privkey.pem
fi
if [ -f /etc/letsencrypt/live/${LEGACY_DOMAIN}/fullchain.pem ]; then
    install -m 644 /etc/letsencrypt/live/${LEGACY_DOMAIN}/fullchain.pem ${PROD_DIR}/ssl/betatest-fullchain.pem
    install -m 600 /etc/letsencrypt/live/${LEGACY_DOMAIN}/privkey.pem   ${PROD_DIR}/ssl/betatest-privkey.pem
fi
${PROD_COMPOSE} exec -T proxy nginx -s reload
HOOK
chmod +x /etc/letsencrypt/renewal-hooks/deploy/ritme-site-proxy.sh
systemctl enable --now certbot.timer 2>/dev/null || true

if [[ "$changed" == "1" ]]; then
    $PROD_COMPOSE exec -T proxy nginx -t && $PROD_COMPOSE exec -T proxy nginx -s reload
    echo "✅ site certificates installed"
fi
