# backend-go — Ritme API in Go (Fiber v3)

**This is THE backend for all new work** (decided 2026-09-23). Staging runs Go only (T-M2-28); production still runs
Laravel (`backend/`) until the cutover (T-M2-26/27), so `backend/` is **frozen — production maintenance only**, no new
features there. `backend-go/` is renamed to `backend/` at T-M2-27.

Plan, decisions and inventories: **`docs/go-migration/`** (start with `README.md`, then the inventory for your area).
Tasks: `tasks/M2/`. New endpoint → the `new-endpoint` skill. Run it locally → the `local-dev` skill.

Existing routes stay **byte-compatible** with Laravel: the web frontend and the Android app must not notice which
stack answered. When in doubt, read the PHP code and copy what it does, quirks included.

## Commands

```bash
make help                        # list targets
make test-db-up                  # MariaDB 11.4 + Redis (docker-compose.test.yml, ports 13317 / 16380)
make run                         # API on :8020 against the test stack (DB ritme_test; override DB_*/REDIS_* in .env)
make test                        # unit tests (integration tests skip)
make test-int PKG=./internal/x/... # integration tests (TEST_DB_DSN / TEST_REDIS_ADDR are set for you)
make vet                         # go vet ./...
make lint                        # golangci-lint v2 (.golangci.yml)
make fmt                         # gofmt + goimports
make sqlc                        # regenerate internal/<domain>/store from db/queries/<domain>/*.sql
make schema-diff                 # Laravel migrations vs goose migrations → identical schema (needs php + backend/vendor)
make contract ROUTES=<pattern>   # contract diff vs Laravel goldens (docs/go-migration/contract.md)
make contract-record ROUTES=<g>  # (re)record Laravel goldens from the contract stack
make docker                      # build ritme-backend-go:dev
```

Verify (what `/verify-all` runs): `go vet ./... && go test ./... && golangci-lint run`.

A local dev database with the goose baseline, a seeded user and OTP login without SMS (`SMS_PROVIDER=log`, code read
from `otp_verifications`) is a step-by-step recipe in `.claude/skills/local-dev/SKILL.md` and `README.md` → *Local dev*.

## Dependencies (parallel agents — read this)

Many agents edit `backend-go/` at the same time in one working tree. `go.mod` / `go.sum` are shared files.

- Prefer the libraries already chosen in `docs/go-migration/README.md` → *Libraries*. Adding anything else
  needs a reason in your task report.
- Add a dependency with **`go get <module>@<version>`** only (pin a version). It edits `go.mod`/`go.sum`
  surgically and is safe to run while others build.
- **Never run `go mod tidy` while other agents may be working** — it rewrites the whole `go.mod`/`go.sum`
  from *your* view of the tree and can drop requirements another agent just added (or fail on their
  half-written packages). Only the orchestrator runs it, once, when no task is in flight.
- `go get` leaves the new module marked `// indirect` until tidy runs. That is fine; do not hand-edit it.
- If `go.mod` changed under you (build says "missing go.sum entry"), re-run `go get <module>@<version>` for
  the module named in the error — don't revert the other agent's lines.
- `proxy.golang.org` can 403 from Iranian IPs; the Dockerfile falls through to `goproxy.io` and `direct`.
  Locally use `GOPROXY='https://proxy.golang.org|https://goproxy.io|direct'` if a download fails.

## Package layout

```
cmd/api/              main: config → db/redis → router → Fiber (CORS, proxy, body limit, /up, shutdown)
cmd/contract/         golden recorder + JSON-aware differ (T-M2-05)
internal/platform/    config, db, cache, clock, civildate, phpround, jsonx, validation, i18n, httpx
internal/enums/       generated enums + hand-written methods
internal/auth/        passport JWT, middleware, OTP, sms
internal/<domain>/    handlers + services per domain (content, profile, healthlog, reminder, cycle, …)
internal/http/        router.go (registry) + routes_<domain>.go — one file per domain
db/migrations/        goose        db/queries/  sqlc queries per domain
```

## Rules

### Routing
- **One routes file per domain**: `internal/http/routes_<domain>.go`, registering itself from `init()` with
  `Register("<domain>", func(r fiber.Router, d *Deps) { … })`. Never edit `router.go` or another domain's file
  to add routes; there is no central list.
- Inside your file, register **static paths before param paths** (`/articles/categories` before
  `/articles/:id`), the same order as `backend/routes/api.php`.
- GET automatically answers HEAD (Fiber). Don't register HEAD by hand.
- Versioned API: public routes live under `/api/v1/*`, the new admin API under `/api/admin/v1/*`.
- CORS, trusted proxies (X-Forwarded-For) and the 25 MB body limit are global in `cmd/api`; don't re-add them.

