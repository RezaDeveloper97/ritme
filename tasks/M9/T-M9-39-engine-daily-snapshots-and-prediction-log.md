---
id: T-M9-39
title: Engine daily snapshots and prediction log
milestone: M9
type: backend
status: todo
depends_on: [T-M9-11, T-M9-32]
parallel_group: M9-C
touches: [backend-go/internal/cycle/snapshots, backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/cycle/snapshots.sql, backend-go/internal/cycle/periods, backend-go/internal/platform/queue, docs/engine-v6/jobs.md]
skills: []
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && go test ./internal/cycle/snapshots/... && make test-int PKG="./internal/cycle/snapshots/... ./db/migrations/..."
---

# T-M9-39 — Engine daily snapshots and prediction log
**Model:** opus. Runs alone in a batch (migration).

## Why
SPEC §5.6 KPIs (engine MAE, ±2-day share, confidence and data-quality distributions) and §5.2 user monitoring need the
engine's output over time. Today the metrics package re-derives MAE from stored lengths with raw SQL and includes
unconfirmed rows (gap 02 §5). DECISIONS.md A-05, A-06, D-137.

## Scope
1. Migration + Laravel twin: `engine_daily_snapshots (user_id, day, main_phase, subphase, confidence, data_quality,
   warnings_mask INT UNSIGNED, days_late SMALLINT NULL, mode, engine_rev, created_at, PK(user_id, day), KEY(day))` and
   `engine_prediction_log (id, user_id, cycle_start DATE, kind ENUM('first','last'), predicted_next DATE,
   basis_cycles TINYINT, ovulation_source, confidence, actual_start DATE NULL, error_days SMALLINT NULL, created_at,
   UNIQUE(user_id, cycle_start, kind))`.
2. Nightly scheduled job (T-M9-11 scheduler, queue `low`, unique per day): active users (A-06: any session, log or
   event in the last 30 days) in batches of 500 via `inputs.LoadMany` + `Prepare` + `Resolve(today in the user's stored
   zone)`; idempotent upsert; runs in the worker.
3. Prediction log: after a real period start commits, a job (with `rev_before`/`rev_after`) records `last` (prediction
   valid just before the write) and fills `actual_start`/`error_days` for that cycle, then records `first` for the new
   cycle from the new state. Skipped cycles and the declared anchor never count as actual starts.
4. Retention: snapshots 400 days (purge job), prediction log forever.
5. Catalogue entries in `docs/engine-v6/jobs.md`.

## Tests
- Job over 3 active + 1 inactive user → 3 rows; rerun → no duplicates; a user in Europe/Berlin gets the Berlin date.
- Prediction log: start 07-01 (first: 07-29), real start 08-02 → last + actual + error 4; edit of an old period does not
  create rows; skip → no actual.
- Migration up/down; schema-diff green.

## Out of scope
- KPI computation and dashboards (T-M10-30, T-M10-34).

## Acceptance
- Tests green; PROGRESS.md cites D-137.
