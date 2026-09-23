---
name: local-dev
description: Start the Ritme local dev environment — backend-go (Go API on :8020 against the docker test-stack MariaDB/Redis, goose migrations, seed) plus the Next.js frontend — and log in without SMS (SMS_PROVIDER=log, OTP read from the DB). Laravel on :8010 only as a legacy fallback. Use when running or manually testing the app locally.
---

# Ritme local development

The backend is **`backend-go/`** (Go + Fiber v3). `backend/` (Laravel) is frozen and only still serves production;
use it locally only for the fallback at the end. All backend commands below run from `backend-go/`.

## 1. Database + Redis (docker test stack)

```bash
cd backend-go
make test-db-up        # MariaDB 11.4 on 127.0.0.1:13317 (root/root, ritme/ritme) + Redis on 127.0.0.1:16380
db() { docker compose -f docker-compose.test.yml exec -T mariadb mariadb "$@"; }   # helper, bash and zsh
db -uroot -proot -e "CREATE DATABASE IF NOT EXISTS ritme_dev; GRANT ALL ON ritme_dev.* TO 'ritme'@'%';"
```

Use a dedicated `ritme_dev` database: `ritme_test` and the `gt_*` databases belong to integration tests (other
agents may be running them on the same stack). Never `make test-db-down` while someone else may be testing — it
wipes the whole stack (tmpfs).

## 2. Schema + seed — pick one

**a) goose baseline + minimal seed** (empty app, correct for testing a new migration):

```bash
go run -tags "no_postgres no_sqlite3 no_mssql no_vertica no_clickhouse no_ydb no_libsql" \
  github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir db/migrations mysql \
  "ritme:ritme@tcp(127.0.0.1:13317)/ritme_dev?parseTime=true&multiStatements=true" up
db -uritme -pritme ritme_dev -e "
  INSERT IGNORE INTO oauth_clients (id,name,provider,redirect_uris,grant_types,revoked,created_at,updated_at)
    VALUES ('0199c0de-0000-7000-8000-00000000de01','Ritme Personal Access','users','[]','[\"personal_access\"]',0,NOW(),NOW());
  INSERT IGNORE INTO users (name,mobile,created_at,updated_at) VALUES ('Dev User','09120000001',NOW(),NOW());"
```

The personal access client row is required: without it `verify-otp` cannot issue a token. The baseline seeds the
`fa`/`en` language rows; there is no content (articles, pregnancy weeks, …) — use (b) for that.

**b) contract fixtures** (content + 18 personas `09900000001…18`, data dated around 2026-09-23; persona list in
`docs/go-migration/contract.md`). It is a Laravel-built dump with the same schema but no `goose_db_version`:

```bash
db -uritme -pritme ritme_dev < contract/fixtures/dump.sql
```

## 3. Run the API

```bash
DB_DATABASE=ritme_dev SMS_PROVIDER=log make run      # http://127.0.0.1:8020/up → OK
```

- `make run` defaults (Makefile): `APP_ENV=local`, `HTTP_ADDR=:8020`, test-stack DB/Redis,
  `STORAGE_PATH=../backend/storage` (Passport `oauth-*.key` + `/storage` files). Put permanent overrides in
  `backend-go/.env` (the Makefile includes it; never commit it).
- No local keys? `STORAGE_PATH=$PWD/contract/fixtures/keys` (contract-only key pair). Never generate or copy
  production keys.
- `SMS_PROVIDER=log` logs a masked "not sent" line instead of calling the gateway; it refuses to start with
  `APP_ENV=production`.
- Once T-M2-28 lands, backend-go can apply goose itself on start (`RUN_MIGRATIONS=true`); until then use step 2.

## 4. Log in without SMS

There is **no test mode and no OTP bypass** (a bypass is a way into every account). Request a code, then read it
from the local DB — this works only because you own the database:

```bash
B=http://127.0.0.1:8020/api/v1 M=09120000001
curl -s -X POST $B/auth/send-otp -H 'Content-Type: application/json' -d "{\"mobile\":\"$M\"}"
CODE=$(db -uritme -pritme ritme_dev -N -e "select code from otp_verifications where mobile='$M' order by id desc limit 1")
TOKEN=$(curl -s -X POST $B/auth/verify-otp -H 'Content-Type: application/json' \
  -d "{\"mobile\":\"$M\",\"code\":\"$CODE\"}" | jq -r .data.access_token)
curl -s $B/auth/user -H "Authorization: Bearer $TOKEN"
```

In the web UI: request the code on the login screen, run the `CODE=…` line, type it in. `send-otp` allows one code
per mobile per 60 s and 5 requests/min per IP. A never-seen mobile creates a new user on verify.

## 5. Frontend (Next.js)

```bash
cd frontend && npm run dev          # http://localhost:3000
```

`frontend/.env.local` must set `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8020/api/v1` (Laravel fallback:
`:8010`). It must never leak into prod builds — the prod URL is baked in by deploy.sh build args.
`http://localhost:3000` is in backend-go's default `CORS_ALLOWED_ORIGINS`.

Admin app: `admin-web/` (Next.js) talks to backend-go's `/api/admin/v1` — see `docs/go-migration/admin-web.md`.

## E2E / browser automation

Skip the login UI: get a real token with step 4, then set `localStorage.ritme_token` via `addInitScript` plus the
cookie `ritme_auth=1`.

## Clean up

```bash
db -uroot -proot -e "DROP DATABASE IF EXISTS ritme_dev"   # keeps the stack for other agents
```

## Legacy fallback: Laravel + sqlite on :8010

Only to reproduce a production (Laravel) bug; no new features there.

```bash
cd backend && php artisan serve --port=8010
cd backend && php artisan tinker --execute="echo App\Models\OtpVerification::latest()->value('code');"   # OTP
```

`backend/.env` uses sqlite (`database/database.sqlite`), `QUEUE_CONNECTION=sync`; Passport keys + personal client
already exist (don't regenerate). Missing sqlite file: `touch database/database.sqlite && php artisan migrate --seed`.

## Rules
- Never point local frontend/tools at the prod API for destructive flows (account delete, data wipes).
- Never add an OTP bypass, test code or magic number — read the code from your own DB.
