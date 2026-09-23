---
id: T-M2-02
title: backend-go skeleton, Fiber v3 app and dev tooling
milestone: M2
type: backend
status: done
depends_on: []
parallel_group: M2-A
touches: [backend-go/go.mod, backend-go/go.sum, backend-go/cmd/api, backend-go/internal/platform/config, backend-go/internal/platform/db/db.go, backend-go/internal/platform/cache, backend-go/internal/http/router.go, backend-go/internal/http/routes_health.go, backend-go/Dockerfile, backend-go/.dockerignore, backend-go/Makefile, backend-go/.golangci.yml, backend-go/docker-compose.test.yml, backend-go/CLAUDE.md, backend-go/README.md, .claude/skills/verify-all/SKILL.md, .claude/hooks/format.sh]
skills: [verify-all]
verify: cd backend-go && go vet ./... && go test ./... && golangci-lint run && docker build -t ritme-backend-go:dev .
---

# T-M2-02 — backend-go skeleton, Fiber v3 app and dev tooling

## Why
Every later M2 task needs a running Go service with config, DB, Redis, a router that parallel tasks can extend
without editing the same file, a test DB, and a verify command. See docs/go-migration/README.md (package layout).

## Scope
1. `backend-go/` Go module, Go 1.25, Fiber **v3**.
2. `internal/platform/config`: env parity with the Laravel names (APP_ENV, APP_DEBUG, APP_URL, APP_TIMEZONE=Asia/Tehran,
   DB_*, REDIS_*, CORS_ALLOWED_ORIGINS, PASSPORT_TOKEN_LIFETIME_DAYS, PASSPORT_REFRESH_WINDOW_DAYS, SMS_*,
   KAVENEGAR_*, SMSIR_*, TELEGRAM_*, SWAGGER_USER/PASSWORD) + Go-only (`HTTP_ADDR`, `STORAGE_PATH` = the mounted
   Laravel `storage/`, `REDIS_PREFIX=ritme-go:`). Fail fast on missing required vars.
3. `internal/platform/db/db.go`: MariaDB pool, DSN `parseTime=true&loc=Asia%2FTehran&charset=utf8mb4&collation=utf8mb4_unicode_ci`;
   **never** set the session `time_zone`. `internal/platform/cache`: go-redis v9 client with key prefix.
4. `cmd/api`: slog JSON to stderr, Fiber app (ProxyHeader X-Forwarded-For, trusted proxies, 25 MB body limit),
   CORS matching `config/cors.php` (api/*, echo origin, `Vary: Origin`, preflight 204, max-age 3600, no credentials),
   graceful shutdown, `/up` 200.
5. `internal/http/router.go`: a registry. Each domain file `routes_<domain>.go` registers itself from `init()`, so
   parallel tasks never edit a shared file. Static-before-param order is kept inside each domain file; GET also
   answers HEAD.
6. Dockerfile: multi-stage, static binary, distroless/alpine, runs as **uid 33** (to read the 0600 Passport keys on
   `backend-storage`), port 80, HEALTHCHECK on `/up`.
7. `docker-compose.test.yml`: MariaDB 11.4 + Redis for integration tests (env `TEST_DB_DSN`). Makefile: `run`,
   `test`, `test-int PKG=<pkgs>`, `lint`, `fmt`, `sqlc`, and **placeholder targets that later tasks fill by adding
   scripts, not by editing the Makefile**: `schema-diff` → `scripts/schema-diff.sh` (T-M2-03),
   `contract ROUTES=` / `contract-record ROUTES=` → `go run ./cmd/contract …` (T-M2-05). Placeholders exit 0 with a
   "not implemented yet" note while the script is absent.
8. `backend-go/CLAUDE.md`: carry over the rules from `backend/CLAUDE.MD` (session/401 rules, multi-language rules,
   versioned API) + Go rules (package layout, one routes file per domain, sqlc only — no hand-built SQL strings in
   handlers, PHP-compat helpers only from `platform/*`, contract diff must be green, deviations need an entry in
   `docs/go-migration/deviations.md`). Point to `docs/go-migration/`.
9. Tooling: `verify-all` skill gets a **backend-go** row (`go vet`, `go test`, `golangci-lint`), skipped when
   `backend-go/` is absent; `.claude/hooks/format.sh` runs `gofmt -w` on `backend-go/**/*.go`.

## Out of scope
Any API endpoint except `/up`. Schema/sqlc (T-M2-03). PHP-compat helpers (T-M2-04).

## Acceptance
- `make run` against the local compose DB serves `/up` 200 and logs JSON; the CORS preflight for
  `https://web.ritme.app` returns the same headers as Laravel.
- `verify:` green; the image builds and runs as uid 33.
- A test proves two domain route files register independently.
