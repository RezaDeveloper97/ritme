# Ritme API — Go

Go (Fiber v3) rewrite of the Laravel backend in `../backend`. **All new backend work happens here.** Staging runs
Go only; production runs Laravel until the cutover (T-M2-26/27), and `../backend` is frozen (production maintenance
only). Design and plan: [`../docs/go-migration/`](../docs/go-migration/README.md). Agent rules: [`CLAUDE.md`](CLAUDE.md).

## Quick start

```bash
make test-db-up      # MariaDB 11.4 + Redis in docker (127.0.0.1:13317 / 127.0.0.1:16380)
make run             # http://localhost:8020/up (database ritme_test on the test stack)
make test lint
make docker          # ritme-backend-go:dev (runs as uid 33, port 80, HEALTHCHECK on /up)
```

Put local overrides (e.g. `DB_PORT=3307` to use the main compose MariaDB) in `backend-go/.env`; the Makefile
loads it. `make help` lists every target.

## Local dev (API you can log in to)

`make run` alone starts against an empty `ritme_test`. For a usable API, give it its own database on the test stack,
apply the goose migrations and seed it (all commands from `backend-go/`):

```bash
make test-db-up
db() { docker compose -f docker-compose.test.yml exec -T mariadb mariadb "$@"; }   # bash and zsh
db -uroot -proot -e "CREATE DATABASE IF NOT EXISTS ritme_dev; GRANT ALL ON ritme_dev.* TO 'ritme'@'%';"

# a) empty schema: goose baseline (00001) + any later migrations
go run -tags "no_postgres no_sqlite3 no_mssql no_vertica no_clickhouse no_ydb no_libsql" \
  github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir db/migrations mysql \
  "ritme:ritme@tcp(127.0.0.1:13317)/ritme_dev?parseTime=true&multiStatements=true" up
#    + the Passport personal access client (tokens need one) and a user
db -uritme -pritme ritme_dev -e "
  INSERT IGNORE INTO oauth_clients (id,name,provider,redirect_uris,grant_types,revoked,created_at,updated_at)
    VALUES ('0199c0de-0000-7000-8000-00000000de01','Ritme Personal Access','users','[]','[\"personal_access\"]',0,NOW(),NOW());
  INSERT IGNORE INTO users (name,mobile,created_at,updated_at) VALUES ('Dev User','09120000001',NOW(),NOW());"

# b) OR rich data instead of (a): the contract fixtures (content, 18 personas 09900000001…18, dated around
#    2026-09-23). A Laravel-built dump: no goose_db_version, so use (a) when you are testing a new migration.
# db -uritme -pritme ritme_dev < contract/fixtures/dump.sql

DB_DATABASE=ritme_dev SMS_PROVIDER=log make run
```

Log in without SMS — `SMS_PROVIDER=log` only logs a masked line (refused when `APP_ENV=production`), there is no
OTP bypass; read the code from the database:

```bash
B=http://127.0.0.1:8020/api/v1 M=09120000001
curl -s -X POST $B/auth/send-otp -H 'Content-Type: application/json' -d "{\"mobile\":\"$M\"}"
CODE=$(db -uritme -pritme ritme_dev -N -e "select code from otp_verifications where mobile='$M' order by id desc limit 1")
TOKEN=$(curl -s -X POST $B/auth/verify-otp -H 'Content-Type: application/json' \
  -d "{\"mobile\":\"$M\",\"code\":\"$CODE\"}" | jq -r .data.access_token)
curl -s $B/auth/user -H "Authorization: Bearer $TOKEN"
```

- Passport keys come from `STORAGE_PATH` (default `../backend/storage/oauth-*.key`). Without them the API refuses to
  start: use `STORAGE_PATH=$PWD/contract/fixtures/keys` (contract-only pair; `/storage` files are then absent).
- `send-otp` allows one code per mobile per 60 s and 5 requests/min per IP.
- Web frontend: `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8020/api/v1` in `frontend/.env.local`
  (`http://localhost:3000` is in the default CORS list).
- Clean up: `db -uroot -proot -e "DROP DATABASE ritme_dev"`, or `make test-db-down` (wipes the whole test stack,
  including other agents' test databases).

## Contract harness

Existing `/api/v1` routes must answer exactly like Laravel. `make contract ROUTES=<group[,group…]|all>` boots a fresh
Go server on its own database and diffs it against the goldens in `contract/golden/<group>/` (cases in
`contract/cases/<group>.yaml`, exit 1 on any diff). `make contract-record ROUTES=<group>` re-records goldens from the
Laravel contract stack (`CONTRACT_PORT`, 18090 on the dev machine). Personas, clock header and stack:
[`../docs/go-migration/contract.md`](../docs/go-migration/contract.md). Intentional differences need a `D-nn` in
`deviations.md` and an allow-list entry in `contract/allowlist/<group>.yaml`.

## Migrations

goose migrations live in `db/migrations/` (embedded). Until T-M2-27 Laravel owns the production schema, so a schema
change is **a new goose migration plus the mirror Laravel migration** in the same commit, with `make schema-diff`
green. Never edit `00001_baseline.sql`. See [`../docs/go-migration/migrations.md`](../docs/go-migration/migrations.md).

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
