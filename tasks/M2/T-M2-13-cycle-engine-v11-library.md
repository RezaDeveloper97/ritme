---
id: T-M2-13
title: Cycle engine v1.1 library (metrics, resolver, view builder, daily card)
milestone: M2
type: backend
status: todo
depends_on: [T-M2-04, T-M2-07]
parallel_group: M2-E
touches: [backend-go/internal/cycle/metrics, backend-go/internal/cycle/resolver, backend-go/internal/cycle/view, backend-go/internal/cycle/model]
skills: []
verify: cd backend-go && go test -count=1 ./internal/cycle/metrics/... ./internal/cycle/resolver/... ./internal/cycle/view/...
---

# T-M2-13 — Cycle engine v1.1 library

## Why
`cycle_view` (the §35 fields the web app renders) comes from `CycleStatusResolver` + `CycleDayViewBuilder`. It is pure
date arithmetic with many branches and has 62 PHP unit tests that port 1:1 — the best parity asset we have.
Read domain-inventory §4.1 fully, plus memory note "Ritme Cycle Engine" and `task.md` v1.1 if present.

## Scope
1. `cycle/model`: input structs (history row: start, end?, is_confirmed, is_estimated, source, bleeding_length,
   data_quality_flags; profile: last_period_start, cycle_duration, period_duration, birthday, goal) decoupled from sqlc.
2. `cycle/metrics` = `CycleMetricsCalculator` (confirmed only, 21–45 valid, 2–10 durations, median of last ≤3 with
   PHP half-away rounding, effective values + `EffectiveSource`, variabilityRange, population std-dev of ≤6,
   regularity).
3. `cycle/resolver` = `CyclePhaseMapper` + `CycleStatusResolver` + `CycleStatus::toApiArray` (anchor order, future
   roll-forward `predicted_reference`, caps soft/warning/hard, resolve order, warning dedup keeping order,
   data_quality, confidence, reasons, `round(diffInDays)`).
4. `cycle/view` = `CyclePredictionService`, `OpenPeriodEvaluator`, `loggedPeriodFor`, `CycleConfidenceCalculator`,
   `DailyCardBuilder` (fa/en hardcoded copy, Persian digits, non-fa → English), `CycleDayViewBuilder` (merge order
   and legacy keys; the legacy-engine parts it needs are injected via an interface that T-M2-14 implements).
5. **Port the tests first**: translate all 62 unit tests (`backend/tests/Unit/**`) into Go table tests with the same
   names and fixtures, watch them fail, then implement.
6. Everything takes a `civildate.Date` "today" and a locale — no global clock, no DB access.

## Out of scope
Legacy `HealthDataEngine` (T-M2-14). HTTP endpoints and cache (T-M2-15).

## Acceptance
- All 62 ported tests green; test names map 1:1 to PHP test methods (a table in the package doc lists the mapping).
- Additional golden test: for every contract persona × 60 consecutive days, `cycle_view` equals the value in
  Laravel's `/cycle/date/{d}` sweep goldens (recorded by T-M2-05, `contract/golden/cycle-sweep/`). If the sweep is
  missing, stop and add a follow-up to T-M2-05 instead of editing `backend-go/contract/` here.
- No imports of `internal/platform/db` or Fiber from these packages.
