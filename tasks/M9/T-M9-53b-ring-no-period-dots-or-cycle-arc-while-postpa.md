---
id: T-M9-53b
title: Ring: no period dots or cycle arc while postpartum before cycles return
milestone: M9
type: frontend
status: done
depends_on: [T-M9-53]
parallel_group: M9-C
touches: [docs/engine-v6/qa/T-M9-53b,frontend/src/entities/cycle/model,frontend/src/screens/home]
skills: []
verify: cd frontend && npm run typecheck && npm run test -- entities/cycle screens/home && npm run fsd:lint
---

# T-M9-53b — Ring: no period dots or cycle arc while postpartum before cycles return

## Why
T-M9-53 stopped the engine from emitting `cycle_day` under postpartum-not-returned (E-14), but the engine still sends
`anchors.current_period_start/end` as history anchors (same as pregnancy). `deriveCycleSchedule`
(`frontend/src/entities/cycle/model/schedule.ts`) builds the ring from those anchors and the effective lengths, so the
ring still draws period dots and a cycle arc eight days after birth.

## Scope
- When `limits.suppression_reason === 'postpartum_not_returned'` (or the day has no `cycle_day` and the engine says no
  prediction), the ring draws no predicted period dots and no cycle arc; logged bleeding days still show.
- Unit test for the schedule with a postpartum fixture; one light + dark screenshot of the home ring for the
  postpartum persona (birth 2026-10-20, breastfeeding) in the scratchpad, looked at by the main session.

## Out of scope
- Backend (done in T-M9-53). Pregnancy mode (its own home).

## Acceptance
- No period dots / «روز N» on the ring in the postpartum-not-returned state; other modes unchanged; verify green.

## Outcome
Not reproducible after T-M9-53: on v6 the home ring takes its day from `cycle_view.cycle_day` (now null) and
`schedule` is null, so the postpartum persona already shows an empty ring with «پیش‌بینی پریود / فعلاً نیست»
(screenshots in `docs/engine-v6/qa/T-M9-53b/`). A guard in `screens/home` was tried and dropped because it could never
fire (v6 has no schedule; Laravel payloads have no limits). Kept: a postpartum regression test in `schedule.test.ts`.
