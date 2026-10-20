---
id: T-M9-10
title: HTTP and infra hardening: gzip, timeouts, readiness, Redis eviction, MariaDB config
milestone: M9
type: backend
status: todo
depends_on: [T-M9-07]
parallel_group: M9-B
touches: [backend-go/cmd/api/app.go, backend-go/cmd/api/main.go, backend-go/internal/platform/httpx, backend-go/internal/http/routes_health.go, backend-go/internal/platform/cache, backend-go/internal/platform/config, docker-compose.stage.yml, docker-compose.beta.yml, deploy/mariadb, deploy-stage.sh, deploy-beta.sh]
skills: []
verify: cd backend-go && go vet ./... && go test ./cmd/... ./internal/platform/... ./internal/http/... && make test-int PKG=./internal/platform/httpx/...
---

# T-M9-10 — HTTP and infra hardening: gzip, timeouts, readiness, Redis eviction, MariaDB config
**Model:** opus.

## Why
Gap 07 §5–§7.5: no compression (`/cycle/month?view=full` ≈ 119 KB of Persian JSON, which gzips 5–8×), unlimited Fiber
timeouts, no readiness probe for deploy waits, no global DB deadline, Redis with no `maxmemory` (under pressure the
default `noeviction` makes throttles and queue writes fail → 500s), MariaDB on defaults (128 MB buffer pool, no slow
log).

## Scope
1. Fiber compress middleware (gzip + brotli) for JSON responses > 1 KB; not for SSE/streams or already-compressed
   bodies. Check the stage/beta nginx includes do not compress `/api` too (no double compression); do **not** touch
   shared nginx files used by production.
2. Fiber `ReadTimeout` 15 s, `WriteTimeout` from env (default 60 s; check the longest AI/upload/export route and set
   the default above it), `IdleTimeout` 60 s.
3. Per-request DB deadline: request context gets a 5 s timeout, except an allow-list (AI, uploads, exports).
4. `GET /ready`: DB ping + Redis ping, 1 s timeout → 200 `{"status":"ok"}` or 503 with which check failed. `/up` stays
   liveness.
5. Redis client pool size and timeouts from env (`REDIS_POOL_SIZE`, `REDIS_DIAL_TIMEOUT`, `REDIS_READ_TIMEOUT`,
   `REDIS_WRITE_TIMEOUT`).
6. `TestCacheWritesHaveTTL`: scans the Go source for cache `Set` calls with a zero TTL outside an allow-list.
7. Compose overrides for **stage and beta only**: Redis `--maxmemory 256mb --maxmemory-policy volatile-lru` (keep
   appendonly); MariaDB conf mounted from `deploy/mariadb/ritme.cnf` (buffer pool from env, default 512M;
   `slow_query_log=1`, `long_query_time=0.2`, `performance_schema=ON`). The base `docker-compose.yml` and production
   files are not touched.
8. `deploy-stage.sh` / `deploy-beta.sh` wait on `/ready`. `deploy-beta.sh` currently has uncommitted changes: if it is
   still dirty when you start, leave it and note it in PROGRESS.md.

## Tests
- `Accept-Encoding: gzip` on a large JSON route → `Content-Encoding: gzip`, compressed size < 30 % of the original;
  small responses uncompressed.
- `/ready` → 503 when Redis is down (closed miniredis) or the DB ping fails; 200 when both are up.
- Integration: a `SELECT SLEEP(6)` through the request context is cancelled at ~5 s.
- The TTL scan test; timeout config unit test.

## Out of scope
- ETag/304 (T-M9-31), worker container (T-M9-12).

## Acceptance
- Tests green; `docker compose -f docker-compose.yml -f docker-compose.stage.yml config` and the beta equivalent
  render with the example env; PROGRESS.md cites D-132 for compression.
