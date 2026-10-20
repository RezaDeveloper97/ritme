---
id: T-M9-24
title: Schema for engine v6 inputs and revisions
milestone: M9
type: backend
status: todo
depends_on: [T-M9-01, T-M9-23]
parallel_group: M9-C
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/cycle, backend-go/db/queries/profile, backend-go/db/queries/postpartum, backend-go/internal/enums, backend-go/sqlc.yaml]
skills: []
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && go test ./internal/enums/... && make test-int PKG=./db/migrations/...
---

# T-M9-24 — Schema for engine v6 inputs and revisions
**Model:** opus. Runs alone in a batch (migration + `make sqlc`).

## Why
v6 needs storage that does not exist: the declared onboarding anchor (I-01), the device timezone (X-07), skipped
cycles (E-15), the breastfeeding flag (I-04) and the per-user revision that makes caching DB-authoritative (X-02,
README §3). One migration keeps the `make sqlc` churn to one task.

## Scope
1. Goose migration (next free number; `00094_app_modules.sql` may still be uncommitted — check `ls` and `git status`)
   + Laravel twin (`backend/database/migrations`, same columns/types/indexes):
   - `user_profiles.onboarding_last_period_start DATE NULL`, `user_profiles.timezone VARCHAR(64) NULL`;
   - `cycle_skips (id BIGINT UNSIGNED AUTO_INCREMENT PK, user_id BIGINT UNSIGNED NOT NULL FK users ON DELETE CASCADE,
     expected_start DATE NOT NULL, created_at, updated_at, UNIQUE(user_id, expected_start))`;
   - `postpartum_profiles.breastfeeding TINYINT(1) NULL`;
   - `user_revisions (user_id BIGINT UNSIGNED PK FK users ON DELETE CASCADE, engine_rev BIGINT UNSIGNED NOT NULL
     DEFAULT 0, analysis_rev BIGINT UNSIGNED NOT NULL DEFAULT 0, updated_at TIMESTAMP NULL)`.
   No data moves here (seeds: T-M9-27; bridge rows: T-M9-28).
2. sqlc queries: `BumpUserRevisions` (`INSERT … VALUES (?, 1, 1, NOW()) ON DUPLICATE KEY UPDATE engine_rev =
   engine_rev + 1, analysis_rev = analysis_rev + 1, updated_at = NOW()`), `BumpAnalysisRevision`, `GetUserRevisions`;
   skip CRUD (`ListCycleSkips`, `InsertCycleSkip`, `DeleteCycleSkip`); profile reads/writes for the two new columns;
   breastfeeding read/write. Add query files to `sqlc.yaml` only if a new package is needed.
3. `DataSource` (Go enums) gains `bleeding_log`, treated as actual data wherever `user_logged` is; grep every reader of
   `cycle_histories.source` and make sure an unknown value never fails parsing. Laravel `DataSource.php` is **not**
   changed (D-138: before any cutover).

## Tests
- Migration integration test: up, insert/read each new column and table, FK cascade on user delete, `down` restores
  the previous schema.
- Revision bump query: 2 bumps on a new user → `engine_rev = 2`; concurrent bumps from 10 goroutines → 10.
- Enum test: `bleeding_log` round trip; every switch on DataSource handles it (`exhaustive` lint or a table test).
- `make schema-diff` green.

## Out of scope
- Readers and writers of the new storage (T-M9-25…31).

## Acceptance
- Migration + twin + queries merged; verify passes.
