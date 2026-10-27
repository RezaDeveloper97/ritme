---
id: T-M9-42b
title: Today banner follow-ups: pregnancy-test deep link, banner CSS, remove legacy schedule helpers
milestone: M9
type: frontend
status: todo
depends_on: [T-M9-42,T-M9-45]
parallel_group: M9-C
touches: [frontend/src/features/log-day/ui/LogDay.tsx,frontend/src/screens/log/ui/LogSheet.tsx,frontend/src/screens/home,frontend/src/app/globals.css,frontend/src/widgets/cycle-calendar/ui/lateStyle.ts,frontend/src/widgets/cycle-calendar/ui/CalendarLegend.tsx,frontend/src/entities/cycle/model]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M9-42b — Today banner follow-ups: pregnancy-test deep link, banner CSS, remove legacy schedule helpers
**Model:** sonnet; main session reviews screenshots.

## Why
T-M9-42 and T-M9-45 left work outside their `touches`: the overdue banner's «ثبت تست بارداری» opens the log sheet on
its default view, the banner and the calendar late style have no real CSS (inline / reused `.ch-alert`), and the legacy
client schedule helpers survive only because the home screen still imports them.

## Scope
1. Log sheet deep link: `openSheet('log', date, { panel: 'measurement', focus: 'pregnancy_test' })` (or equivalent)
   through `screens/log/ui/LogSheet.tsx` and `features/log-day/ui/LogDay.tsx`; the overdue banner uses it.
2. CSS in `frontend/src/app/globals.css`: banner action buttons (`.ch-alert` actions), `.ch-pred-disclaimer`, and the
   calendar late-day style (move the inline style out of `lateStyle.ts` / `CalendarLegend.tsx`). Tokens only, light+dark.
3. Home stops importing `cycleScheduleFor`, `daysUntilNextPeriod`, `hasFertileWindow` and the legacy branch of
   `cycleDayMarkerAt` on the v6 path; delete what no longer has an importer (grep first), with its tests
   (`schedule.test.ts` legacy cases).
4. Visual QA screenshots light+dark (headless CDP, local Go with `CYCLE_ENGINE=v6`): Today with each banner
   (end_missing, overdue 8–14, overdue 15+, ask_last_period), the `unknown` phase ring, and the calendar for late,
   pill, onboarding-only and regular personas.

## Out of scope
- Backend changes; new copy beyond button labels.

## Acceptance
- The overdue banner button lands on the pregnancy-test field.
- No inline styles left for the banner/late day; style and dark gates green.
- No importer left for the deleted helpers; verify green; screenshots reviewed in the main session.
