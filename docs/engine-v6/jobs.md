# Background jobs (queue v2)

T-M9-11. Code: `backend-go/internal/platform/queue` (wrapper over asynq on the shared Redis; README §6 and X-02: no
RabbitMQ, NATS or Kafka). On stage and beta the worker, scheduler and pollers run in the `backend-go-worker` container (`api worker`,
T-M9-12) and the API runs with `API_RUN_WORKERS=0`; elsewhere (`API_RUN_WORKERS=1`, the default) the API process runs
them itself. The worker serves `/up`, `/ready` and `/metrics` (incl. `asynq_queue_size`) on `METRICS_ADDR`.

## Queues

| Priority | asynq queue | Weight | Used by |
|---|---|---|---|
| `queue.Critical` | `<base>-critical` | 6 | user-facing latency (OTP SMS) |
| `queue.Default` | `<base>` | 3 | everything else that a user waits for indirectly |
| `queue.Low` | `<base>-low` | 1 | housekeeping (sweeps, purges, snapshots) |

`<base>` is `queue.NameFromPrefix(REDIS_PREFIX)` (`ritme-go` on every environment; stage and beta have their own
Redis). The default queue keeps the pre-v2 name, so tasks enqueued by an older binary still run after a deploy.

## API

```go
q := auth.Jobs(r, d.Cache, d.Logger) // the process queue, one per *fiber.App, started on listen, drained on shutdown

q.Register(queue.Job{Type: "x", Priority: queue.Low, MaxTries: 3,
	Backoff: []time.Duration{time.Minute, 5 * time.Minute}, Timeout: 2 * time.Minute,
	OnFailed: func(ctx context.Context, payload []byte, err error) { /* exhausted */ }}, handler)

q.Dispatch(ctx, "x", payload)                                    // now
q.DispatchWith(ctx, "x", payload, queue.Unique(time.Hour))       // dropped (ErrDuplicate) while a twin is locked
q.DispatchWith(ctx, "x", payload, queue.TaskID("x:2026-10-21"))  // natural-key id (ErrDuplicate while it exists)
q.DispatchIn(ctx, 10*time.Minute, "x", payload)                  // not before now+10m
q.DispatchAt(ctx, at, "x", payload)                              // not before at

q.Scheduler().Register("0 3 * * *", "x", payload, queue.Unique(time.Hour), queue.RunAtStart())
q.RegisterPeriodic(job, handler, "*/10 * * * *", opts...)        // Register + Scheduler().Register with payload {}
```

- **Cron specs** are standard 5-field (or `@every 1m`) on the **Asia/Tehran** wall clock. `queue.Next(spec, t)` gives
  the next fire time.
- **Leader key.** Only the process holding `<base>:scheduler:leader` (Redis `SET NX PX`, TTL 30 s, renewed every
  10 s with a compare-and-`PEXPIRE` script, deleted on shutdown) runs the `asynq.Scheduler`, so an entry is enqueued
  once per tick however many API / worker replicas run. Every other process retries the key on each renewal tick and
  takes over within the TTL when the leader dies. The task runs on whichever worker picks it up.
- **`RunAtStart`** fires the entry once when a process becomes leader (the old tickers ran at start-up), deduped by
  the entry's `Unique` ttl (1 min without one).
- **`QUEUE_CONNECTION=sync`** (tests, contract stack, local dev): Dispatch/DispatchIn/DispatchAt run the handler
  inline, now, once; `Unique`/`TaskID` are ignored; the scheduler fires entries from an in-process cron (no leader
  key). Without a cache (unit tests) the queue is always sync. On a router that is not the `*fiber.App`, `auth.Jobs`
  returns a never-started queue: nothing is scheduled.

## Handler rules

1. **Idempotent on a natural key.** Running a handler twice leaves the same state (each job has a test that runs it
   twice). Periodic jobs take their clock from the handler (`now()`), never from the payload.
