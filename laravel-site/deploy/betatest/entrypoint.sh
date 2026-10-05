#!/usr/bin/env bash
# Ritme site — container entrypoint (betatest).
#   app        → Apache (default CMD)
#   scheduler  → `php artisan schedule:work` (drains the database queue every minute, nightly backups, audits …)
# Migrations are NOT run here: deploy-betatest.sh runs `php artisan app:upgrade` explicitly after `up -d`, so a
# failing migration stops the deploy loudly instead of crash-looping a container.
set -euo pipefail
cd /var/www/html

# Named volumes start empty on first use (or keep an older layout): make sure Laravel's folders exist and are writable.
mkdir -p storage/app/private storage/app/pending storage/framework/cache/data storage/framework/sessions \
    storage/framework/views storage/logs bootstrap/cache public/media
chown -R www-data:www-data storage bootstrap/cache public/media

# Wait for MariaDB (healthcheck already gates startup; this covers restarts of the database alone).
for _ in $(seq 1 60); do
    php -r 'try { new PDO("mysql:host=".getenv("DB_HOST").";port=".(getenv("DB_PORT") ?: 3306), getenv("DB_USERNAME"), getenv("DB_PASSWORD")); exit(0); } catch (Throwable $e) { exit(1); }' && break
    sleep 2
done

if [ "${1:-}" = "scheduler" ]; then
    exec su -s /bin/bash www-data -c 'php artisan schedule:work --no-interaction'
fi

exec "$@"
