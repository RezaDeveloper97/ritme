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
| `engine:snapshot_nightly` | low | `30 3 * * *` | `Unique(23h)` | 3 / 10 m, 30 m / 30 m | (user, user-local day): `engine_daily_snapshots` PK, upsert | `internal/cycle/snapshots` |
| `engine:snapshot_purge` | low | `50 3 * * *` | `Unique(23h)` | 3 / 10 m, 30 m / 10 m | Tehran day − 400; deletes only older rows, in chunks of 5 000 | `internal/cycle/snapshots` |
| `engine:prediction_log` | low | on demand (a committed real period start, `periods.StartHook`) | `TaskID(engine:prediction_log:<uid>:<rev_after>)` | 3 / 1 m, 5 m / 60 s | (user, cycle_start, kind): `first` insert-once, `last` upsert, actual/error recomputed | `internal/cycle/snapshots` |
| `admin.audit_verify` | low | `40 3 * * *` | `Unique(23h)` | 1 / — / 10 m | read-only walk of `admin_audit_logs` by id (plus the one-off genesis, a no-op once `genesis_last_id` is set) | `internal/admin/auditchain` |
| `messages.rollup` | low | `20 0 * * *` | `Unique(23h)` | 3 / 5 m, 30 m / 10 m | (Tehran day, rule, version, variant, slot, event): `message_event_daily` PK, upsert overwrites the count | `internal/messages/events` |
| `messages.prune` | low | `45 3 * * *` | `Unique(23h)` | 3 / 10 m, 30 m / 30 m | Tehran day − 180; backfills missing rollup keys of the doomed rows (existing counts kept), then deletes older raw rows in batches of 5 000 | `internal/messages/events` |
| `messages.inbox_sweep` | low | `10 4 * * *` | `Unique(23h)` | 3 / 10 m, 30 m / 30 m | (user, rule) open row: re-evaluation writes only what changed; archive matches only resolved / acknowledged / noted rows with `last_at` before Tehran midnight of today − 29, in chunks of 5 000 | `internal/messages/inbox` |
| `messages.push_slot` | low | `0 * * * *` | `Unique(55m)` | 2 / 5 m / 20 m | (user, user-local day): the push slot's `shown` row (slot cap 1 a day, 3 in 7 days) is written in the same transaction as the notification | `internal/messages/inbox` |

Notes:

- `files:sweep` and `media:sweep` read the local storage volumes (`STORAGE_PATH`, `MEDIA_STORAGE_PATH`): a process
  that runs the worker must mount them (T-M9-12).
- `safety:nightly_scenarios` keeps its result in the memory of the process that ran it; the API also runs the set once
  at start-up so its own dashboard row is filled. Persisting the result is T-M10-24.
- `engine:snapshot_nightly` (T-M9-39, D-137; A-06): active users = any token issued/refreshed, session or logged day
  (`user_activity_days`), daily log, v2 log entry, period row or analytics event in the last 30 days; pages of 500 by
  user id through `inputs.LoadMany` + `engine.Prepare` + `Resolve` on each user's local day (stored
  `user_profiles.timezone`, Tehran without one). Rows hold codes only (phase, subphase, confidence, data_quality,
  warning bits, days_late, mode, engine_rev).
- `engine:prediction_log` (T-M9-39, A-05): payload = ids only (`user_id`, `period_id`, `rev_before`, `rev_after`).
  `last` = the state without the new start resolved the day before it (anchored on a real start only: the declared
  onboarding anchor and roll-forward references never count), then `actual_start` / `error_days` on that cycle's rows;
  `first` = the new state resolved on the start day. An edit (`Update`), a backfilled older period, an estimate or a
  skip writes nothing.
- Not migrated, on purpose (replica-safe pollers with `SKIP LOCKED` / leases): the labs runner and its job sweep
  (`internal/labs/worker.go`), the health-record extract runner (`internal/healthrecord/extract/runner.go`) and the
  learning dispatch poller (`internal/learning/dispatch.go`).
- `admin.audit_verify` (T-M10-33, D-182; SPEC §5.8): runs genesis if it has not run (hashes the pre-chain rows oldest
  first into `admin_audit_chain_head.genesis_hash` / `genesis_last_id`; the first chained `httpadmin.Audit` write does
  the same), then verifies the whole hash chain. A break sets the gauge `admin_audit_chain_broken` to 1, counts
  `admin_audit_chain_verify_total{result="broken"}` and logs `alert=admin_audit_chain_broken` with the row id and
  reason; every run logs `admin audit chain anchor {head_id, head_hash, checked}` as the off-database anchor (compare
  it with yesterday's to catch a rewrite of the whole tail). No row content reaches the log.

- `messages.rollup` / `messages.prune` (T-M10-06, D-163; M-13): raw `message_events` rows are kept 180 days, the daily
  rollup `(day, rule, version, variant, slot, event) → count` forever. The rollup counts yesterday (Tehran) and can be
  re-run at will; a legacy row without a version counts as version 0, without a slot under the rule's placement. Prune
  first adds to the rollup every key its rows make that the rollup lacks (`ON DUPLICATE KEY UPDATE count = count`, so a
  day already pruned in part never shrinks), then deletes oldest first, 5 000 per statement, at most 2 000 statements
  per run. The admin rules list reads the rollup for the 6 days before today plus today's raw rows. Not a job: the
  message events writer (`events.Writer`) is a buffered channel (10 000) in each API process, flushed every 1 s or 500
  rows as multi-row INSERTs of ≤ 100 rows, drained before shutdown (`OnPreShutdown`, it runs whatever
  `API_RUN_WORKERS` says); a full buffer inserts synchronously. Metrics: `messages_events_queue_depth`,
  `messages_events_flush_seconds`, `messages_events_sync_fallback_total`, `messages_events_write_errors_total`.

- `messages.inbox_sweep` / `messages.push_slot` (T-M10-07, D-164; SPEC §5.4.1, M-09, M-16): the sweep pages the
  users with an open `message_inbox` row (500 per page), re-evaluates each one's `alerts_inbox` slot on her own local
  day (a row whose condition no longer holds over known facts is resolved; a failing user is skipped and counted in
  `messages_inbox_job_user_errors_total{job}`), then archives resolved, acknowledged and added-to-visit-note rows whose
  last hit is 30 or more Tehran days old. The push slot runs hourly and picks the users active in the last 30 days
  (A-06, the same definition as `engine:snapshot_nightly`) whose stored zone puts them in 09:00–09:59 local; it
  selects the `push` slot once with its caps (1 a local day, 3 in 7 local days) and quiet hours (23–08), gates the
  winner through the user's notification preferences (tips → `articles`, health kinds → `daily_log`, her quiet
  hours) and writes one `user_notifications` row (`type = message_push`) whose title and body are the neutral
  «یادآور ریتمی» line in every active language — never the message text — with the CTA path as `action_url` and only
  ids in `data`. No real web push yet (M-09). Metrics: `messages_push_slot_sent_total`, `inbox_missing_evidence_total`.

New jobs add a row here in the same change.
