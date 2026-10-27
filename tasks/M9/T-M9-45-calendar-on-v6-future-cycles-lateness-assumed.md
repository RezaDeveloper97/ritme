---
id: T-M9-45
title: Calendar on v6: future cycles, lateness, assumed ranges, suppression
milestone: M9
type: frontend
status: done
depends_on: [T-M9-40, T-M9-33]
parallel_group: M9-C
touches: [frontend/src/screens/calendar, frontend/src/widgets/cycle-calendar, frontend/src/entities/cycle/model, frontend/src/entities/cycle/api/schema.ts, frontend/src/entities/cycle/index.ts, frontend/messages/fa/calendar.json, frontend/messages/en/calendar.json, backend-go/resources/translations/fa/calendar.json, backend-go/resources/translations/en/calendar.json]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- calendar
---

# T-M9-45 — Calendar on v6: future cycles, lateness, assumed ranges, suppression
**Model:** sonnet.

## Why
Future days are painted from two sources (legacy month periods + the browser's own unlimited roll-forward), late days
are painted as a new period, the open period's end is the profile length instead of the engine's anchor, and the
calendar shows fertile days under the pill (gap 02c C, D.3, D.7). DECISIONS.md E-04, E-05, E-22, E-28, I-01.

## Scope
1. With v6 month data (`period_kind`, per-day `main_phase`, month `future_cycles`): paint from the server — `logged`
   (solid period), `assumed` (open period's assumed days and the declared onboarding range, styled as predicted),
   `predicted` (future cycles, k ≤ 6), period_expected days with the late style; nothing beyond `future_cycles[6]`.
2. Replace `cycleScheduleFor` and the forward branch of `cycleDayMarkerAt` with a `future_cycles` day map; no client
   roll-forward while `days_late > 0`.
3. Open-period end from `anchors.current_period_end` + `current_period_end_is_confirmed`.
4. Suppression: `withoutFertility` + `NO_FERTILITY_LEGEND` when `isFertilitySuppressed`.
5. Laravel month payload → today's rendering (fallback kept).
6. Delete the unused helpers listed in gap 02c §C (`phase.ts`, `highlights.ts`, `markerIntensityByDate`,
   `fertilityStyle.ts`, `DailyStatusCard`, `cycleProgressPercent`, `fertileWindowDays`, `calcToPhase`) **only after
   grep proves they have no importer**; delete their tests with them.

## Tests
- Day map: SPEC §2.35 example month; k = 6 boundary; 10 days late → late days, no rolled period, predicted period from
   tomorrow; 15 days late → no predicted days; confirmed vs assumed end; onboarding-only → assumed range; pill → no
   fertile marks; Laravel payload → existing tests green.
- Rewrite `schedule.test.ts:185-258, 303-460` for the new behaviour (keep Laravel cases).
- Visual QA light/dark: late, pill, onboarding-only, regular.

## Out of scope
- Backend month (T-M9-33).

## Acceptance
- Verify passes; screenshots reviewed.

## Touches added during the work
`entities/cycle/api/schema.ts` and `entities/cycle/index.ts`: the /cycle/month envelope parser dropped every v6 month field and null-`cycle_day` days, so the calendar could not see them.
