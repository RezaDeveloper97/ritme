---
id: T-M9-42c
title: Home timeline on v6 fields, delete legacy schedule helpers, Today/calendar screenshots
milestone: M9
type: frontend
status: todo
depends_on: [T-M9-42b,T-M9-34]
parallel_group: M9-C
touches: [frontend/src/screens/home,frontend/src/screens/calendar/ui/CalendarPage.tsx,frontend/src/screens/onboarding-flow/ui/ReadyStep.tsx,frontend/src/entities/cycle/model,frontend/src/entities/cycle/index.ts]
skills: [local-dev]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M9-42c — Home timeline on v6 fields, delete legacy schedule helpers, Today/calendar screenshots
**Model:** sonnet; main session reviews screenshots.

## Why
Left over from T-M9-42b: `cycleScheduleFor`, `daysUntilNextPeriod`, `hasFertileWindow` and the legacy branch of
`cycleDayMarkerAt` still compute cycle dates in the browser. HomePage uses them for the week-strip dots, the
ovulation / fertile-window / PMS slots and the countdown fallback; CalendarPage (`cycleDayMarkerAt`) and onboarding
ReadyStep (`hasFertileWindow`) import them too. The v6 `/home` (T-M9-34) already sends `week_calendar` (per-day
main_phase, fertility_level, period_kind, is_predicted, days_late) and `cycle_prediction` (fertile O−5..O, ovulation,
pms_window, null under suppression).

## Scope
1. Home on a v6 payload: week strip and the ovulation / fertile / PMS / countdown slots read the server fields; the
   Laravel payload keeps the legacy path until T-M9-51.
2. CalendarPage and ReadyStep: use server data on v6; keep the legacy branch only for the Laravel payload.
3. Delete whatever has no importer left (grep first), with its tests.
4. Screenshots light + dark (local Go with `CYCLE_ENGINE=v6`, `local-dev` skill, headless CDP): Today with each
   banner (end_missing, overdue 8–14, overdue 15+, ask_last_period), `unknown` phase ring, overdue banner button
   landing on the pregnancy-test chip, calendar for late, pill, onboarding-only and regular personas.

## Out of scope
- Backend changes.

## Acceptance
- No browser cycle math on the v6 path (grep); verify green; screenshots reviewed in the main session.
