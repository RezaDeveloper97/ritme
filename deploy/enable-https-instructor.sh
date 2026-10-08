#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Issue a Let's Encrypt certificate for an instructor host and switch its vhost
# to HTTPS (bloom B-N8-05). Run ON THE SERVER, from /opt/ritme, after DNS points
# here and the :80 vhost is on (docs/night-bloom/instructor-web-deploy.md):
#
#   ssh root@<server> 'cd /opt/ritme && ./deploy/enable-https-instructor.sh stage'
#   ssh root@<server> 'cd /opt/ritme && ./deploy/enable-https-instructor.sh prod'
#
# Own certificate per host (ssl/instructor-stage-*.pem, ssl/instructor-*.pem):
# prod's SAN certificate and the stage certificate are never touched.
# Idempotent: re-running re-copies the cert and reloads nginx.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail
cd "$(dirname "$0")/.."

case "${1:-}" in
  stage) DOMAIN=instructor.ritmeapp.ir; PREFIX=instructor-stage; VAR=INSTRUCTOR_STAGE_CONF; CONF=./deploy/instructor-stage-ssl.conf ;;
  prod)  DOMAIN=instructor.ritme.app;   PREFIX=instructor;       VAR=INSTRUCTOR_CONF;       CONF=./deploy/instructor-ssl.conf ;;
  *) echo "usage: $0 stage|prod" >&2; exit 2 ;;
esac
EMAIL="${CERTBOT_EMAIL:-admin@${DOMAIN#*.}}"
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.prod.yml"

command -v certbot >/dev/null || { apt-get update -qq && apt-get install -y -qq certbot; }
mkdir -p certbot-www ssl

certbot certonly --webroot -w "$(pwd)/certbot-www" \
  --non-interactive --agree-tos -m "$EMAIL" --keep-until-expiring -d "$DOMAIN"

install -m 644 "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" "ssl/${PREFIX}-fullchain.pem"
install -m 600 "/etc/letsencrypt/live/${DOMAIN}/privkey.pem"   "ssl/${PREFIX}-privkey.pem"

# Point the selector at the TLS config and keep it across deploys.
if grep -q "^${VAR}=" .env 2>/dev/null; then
  sed -i "s|^${VAR}=.*|${VAR}=${CONF}|" .env
else
  echo "${VAR}=${CONF}" >> .env
fi

mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > "/etc/letsencrypt/renewal-hooks/deploy/ritme-${PREFIX}-proxy.sh" <<HOOK
#!/usr/bin/env bash
set -e
[ -f /etc/letsencrypt/live/${DOMAIN}/fullchain.pem ] || exit 0
install -m 644 /etc/letsencrypt/live/${DOMAIN}/fullchain.pem /opt/ritme/ssl/${PREFIX}-fullchain.pem
install -m 600 /etc/letsencrypt/live/${DOMAIN}/privkey.pem   /opt/ritme/ssl/${PREFIX}-privkey.pem
docker compose -f /opt/ritme/docker-compose.yml -f /opt/ritme/docker-compose.prod.yml \\
  --project-directory /opt/ritme exec -T proxy nginx -s reload
HOOK
chmod +x "/etc/letsencrypt/renewal-hooks/deploy/ritme-${PREFIX}-proxy.sh"
systemctl enable --now certbot.timer 2>/dev/null || true

# Recreate (not reload): the selector is a bind-mount source.
$COMPOSE up -d proxy
sleep 3
$COMPOSE exec -T proxy nginx -t
echo "✅ HTTPS enabled for ${DOMAIN}"
