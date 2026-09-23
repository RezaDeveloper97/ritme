#!/usr/bin/env bash
# Push ONLY deploy/vhost-stage.inc to the shared proxy and reload nginx.
# Production config (proxy-ssl.conf, vhost-web.inc, ...) is not touched.
# The file is a single-file bind mount, so it is rewritten in place (same inode).
# nginx is reloaded only if `nginx -t` passes; otherwise the backup is restored.
set -euo pipefail
cd "$(dirname "$0")/.."

SERVER="${SERVER:-root@89.251.8.115}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_ed25519}"

ssh -i "$SSH_KEY" -o ConnectTimeout=20 "$SERVER" 'set -e
cd /opt/ritme
C="docker compose -f docker-compose.yml -f docker-compose.prod.yml"
BK=/root/vhost-stage.inc.bak-$(date +%Y%m%d-%H%M%S)
cp deploy/vhost-stage.inc "$BK"; echo "backup: $BK"
cat > deploy/vhost-stage.inc
if $C exec -T proxy grep -q "session/flag" /etc/nginx/conf.d/vhost-stage.inc && $C exec -T proxy nginx -t; then
  $C exec -T proxy nginx -s reload && echo "RELOADED"
else
  echo "nginx -t FAILED -> restoring backup"
  cat "$BK" > deploy/vhost-stage.inc
  $C exec -T proxy nginx -t
fi' < deploy/vhost-stage.inc
