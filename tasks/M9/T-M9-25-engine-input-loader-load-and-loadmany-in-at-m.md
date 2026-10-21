---
id: T-M9-25
title: Engine input loader: Load and LoadMany in at most 4 queries
milestone: M9
type: backend
status: done
depends_on: [T-M9-24, T-M9-15, T-M9-07]
parallel_group: M9-C
touches: [backend-go/internal/cycle/inputs, backend-go/db/queries/cycle/inputs.sql, backend-go/internal/cycle/store, backend-go/internal/contraception/hormonal.go]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/cycle/... ./internal/contraception/... && make test-int PKG=./internal/cycle/inputs/...
---

# T-M9-25 — Engine input loader: Load and LoadMany in at most 4 queries
**Model:** opus (health data mapping).

## Why
The v1.1 snapshot reads 5 of the 15 v6 inputs; LH, BBT, mode, contraception, the pregnancy test, the delivery date,
skips, breastfeeding and the declared anchor never reach the engine, and every module has its own loader (gap 02 §2,
§5). One loader with a fixed query budget is also what makes the state cache cheap on a miss.

## Scope
1. `inputs.Load(ctx, q, uid) (engine.Input, error)` with at most 4 queries:
   1. one joined row: `user_profiles` (declared lengths, `onboarding_last_period_start`, birthday, goal) ⋈
      `cycle_preferences.lengths_auto` ⋈ `user_life_profiles` (life_mode, medications) ⋈ active `pregnancy_profiles` ⋈
      `postpartum_profiles` (birth date, breastfeeding, active) ⋈ active `contraception_methods.method` ⋈
      `teen_profiles.age_band`;
   2. all periods of the user, narrow projection (start, end, source, is_confirmed, is_estimated), ordered by start;
   3. all skips;
   4. markers: one `UNION ALL` over `fertility_logs.lh_test`, `health_log_entries` (`measurements` lh_test, bbt,
      pregnancy_test) and `daily_health_logs.basal_body_temperature`, from the start of the 7th newest period (or 400 days
      back, whichever is earlier) to today + 1.
2. Mapping rules: effective mode via `enums.ResolveLifeMode` (active pregnancy wins); hormonal per I-03
   (`contraception.IsHormonal(method)` + onboarding `contraceptive_pill`; onboarding `iud` → not hormonal);
   `TeenBandOnly` from an age band without a birthday; BBT → integer hundredths of °C (one reading per day; v2 wins over
   the legacy column when both exist; readings outside 34.00–43.00 dropped); LH positive only, one per day; latest
   pregnancy tests; manual lengths from `lengths_auto = 0`.
3. Period rows: `is_estimated = 1` rows are ignored; `user_profile_confirmed` seed rows are read as the declared
   anchor when the new column is empty (pre-T-M9-27 data); every other row is an actual period (`bleeding_log` and
   unconfirmed `user_logged` included — I-02).
4. `inputs.LoadMany(ctx, q, uids)` with the same 4 queries using `IN (…)` (batches of ≤ 500), for snapshots/admin.
5. JSON encoding of `engine.Input` (stable, versioned `v` field) for the L2 snapshot (T-M9-31).
6. Nothing else calls these tables for engine purposes after T-M9-36/37; this task does not migrate consumers.

## Tests
- Adapter unit tests per input (hormonal table for every method; BBT unit conversion and range; LH dedupe; mode
  precedence; teen band).
- Integration: a fixture user with every input → `Load` returns all of them; exactly ≤ 4 queries (T-M9-07 counter);
  `LoadMany` for 3 users equals 3 × `Load`; a user with no profile row → zero input, no error.
- JSON round trip of `engine.Input`; decoding an unknown version → error (forces a reload).

## Budgets
- `Load` cold ≤ 4 queries; p95 ≤ 5 ms on the local test DB for a user with 120 periods and 400 markers.

## Out of scope
- Caching (T-M9-31), consumers (T-M9-32…38).

## Acceptance
- Tests green; PROGRESS.md lists any input the fixtures could not cover.
