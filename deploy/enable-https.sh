#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Issue a Let's Encrypt certificate and switch the proxy to HTTPS.
# Run ON THE SERVER, from /opt/ritme, AFTER the domains' DNS points here.
# With no arguments it covers the three production hostnames:
#
#   ssh root@<server> 'cd /opt/ritme && ./deploy/enable-https.sh'
#
# One SAN certificate covers all of them, so there is a single pair of files to
# copy and a single renewal to worry about.
#
# Idempotent: re-running just re-copies the cert and reloads nginx.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail
cd "$(dirname "$0")/.."

DOMAINS=("$@")
if [[ ${#DOMAINS[@]} -eq 0 ]]; then
  DOMAINS=(api.ritme.app adpanell.ritme.app web.ritme.app)
fi
PRIMARY="${DOMAINS[0]}"
# Strip the leftmost label so the contact address is on the registrable domain
# (admin@ritme.app) rather than admin@api.ritme.app.
EMAIL="${CERTBOT_EMAIL:-admin@${PRIMARY#*.}}"

command -v certbot >/dev/null || { apt-get update -qq && apt-get install -y -qq certbot; }

mkdir -p certbot-www ssl

# HTTP-01 over the proxy's webroot — the proxy must already be up on :80.
CERTBOT_ARGS=(certonly --webroot -w "$(pwd)/certbot-www"
              --non-interactive --agree-tos -m "$EMAIL" --keep-until-expiring)
for d in "${DOMAINS[@]}"; do CERTBOT_ARGS+=(-d "$d"); done
certbot "${CERTBOT_ARGS[@]}"

# The container can't follow /etc/letsencrypt/live's symlinks, so copy.
install -m 644 "/etc/letsencrypt/live/${PRIMARY}/fullchain.pem" ssl/fullchain.pem
install -m 600 "/etc/letsencrypt/live/${PRIMARY}/privkey.pem"   ssl/privkey.pem

# Point the proxy at the TLS config and keep it that way across deploys.
if grep -q '^PROXY_CONF=' .env 2>/dev/null; then
  sed -i 's|^PROXY_CONF=.*|PROXY_CONF=./deploy/proxy-ssl.conf|' .env
else
  echo 'PROXY_CONF=./deploy/proxy-ssl.conf' >> .env
fi

# Renewals: copy the fresh cert into ./ssl and reload nginx, driven by the
# system certbot.timer. (`certbot renew --dry-run` can fail where the staging
# ACME host doesn't resolve; real renewals still work.)
mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/ritme-proxy.sh <<HOOK
#!/usr/bin/env bash
set -e
install -m 644 /etc/letsencrypt/live/${PRIMARY}/fullchain.pem /opt/ritme/ssl/fullchain.pem
install -m 600 /etc/letsencrypt/live/${PRIMARY}/privkey.pem   /opt/ritme/ssl/privkey.pem
docker compose -f /opt/ritme/docker-compose.yml -f /opt/ritme/docker-compose.prod.yml \\
  --project-directory /opt/ritme exec -T proxy nginx -s reload
HOOK
chmod +x /etc/letsencrypt/renewal-hooks/deploy/ritme-proxy.sh
systemctl enable --now certbot.timer 2>/dev/null || true

docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d proxy
sleep 3
docker compose -f docker-compose.yml -f docker-compose.prod.yml exec -T proxy nginx -t

echo "✅ HTTPS enabled for: ${DOMAINS[*]}"