### Data access
- **sqlc only.** Queries live in `db/queries/<domain>/*.sql`; handlers and services never build SQL strings.
- **Schema-change rule (until T-M2-27):** Laravel migrations still own the prod schema. Every schema change needs a
  goose migration (`db/migrations/000NN_<name>.sql`, never edit `00001_baseline.sql`) **and** the mirror Laravel
  migration in `backend/database/migrations` in the same commit, so prod keeps working; `make schema-diff` must stay
  green. After T-M2-27 goose alone owns the schema. Details: `docs/go-migration/migrations.md`.
- DB timestamps are **Asia/Tehran wall-clock**. The DSN has `parseTime=true&loc=Asia%2FTehran`; **never** set
  the session `time_zone` and never convert to UTC before writing.
- Redis keys always go through `cache.Client.Key()` (prefix `ritme-go:`). Never read Laravel's PHP-serialized
  cache or queue entries.

### PHP compatibility
- PHP-compat behaviour (rounding, Carbon diffs, date casts, JSON escaping, envelopes, paginators, validation
  messages) comes **only from `internal/platform/*`**. Don't re-implement it inside a domain package; add or
  fix the helper instead.
- Laravel quirks are preserved (decimal-as-string, `[]` for empty maps, 201 vs 200, raw bilingual blobs, …).
- The contract diff (`make contract ROUTES=…`) must be green for every route you port.
- Any intentional difference needs an entry in **`docs/go-migration/deviations.md`** first (user OK), and the
  contract allow-list references its id.

### Sessions and 401 (carried over from backend/CLAUDE.MD — mandatory)
- Token lifetime is a fixed **365 days** (`PASSPORT_TOKEN_LIFETIME_DAYS`). Never shorten it.
- `POST /api/v1/auth/refresh-session` issues a new 365-day token only when fewer than
  `PASSPORT_REFRESH_WINDOW_DAYS` (30) days are left: issue first, then revoke the old one. Otherwise it returns
  `refreshed: false` and does nothing.
- An authentication 401 on `/api` is always JSON with `error_code` ∈ `token_revoked` | `token_expired` |
  `unauthenticated`. Clients wipe the session only on these, so **never** return 401 for anything else
  (use 403 / 422 / 429).
- Tokens are revoked only on logout, admin block, account deletion, and by refresh-session. No scheduled purge.
- The `ritme_auth` flag is set by the frontend route, not the API. Tokens never go into cookies.
- Passport keys (`storage/oauth-*.key`) are read from `STORAGE_PATH` (the `backend-storage` volume, 0600,
  uid 33). Go never generates, copies or ships keys; the image runs as uid 33 to read them.

### Multi-language (mandatory, no exceptions)
- The language list is **data** (`languages` table), not code. `fa` and `en` are just seeded rows. Never
  enumerate languages in code (`[]string{"fa","en"}`, `title["fa"]`, `Accept-Language` defaulting to `fa`, …).
- Resolve the request locale through the shared locale middleware/helper, read translatable JSON columns
  through the platform helper (fallback to the default language), and validate them with the platform rule
  builder (only the default language is required).
- Translatable columns are JSON keyed by language code; the key set grows with the `languages` table.
- `GET /languages` and `GET /languages/{code}/messages` are public; the frontend loads languages at runtime.
- `resources/translations/<code>/*.json` (embedded) is the seed copy of `frontend/messages/<code>/`. Resync with
  `cp ../frontend/messages/<code>/*.json resources/translations/<code>/` (and `php artisan translations:import` in
  `backend/` until cutover); then the `public` contract goldens and `internal/i18n/testdata/messages_*.json`
  (`TestBundle_MatchesLaravelGoldens`) must be re-recorded, since they pin the bundle content.

### Code
- SOLID, DRY, KISS, YAGNI. Small packages, explicit dependencies (`http.Deps`), no globals besides the route
  registry.
- Logging: `log/slog` JSON to stderr. Never log tokens, OTP codes, phone numbers in full or passwords.
- Errors wrap with `%w`; handlers return errors, the error handler maps them (see `platform/httpx`).
- Tests: `testify`; integration tests `t.Skip()` when `TEST_DB_DSN` / `TEST_REDIS_ADDR` are unset.
- After every Edit/Write of a `.go` file the hooks run `golangci-lint fmt` (gofmt + goimports) and then a lint gate
  on the file's package (`.claude/hooks/format.sh` → `go-lint.sh`): issues in that file come back as errors to fix
  immediately; compile errors of a half-written package are ignored (vet/test catch them).
