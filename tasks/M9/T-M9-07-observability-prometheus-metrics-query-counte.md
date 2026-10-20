---
id: T-M9-07
title: Observability: Prometheus metrics, query counter, request id, pprof listener
milestone: M9
type: backend
status: todo
depends_on: []
parallel_group: M9-A
touches: [backend-go/cmd/api/app.go, backend-go/cmd/api/main.go, backend-go/internal/platform/db, backend-go/internal/platform/cache, backend-go/internal/platform/httpx, backend-go/internal/platform/metrics, backend-go/internal/platform/config, backend-go/go.mod, backend-go/go.sum]
skills: []
verify: cd backend-go && go vet ./... && go test ./cmd/... ./internal/platform/... && make test-int PKG=./internal/platform/db/...
---

# T-M9-07 — Observability: Prometheus metrics, query counter, request id, pprof listener
**Model:** opus.

## Why
There is no `/metrics`, no pprof, no request id, no per-request query count and no cache hit ratio (gap 07 §5, §7.6).
Every M9 performance task needs these numbers to prove its budget, and the rollout needs shadow-diff and cache-verify
counters.

## Scope
1. `internal/platform/metrics`: Prometheus registry + helpers. Add `github.com/prometheus/client_golang` with
   `go get …@<current stable>` (never `go mod tidy`).
2. Second listener on `METRICS_ADDR` (default `127.0.0.1:9090`, empty = off) serving `/metrics` and
   `/debug/pprof/*`. Nothing on the public Fiber app.
3. HTTP middleware: `http_request_duration_seconds{route,method,status}` using the **route template** (Fiber
   `c.Route().Path`), buckets 1 ms … 5 s.
4. DB wrapper implementing sqlc's `DBTX` for both `*sql.DB` and `*sql.Tx` paths: `db_query_duration_seconds{query}`
   (sqlc query name from the `-- name:` comment, `raw` otherwise) and a per-request counter carried in the context →
   `db_queries_per_request{route}` and a `queries` field in the access log. Preserves `sql.ErrNoRows`, tx semantics and
   context cancellation.
5. `X-Request-ID`: accept a valid incoming id (≤ 64 chars, `[A-Za-z0-9-]`), else generate; echo it; add it to the
   access log and the slog context.
6. Warn logs: request > 500 ms (route, status, queries), query > 200 ms (name).
7. `cache_requests_total{cache,result}` helpers in `platform/cache` (hit/miss/error), used by the cycle cache now.

## Tests
- Route label is the template (`/api/v1/cycle/date/:date`), never the raw path.
- Integration: the per-request counter equals the number of DB calls a known route makes.
- `GET /metrics` on the public app → 404; the metrics listener serves it.
- Request id is echoed, generated when invalid, present in the log line.
- The DB wrapper passes the existing integration suite of one package unchanged (pick `internal/cycle/...`).

## Out of scope
- Dashboards/alerting rules, OpenTelemetry tracing.

## Acceptance
- Metrics and pprof available on the private listener only; tests green; README §6 metric names used verbatim.
