---
id: T-M9-52
title: Engine: stale anchor still paints a period day; onboarding-estimate row resolves to unknown
milestone: M9
type: backend
status: done
depends_on: [T-M9-42d]
parallel_group: M9-C
touches: [backend-go/internal/cycle/engine,backend-go/internal/cycle/resolver,backend-go/internal/cycle/inputs,backend-go/internal/cycle/model,backend-go/contract/golden-v6]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/cycle/... && make test-int PKG=./internal/cycle/... && make contract ROUTES=cycle
---

# T-M9-52 — Engine: stale anchor still paints a period day; onboarding-estimate row resolves to unknown

## Why
Found in T-M9-42d QA (`docs/engine-v6/qa/T-M9-42d/today_ask_last_period_*.png`, persona 1002): the declared anchor is
57 days old, Today shows the `ask_last_period` banner («تاریخ آخرین پریودت را بگو… فعلاً به‌روز نیستند») **and** a
menstrual ring «پریود · روز ۲» with the daily card «پریود · روز ۲». The two contradict each other: when the engine asks
for the last period, it must not project a period day from that stale anchor.
Also: a `cycle_histories` row with `source=onboarding_estimate`, `is_estimated=1` (persona 1003 as shipped in the
fixtures) gives `main_phase=unknown` and no prediction card; deleting the row and keeping
`onboarding_last_period_start` gives a proper prediction. An onboarding-only user should get the same result either way.

## Scope
- Find where the stale-anchor case is decided (`todayui.go` emits ask_last_period) and make `main_phase`/ring/daily
  card consistent with it (unknown / neutral) — check SPEC v6 + DECISIONS for the intended rule first and cite it.
- Make an `onboarding_estimate` row behave like the declared anchor (or document why not, with a SPEC reference).
- Regenerate the affected v6 goldens and explain each diff.

## Out of scope
- Frontend changes.

## Acceptance
- Int/unit tests for both cases; ask_last_period never coexists with a period main_phase; onboarding-only user with
  the estimate row gets a prediction; v1 contract unchanged; verify green.

## Resolution (main session, 2026-10-28)
- Case 1 fixed in the engine (§2.20 priority 1, §2.21, §2.41, §2.43): on a rolled-forward start (`stale_onboarding_anchor`,
  `futureK == 0`) days ≤ EPL resolve `unknown/unknown` instead of `menstrual`.
- Case 2 **not** changed in the engine. The agent's fallback (read the legacy `onboarding_estimate` row when the column is
  NULL) contradicted I-01 (D-123: the column is the only declared anchor). User chose (2026-10-28): keep I-01, fix the
  fixtures instead → T-M9-52b. Production is unaffected (goose 00103 / Laravel promote migration moved those rows).
