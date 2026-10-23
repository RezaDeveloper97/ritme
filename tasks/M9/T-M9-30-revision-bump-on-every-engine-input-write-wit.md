---
id: T-M9-30
title: Revision bump on every engine-input write, with a manifest test
milestone: M9
type: backend
status: done
depends_on: [T-M9-24, T-M9-27, T-M9-28]
parallel_group: M9-C
touches: [backend-go/internal/cycle/revision, backend-go/internal/cycle/periods, backend-go/internal/healthlog, backend-go/internal/fertility/service.go, backend-go/internal/profile, backend-go/internal/postpartum, backend-go/internal/pregnancy, backend-go/internal/contraception, backend-go/internal/ivf, backend-go/internal/menopause, backend-go/internal/babylog, backend-go/internal/admin/users]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/cycle/revision/... && make test-int PKG="./internal/cycle/revision/... ./internal/cycle/periods/... ./internal/healthlog/... ./internal/fertility/... ./internal/profile/... ./internal/postpartum/... ./internal/pregnancy/... ./internal/contraception/..."
---

# T-M9-30 — Revision bump on every engine-input write, with a manifest test
**Model:** opus.

## Why
A version-keyed cache is only correct if *every* write to an engine input bumps the version in the same transaction.
Today `calculation_version` is bumped by period writes, by health logs only when an LMP exists, by profile save — not
by cycle preferences, mode changes, pregnancy/postpartum, contraception, LH logs or admin edits (gap 07 §5, B7;
gap 02 §7).

## Scope
1. `internal/cycle/revision`: `Bump(ctx, q, uid)` and `BumpAnalysis(ctx, q, uid)` over the T-M9-24 queries, taking the
   caller's transaction (`DBTX`).
2. Call `Bump` inside the transaction of every write to an engine-input table (README §3): `cycle_histories`,
   `cycle_skips`, `daily_health_logs`, `health_log_entries` (engine categories), `fertility_logs`, `user_profiles`
   (engine columns), `cycle_preferences`, `user_life_profiles`, `contraception_methods`, `pregnancy_profiles`,
   `postpartum_profiles`. Writers that do not use a transaction today get one (or a single-statement bump right after,
   only where a transaction is impossible — documented). Data migrations bump in SQL.
3. Manifest: `revision/manifest.go` lists every sqlc write query on those tables with either `Bumps: "<integration test
   name>"` or `Exempt: "<reason>"` (e.g. a baby-log entry category the engine never reads, `updated_at` touches). A unit
   test parses `db/queries/**/*.sql` (`-- name:` + statement) and fails when a write query on an engine table is missing
   from the manifest or names a test that does not exist.
4. Keep `calculation_version` / `MarkRecalculated` as the client-visible "recalculated" signal (unchanged).

## Tests
- One integration test per manifest entry group asserting `engine_rev` increases by exactly 1 per write request
  (period start/end/store/update/destroy, health-log save/delete (v1 + v2), fertility day save, profile save, cycle
  settings, life-mode switch, pregnancy activate/deactivate/onboarding/v2 setup, postpartum activate + breastfeeding,
  contraception set/stop, IVF/menopause mode writes, admin user edit).
- A failing write (validation error) does not bump; a rolled-back transaction does not bump.
- Manifest test fails when a fake query file with an `UPDATE cycle_histories` is added (test fixture dir).

## Out of scope
- Using the revision (T-M9-31), analysis-only bumps for vitals/labs (T-M10-28).

## Acceptance
- Manifest complete and green; every listed test exists and passes.
