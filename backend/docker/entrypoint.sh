#!/bin/sh
# Shared entrypoint for the web (apache) and queue-worker containers.
# Migrations / Passport bootstrap run only where RUN_MIGRATIONS=true so the
# queue worker doesn't race the web container on first boot.
set -e

cd /var/www/html

# storage/ is a named volume; on a fresh volume make sure Laravel's expected
# directory skeleton exists and is writable by the PHP user.
mkdir -p \
  storage/app/public \
  storage/framework/cache/data \
  storage/framework/sessions \
  storage/framework/views \
  storage/logs \
  bootstrap/cache
chown -R www-data:www-data storage bootstrap/cache

if [ "$RUN_MIGRATIONS" = "true" ]; then
  tries=0
  until php artisan migrate --force; do
    tries=$((tries + 1))
    if [ "$tries" -ge 10 ]; then
      echo "entrypoint: migrations still failing after ${tries} attempts, giving up" >&2
      exit 1
    fi
    echo "entrypoint: database not ready, retrying in 3s (${tries}/10)"
    sleep 3
  done

  # Passport signing keys live on the storage volume, so they survive rebuilds
  # and both containers see the same pair.
  if [ ! -f storage/oauth-private.key ]; then
    php artisan passport:keys --force
    chown www-data:www-data storage/oauth-private.key storage/oauth-public.key
  fi

  # createToken() needs a personal-access client row; create it once.
  clients=$(php artisan tinker --execute='echo \Laravel\Passport\Client::count();' 2>/dev/null | tr -dc '0-9')
  if [ -z "$clients" ] || [ "$clients" = "0" ]; then
    php artisan passport:client --personal --name="Ritme Personal Access" --no-interaction
  fi

  php artisan storage:link || true
fi

exec "$@"
