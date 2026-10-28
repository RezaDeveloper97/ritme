---
id: T-M9-52b
title: Contract fixtures in I-01 shape: declared LMP in onboarding_last_period_start, no estimate seed rows
milestone: M9
type: backend
status: todo
depends_on: [T-M9-52]
parallel_group: M9-C
touches: [backend/database/seeders/ContractFixtureSeeder.php,backend-go/contract/fixtures,backend-go/contract/golden,backend-go/contract/golden-v6,backend-go/internal/fertility/testdata,backend-go/internal/cycle/service/testdata,backend-go/internal/home/testdata]
skills: []
verify: cd backend-go && make contract ROUTES=cycle && make contract-v6 ROUTES=cycle && make test-int PKG="./internal/cycle/... ./internal/fertility/... ./internal/home/..."
---

# T-M9-52b — Contract fixtures in I-01 shape: declared LMP in onboarding_last_period_start, no estimate seed rows

## Why
`ContractFixtureSeeder.php` (~line 168) and `backend-go/contract/fixtures/dump.sql` still seed the pre-I-01 shape:
the onboarding-only personas (1002, 1003, …) keep the declared LMP in a `cycle_histories` row
(`source=onboarding_estimate`, `is_estimated=1`) with `user_profiles.onboarding_last_period_start` NULL. On a real
database goose 00103 / the Laravel promote migration already moved these, so v6 goldens for these personas show
`insufficient_anchor_data`/`unknown`, which no real user gets. User decision 2026-10-28 (T-M9-52): keep I-01, fix the
fixtures, no engine fallback.

## Scope
- Seed the declared LMP into `onboarding_last_period_start` and drop the estimate seed rows (mirror what the promote
  migration does), in both the Laravel seeder and `dump.sql`.
- Re-record the v1 goldens only if the Laravel responses change (they should not: v1 reads the same date) and explain
  any diff; regenerate the affected v6 goldens (`UPDATE_GOLDEN_V6=1`) for cycle, cycle-month, home and fertility and
  explain each diff (expected: onboarding_declared gets a declared anchor and a prediction card).

## Out of scope
- Engine changes.

## Acceptance
- No `onboarding_estimate` rows left in fixtures; v1 contract green with no new allow-list entries; v6 goldens
  regenerated and explained; verify green.