2. **Timeout** set on every job (`Job.Timeout`, default 60 s); the context is cancelled when it passes.
3. **Retries with backoff** (`MaxTries`, `Backoff`); after the last try the task is archived, `Job.OnFailed` fires and
   `asynq_archived_total` moves. A periodic job whose next tick is the retry uses `MaxTries: 1`.
4. **Unique** on periodic entries with a ttl a little under the interval, so a slow run or a retry never lets ticks
   stack up.
5. **No personal data in errors or logs.** The queue logs `queue: job failed {job, final, error}`; handlers whose
   errors could carry data (files) swallow them.

## Metrics (private `METRICS_ADDR` listener, T-M9-07)

| Metric | Meaning |
|---|---|
| `asynq_queue_size{queue,state}` | tasks per state (`pending`, `active`, `scheduled`, `retry`, `archived`, `completed`), read with `asynq.Inspector` at scrape time |
| `asynq_archived_total{queue}` | tasks this process archived after their last try |
| `asynq_task_duration_seconds{type,result}` | one try of a job, `result` = `ok` \| `error` (both modes) |

## Catalogue

| Type | Queue | Schedule (Tehran) | Uniqueness | Tries / backoff / timeout | Idempotency key | Owner package |
|---|---|---|---|---|---|---|
| `send_otp_sms` | critical | on demand (OTP request) | — | 3 / 5 s, 15 s / 60 s | the payload (a retry resends the same code) | `internal/auth` |
| `partners:sync` | low | `*/10 * * * *` + at start | `Unique(9m)` | 2 / 1 m / 5 m | request id + state machine (a repeated status is a no-op) | `internal/partners` |
| `partners:purge` | low | `5 * * * *` + at start | `Unique(55m)` | 3 / 1 m, 5 m / 2 m | request id; only blobs still present and closed > 30 d | `internal/partners` |
| `telemed:booking_sweep` | default | `* * * * *` + at start | `Unique(50s)` | 2 / 10 s / 50 s | booking id; `ExpireHold` matches only a lapsed hold, `Complete` only a confirmed booking | `internal/telemed` |
| `sharelinks:purge` | low | `10 * * * *` + at start | `Unique(55m)` | 3 / 1 m, 5 m / 2 m | link id; wipes only non-null reports, deletes rows expired > 30 d | `internal/sharelinks` |
| `admin_metrics:snapshot` | low | `15 * * * *` + at start | `Unique(55m)` | 3 / 1 m, 5 m / 10 m | Tehran day (`metric_snapshots.day`, upsert) | `internal/metrics` |
| `files:sweep` | low | `20 */6 * * *` + at start | `Unique(5h)` | 1 / — / 30 m | blob path; removes only blobs with no row, older than 1 h | `internal/files` |
| `media:sweep` | low | `25 * * * *` + at start | `Unique(55m)` | 1 / — / 30 m | media id / file path; removes only abandoned, orphan or stray ones | `internal/media` |
| `safety:nightly_scenarios` | low | `0 3 * * *` | `Unique(1h)` | 1 / — / 60 s | Tehran day (pure scenarios; the run replaces the last) | `internal/safety` |

Notes:

- `files:sweep` and `media:sweep` read the local storage volumes (`STORAGE_PATH`, `MEDIA_STORAGE_PATH`): a process
  that runs the worker must mount them (T-M9-12).
- `safety:nightly_scenarios` keeps its result in the memory of the process that ran it; the API also runs the set once
  at start-up so its own dashboard row is filled. Persisting the result is T-M10-24.
- Not migrated, on purpose (replica-safe pollers with `SKIP LOCKED` / leases): the labs runner and its job sweep
  (`internal/labs/worker.go`), the health-record extract runner (`internal/healthrecord/extract/runner.go`) and the
  learning dispatch poller (`internal/learning/dispatch.go`).

New jobs add a row here in the same change.
