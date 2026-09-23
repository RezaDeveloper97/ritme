# Laravel → Go Fiber v3 migration (milestone M2)

Rewrite `backend/` (Laravel 12, ~27k PHP LOC, 74 `/api/v1` routes, Blade admin) as `backend-go/` (Go + Fiber v3)
without changing the web frontend, the Android app, or the production data. Tasks: `tasks/M2/` (run with `/next-task`).

**Status (2026-09-23):** **dev on Go** — all new backend work goes to `backend-go/`; `backend/` is frozen
(production fixes + mirror migrations only). **Staging on Go** — every `/api/v1` group and the admin API
(`admin-web` under `/panel`) are served by backend-go; Laravel is being switched off there (T-M2-28).
**Production on Laravel** until the prod rollout and cutover (T-M2-26/27). Parity: 986/986 contract cases
([parity-report.md](parity-report.md)). Local dev: `backend-go/README.md` → *Local dev* (skill `local-dev`).

Inventories (read the relevant one before starting a task):
- [api-inventory.md](api-inventory.md) — every route, envelope, error family, status code, date format, client usage.
- [domain-inventory.md](domain-inventory.md) — schema, models/casts, enums, services (cycle engines, pregnancy,
  messages, home), PHP-vs-Go pitfalls, suggested package layout.
- [infra-auth-admin-inventory.md](infra-auth-admin-inventory.md) — Passport token internals, OTP/SMS, admin panel,
  env vars, deploy, cutover risks.
- [deviations.md](deviations.md) — every intentional behaviour difference between Laravel and Go (needs user OK).

## Decisions

| Topic | Decision |
|---|---|
| Cutover | **Strangler.** Go runs beside Laravel on the same MariaDB + Redis; nginx moves path groups to Go one domain at a time (stage first, then prod). Rollback = flip the path group back. |
| Location | `backend-go/` during M2; renamed to `backend/` when Laravel is removed (T-M2-27). |
| DB | **sqlc** (typed queries) + **goose** (baseline from the current schema, no data migration). DSN `parseTime=true&loc=Asia%2FTehran`; never set the session `time_zone` (timestamps are stored as Tehran wall-clock). |
| Migrations | Laravel owns migrations until cutover; **schema is frozen during M2**. Any schema change needs a Laravel migration *and* a goose migration. After T-M2-27 goose owns them. |
| Admin | Blade panel is dropped. New **Go admin API** under `/api/admin/v1` + a separate **Next.js app `admin-web/`** served on `adpanell.ritme.app`. Admins re-login once (PHP sessions can't be shared); bcrypt `$2y$` hashes carry over. |
| Auth | Go validates and issues **Passport-compatible** RS256 JWTs with the same key pair and `oauth_*` rows → no user is logged out, both stacks accept each other's tokens during the strangler period. |
| Contract | **Semantic JSON equality** (normalised key order, `\/`, `\uXXXX`) + exact status codes + relevant headers, checked by the contract harness (`backend-go/cmd/contract`) against goldens recorded from Laravel with a fixed clock. |
| Quirks | Laravel output quirks are **preserved** (decimal-as-string, plain `date` cast → previous-day UTC, `[]` for empty maps, raw bilingual blobs, 201 vs 200, pretty-printed framework errors). Fixes go to `deviations.md` first. |
| Cache/queue | Redis with key prefix `ritme-go:`. OTP SMS job via **asynq** (3 tries, backoff 5s/15s). PHP-serialized cache/queue entries are never read by Go. |
| Libraries | Fiber v3, go-sql-driver/mysql, sqlc, goose, golang-jwt/v5, go-redis/v9, asynq, x/crypto/bcrypt, bluemonday, slog, testify. Tests run against MariaDB 11.4 + Redis in docker (`backend-go/docker-compose.test.yml`). |

## Package layout (`backend-go/`)

```
cmd/api/            main: config → db/redis → router → Fiber listen
cmd/contract/       golden recorder + JSON-aware differ
cmd/enumgen/        PHP enum → Go generator (dev tool)
internal/platform/  config, db, cache, clock, civildate, phpround, jsonx, validation, i18n, httpx (envelopes, errors, paginators, rate limit)
internal/enums/     generated enums + hand-written logic methods
internal/auth/      passport JWT, middleware, OTP, sms
internal/<domain>/  content, profile, healthlog, reminder, cycle (metrics, resolver, legacy, view, periods), pregnancy, messages, home, admin
internal/http/      routes_<domain>.go — one registration file per domain (keeps parallel tasks conflict-free)
db/migrations/      goose
db/queries/         sqlc queries per domain
```

## Strangler route groups (nginx `deploy/go-routes.inc`)

Order of moving groups to Go: `content` (languages, info, banners, articles, phase-content, /storage) →
`reminders` + `health-logs` → `profile` → `pregnancy` → `cycle` → `messages` → `home` → `auth` (last, once
token interop is proven in both directions) → admin (switch `adpanell` to `admin-web`).

## Risks (see infra inventory §Risks for detail)
1. Wrong 401 `error_code` mapping → clients log themselves out.
2. Key file access (0600, uid 33) — Go container must run as uid 33; Go never generates keys.
3. nginx upstream pinned to container name `ritme-backend-1` — Go gets its own pinned name (`ritme-backend-go-1`).
4. Two cycle engines disagree by design — both are ported exactly; golden parity is the gate.
5. Time: Asia/Tehran, Saturday week start, Carbon 3 float `diffInDays`, PHP `round()` half-away-from-zero.
6. Queue drain before OTP moves to Go.
