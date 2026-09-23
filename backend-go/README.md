# Ritme API — Go

Go (Fiber v3) rewrite of the Laravel backend in `../backend`, running beside it during the strangler migration.
Design and plan: [`../docs/go-migration/`](../docs/go-migration/README.md). Agent rules: [`CLAUDE.md`](CLAUDE.md).

## Quick start

```bash
make test-db-up      # MariaDB 11.4 + Redis in docker (127.0.0.1:13317 / 127.0.0.1:16380)
make run             # http://localhost:8020/up
make test lint
make docker          # ritme-backend-go:dev (runs as uid 33, port 80, HEALTHCHECK on /up)
```

Put local overrides (e.g. `DB_PORT=3307` to use the main compose MariaDB) in `backend-go/.env`; the Makefile
loads it.

## Configuration

Same variable names as the Laravel backend, so both can share one `.env`:

| Group | Variables |
|---|---|
| App | `APP_NAME`, `APP_ENV` (default `production`), `APP_DEBUG`, `APP_URL`, `APP_TIMEZONE` (`Asia/Tehran`), `APP_LOCALE` |
| DB | `DB_HOST`\*, `DB_PORT` (3306), `DB_DATABASE`\*, `DB_USERNAME`\*, `DB_PASSWORD` |
| Redis | `REDIS_HOST`\*, `REDIS_PORT` (6379), `REDIS_USERNAME`, `REDIS_PASSWORD`, `REDIS_DB` (0) |
| CORS | `CORS_ALLOWED_ORIGINS` (comma list; default web.ritme.app + localhost:3000) |
| Sessions | `PASSPORT_TOKEN_LIFETIME_DAYS` (365), `PASSPORT_REFRESH_WINDOW_DAYS` (30), `PASSPORT_PASSWORD_CLIENT_ID/SECRET` |
| SMS | `SMS_PROVIDER`, `KAVENEGAR_API_KEY`, `KAVENEGAR_SENDER`, `KAVENEGAR_TEMPLATE_LOGIN_OTP`, `SMSIR_API_KEY`, `SMSIR_TEMPLATE_ID`, `SMSIR_LINE_NUMBER` |
| Other | `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`, `TELEGRAM_TIMEOUT`, `SWAGGER_USER`, `SWAGGER_PASSWORD` |
| Go only | `HTTP_ADDR` (`:8020`, image `:80`), `STORAGE_PATH`\* (the mounted Laravel `storage/`), `REDIS_PREFIX` (`ritme-go:`) |

\* required — the service refuses to start and lists every missing/invalid variable. `APP_DEBUG=true` is
rejected when `APP_ENV=production`.

## Docker

The image runs as **uid 33** (www-data of the PHP image) so it can read the 0600 Passport keys on the shared
`backend-storage` volume; mount it at `STORAGE_PATH` (default `/var/www/html/storage`). `api healthcheck` is the
HEALTHCHECK command (no curl needed). The runtime base is `alpine` because `gcr.io` (distroless) returns
403 from Iranian IPs; `GOPROXY` falls through to `goproxy.io` for the same reason.
