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

  # Passport signing keys live ONLY on the storage volume (they are excluded
  # from the rsync and the image build), so they survive rebuilds and both
  # containers see the same pair. Never regenerate an existing pair: new keys
  # invalidate every issued token and sign out every user.
  if [ -f storage/oauth-private.key ] && [ -f storage/oauth-public.key ]; then
    echo "entrypoint: passport keys present, leaving them untouched"
  elif [ -f storage/oauth-private.key ] || [ -f storage/oauth-public.key ]; then
    echo "entrypoint: FATAL: only one passport key exists in storage/ — refusing to generate a new pair over it" >&2
    exit 1
  else
    # Keys are missing. If tokens were already issued, generating a pair now
    # will sign out every user: say so loudly (a fresh install has none).
    tokens=$(php artisan tinker --execute='echo "count=".\Laravel\Passport\Passport::token()->count();' 2>/dev/null \
      | sed -n 's/^count=\([0-9][0-9]*\)$/\1/p' | tail -n 1)
    if [ -n "$tokens" ] && [ "$tokens" != "0" ]; then
      echo "entrypoint: WARNING: passport keys are MISSING but ${tokens} access tokens exist." >&2
      echo "entrypoint: WARNING: generating a new pair — every existing session is now invalid." >&2
    fi
    php artisan passport:keys
    chown www-data:www-data storage/oauth-private.key storage/oauth-public.key
  fi

  # createToken() needs a personal-access client row; create it once. Only an
  # explicit count of 0 creates one: a failed or silent tinker (DB hiccup,
  # PHP warning) must never be mistaken for "no client" and add a second one.
  clients=$(php artisan tinker --execute='echo "count=".\Laravel\Passport\Client::count();' 2>/dev/null \
    | sed -n 's/^count=\([0-9][0-9]*\)$/\1/p' | tail -n 1) || clients=""
  if [ "$clients" = "0" ]; then
    php artisan passport:client --personal --name="Ritme Personal Access" --no-interaction
  elif [ -z "$clients" ]; then
    echo "entrypoint: WARNING: could not count passport clients; not creating one" >&2
  fi

  php artisan storage:link || true

  # Idempotent essential data: the language registry, the initial admin
  # (credentials from ADMIN_SEED_*), the editable smart-message content and the
  # daily recommendations. All use firstOrCreate, so existing rows / admin edits
  # are never overwritten.
  php artisan db:seed --class=Database\\Seeders\\LanguageSeeder --force || true
  php artisan db:seed --class=Database\\Seeders\\AdminSeeder --force || true
  php artisan db:seed --class=Database\\Seeders\\MessageContentSeeder --force || true
  php artisan db:seed --class=Database\\Seeders\\RecommendationSeeder --force || true

  # Build the OpenAPI spec once at boot. `generate_always` is off (regenerating
  # per request is a DoS vector), so the spec is produced here instead; the docs
  # UI is gated by SwaggerBasicAuth regardless.
  php artisan l5-swagger:generate || true
fi

exec "$@"
