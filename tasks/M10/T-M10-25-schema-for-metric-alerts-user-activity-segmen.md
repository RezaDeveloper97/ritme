---
id: T-M10-25
title: Schema for metric alerts, user activity, segments, reference codes, audit chain and safety runs
milestone: M10
type: backend
status: todo
depends_on: [T-M10-01]
parallel_group: M10-D
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/sqlc.yaml, backend-go/db/queries/metrics, backend-go/internal/metrics/store, backend-go/db/queries/admin, backend-go/db/queries/safety, backend-go/db/queries/auth]
skills: []
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && make test-int PKG=./db/migrations/...
---

# T-M10-25 — Schema for metric alerts, user activity, segments, reference codes, audit chain and safety runs
**Model:** opus. Runs alone in a batch (migration + `make sqlc`).

## Why
SPEC §5.1, §5.2, §5.6 and §5.8 need storage that does not exist: automatic alerts (`Alerts` is always empty,
`metrics.go:72-77,150`), a per-user activity record for DAU/MAU from sessions and loggers per day, a nightly user
directory projection (the directory aggregates every user's 30 days of logs on each page, `admin/users/v2.go:75-86`),
saved segments, a non-reversible reference code (today `RT-%04X` of the id, `v2.go:30-31`), an append-only audit log
(convention only today) and persisted safety runs (gap 06 §2.4–§2.8, §4.3, §4.5; gap 04 row 7.4). One migration keeps
the `make sqlc` churn to one task so T-M10-24 and T-M10-30 … T-M10-33 run in parallel. DECISIONS.md A-06, A-08, A-09.

## Scope
1. Goose migration (next free number) + Laravel twin (`DB::unprepared` for triggers):
   - `user_activity_days (user_id BIGINT UNSIGNED, day DATE, session TINYINT(1) NOT NULL DEFAULT 0, logged TINYINT(1)
     NOT NULL DEFAULT 0, PRIMARY KEY (day, user_id), KEY (user_id, day))` — FK users ON DELETE CASCADE;
   - `metric_alert_rules (id, metric_key VARCHAR(64) UNIQUE, comparator VARCHAR(16) (gt | lt | drop_vs_mean |
     rising_days), threshold DECIMAL(12,4), window_days SMALLINT UNSIGNED, min_volume INT UNSIGNED NULL, enabled
     TINYINT(1), created_at, updated_at)` seeded with SPEC §5.1: `ocr_correction_rate gt 0.12`, `message_ctr
     drop_vs_mean 0.30 (7 days, min_volume 200 shown/day)`, `rui_users rising_days 7`, `engine_mae gt 2.5`, plus
     `safety_selftest_failed gt 0`;
   - `metric_alerts (id, rule_id FK, subject VARCHAR(64) NOT NULL DEFAULT '', day DATE, value DECIMAL(12,4), threshold
     DECIMAL(12,4), state VARCHAR(12) (open | resolved), open_flag TINYINT(1) AS (IF(state = 'open', 1, NULL)) STORED,
     opened_at, resolved_at, UNIQUE (rule_id, subject, open_flag), KEY (state, opened_at))`;
   - `admin_user_activity (user_id PK FK users ON DELETE CASCADE, last_activity_at TIMESTAMP NULL, log_days_30
     TINYINT UNSIGNED, log_days_recent14 TINYINT UNSIGNED, log_days_prior14 TINYINT UNSIGNED, churn VARCHAR(8), mode
     VARCHAR(16), plan VARCHAR(16), data_quality VARCHAR(16) NULL, main_warning VARCHAR(48) NULL, updated_on DATE,
     KEY (churn), KEY (plan), KEY (mode), KEY (data_quality), KEY (last_activity_at))`;
   - `admin_segments (id, name VARCHAR(80) UNIQUE, filters JSON NOT NULL, created_by, created_at, updated_at)`;
   - `users.ref_code VARCHAR(12) NULL UNIQUE`;
   - `admin_audit_logs.prev_hash CHAR(64) NULL`, `row_hash CHAR(64) NULL`; `admin_audit_chain_head (id TINYINT
     UNSIGNED PRIMARY KEY CHECK (id = 1), last_id BIGINT UNSIGNED NULL, last_hash CHAR(64) NULL, genesis_last_id
     BIGINT UNSIGNED NULL, genesis_hash CHAR(64) NULL)` with its one row seeded (the writer locks this row
     instead of the log's tail, so inserts never take gap locks on the log); triggers `BEFORE UPDATE` and
     `BEFORE DELETE` on `admin_audit_logs` and `clinical_config_approvals` →
     `SIGNAL SQLSTATE '45000'` (FK actions do not fire triggers, so neither table may gain an FK that updates rows);
   - `safety_scenario_runs (id, run_id CHAR(26), scenario VARCHAR(64), passed TINYINT(1), detail JSON NULL,
     duration_ms INT UNSIGNED, created_at, KEY (created_at), KEY (scenario, created_at))`;
   - indexes: `daily_health_logs (log_date, user_id)`, `cycle_histories (period_start_date)`, `users (created_at)`.
2. sqlc queries for each table (upserts keyed on natural keys, open/resolve alert, segment CRUD, ref-code lookup,
   activity marks `INSERT … ON DUPLICATE KEY UPDATE session = 1` / `logged = 1`, run inserts and reads). The metric
   tables get a new `db/queries/metrics` package registered in `sqlc.yaml` (output `internal/metrics/store`); the
   existing raw-SQL aggregation in `internal/metrics` stays as is.
3. If `make schema-diff` does not compare triggers, extend it to (`information_schema.TRIGGERS`) or add an integration
   test that both migrations create the same triggers.

## Tests
- Migration up/down; UPDATE and DELETE on `admin_audit_logs` and `clinical_config_approvals` fail; deleting an admin
  or user still works through FK actions; `open_flag` allows one open alert per (rule, subject) and many resolved;
  EXPLAIN shows the new indexes used by `WHERE log_date >= ?` and `WHERE period_start_date BETWEEN ? AND ?`.

## Out of scope
- Every writer and reader (T-M10-24, T-M10-30 … T-M10-33).

## Acceptance
- Verify passes; `make schema-diff` green (triggers included).
