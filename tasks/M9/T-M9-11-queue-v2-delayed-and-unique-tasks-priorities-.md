---
id: T-M9-11
title: Queue v2: delayed and unique tasks, priorities, scheduler; tickers migrated
milestone: M9
type: backend
status: todo
depends_on: [T-M9-07]
parallel_group: M9-B
touches: [backend-go/internal/platform/queue, backend-go/internal/auth/module.go, backend-go/internal/auth/lifecycle.go, backend-go/internal/http/routes_admin_core.go, backend-go/internal/http/routes_admin_safety.go, backend-go/internal/http/routes_files.go, backend-go/internal/http/routes_media.go, backend-go/internal/http/routes_partners.go, backend-go/internal/http/routes_sharelinks.go, backend-go/internal/http/routes_telemed.go, backend-go/internal/partners/service.go, backend-go/internal/telemed/booking_service.go, backend-go/internal/sharelinks/service.go, backend-go/internal/metrics/metrics.go, backend-go/internal/files/sweep.go, backend-go/internal/media/sweep.go, backend-go/internal/safety/scenarios.go, docs/engine-v6/jobs.md]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/platform/queue/... ./internal/auth/... ./internal/partners/... ./internal/telemed/... ./internal/sharelinks/... ./internal/metrics/... ./internal/safety/... && make test-int PKG=./internal/platform/queue/...
---

# T-M9-11 — Queue v2: delayed and unique tasks, priorities, scheduler; tickers migrated
**Model:** opus.

## Why
The asynq wrapper only supports immediate `Dispatch` (`internal/platform/queue/queue.go:129-215`): no delay, no
uniqueness, one queue, no scheduler. Ten periodic loops run as in-process tickers with no leader lock, and the safety
nightly results live in process memory (gap 07 §3). The nightly engine snapshot (T-M9-39), the message rollups and
the KPI jobs (M10) need a real scheduler, and the worker process (T-M9-12) needs jobs that can move out of the API.

## Scope
1. Wrapper v2: `DispatchIn`, `DispatchAt`, options `Unique(ttl)` and `TaskID`, three weighted queues
   (`critical` 6 / `default` 3 / `low` 1; OTP → critical), archived-task count via `asynq.Inspector` exported as
   metrics (`asynq_queue_size{queue,state}`, `asynq_archived_total`, task duration histogram).
2. `Scheduler.Register(cron, type, payload, opts)` over `asynq.Scheduler`, Tehran timezone, guarded by a Redis leader
   key (`SET NX` with renewal) so only one process schedules even if more replicas appear.
3. Migrate the tickers to scheduler entries + handlers: partners sync/purge, telemed booking expiry, sharelinks purge,
   admin metrics snapshot, files sweep, media sweep, safety nightly. Keep the labs runner, health-record extract runner
   and learning dispatch pollers as they are (`SKIP LOCKED` / lease, replica-safe). Behaviour of every job unchanged.
4. Handler rules: idempotent on natural keys, timeouts set, retries with backoff; documented in
   `docs/engine-v6/jobs.md` (catalogue: type, queue, schedule, uniqueness, idempotency key, owner package).
5. `QUEUE_CONNECTION=sync` keeps working for tests and local dev.

## Tests
- miniredis/asynq: a unique task is deduplicated within its TTL; `DispatchIn` is not processed early; a scheduled entry
  is enqueued once per tick even with two scheduler instances (leader key); a failing task retries then archives and the
  gauge moves.
- Each migrated job: existing tests still green; one test per job proving the handler is idempotent (run twice → same
  state).

## Out of scope
- Persisting safety results (T-M10-24), the worker container (T-M9-12), new jobs (their own tasks).

## Acceptance
- No `time.NewTicker` loop left for the migrated jobs; jobs.md complete; tests green.
