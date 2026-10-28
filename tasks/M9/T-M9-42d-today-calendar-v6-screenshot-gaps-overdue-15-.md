---
id: T-M9-42d
title: Today/calendar v6 screenshot gaps: overdue 15+, ask_last_period, unknown ring, pill + onboarding-only calendars
milestone: M9
type: frontend
status: done
depends_on: [T-M9-42c,T-M9-40b]
parallel_group: M9-C
touches: [frontend/src/screens/home,frontend/src/screens/calendar/ui/CalendarPage.tsx,docs/engine-v6]
skills: [local-dev]
verify: cd frontend && npm run typecheck && npm run test
---

# T-M9-42d — Today/calendar v6 screenshot gaps: overdue 15+, ask_last_period, unknown ring, pill + onboarding-only calendars

## Why
T-M9-42c shipped the code, but its screenshot step was partial: these were not captured (persona 11 hit the
schema bug fixed in T-M9-40b; pill/onboarding-only fixtures were not found).

## Scope
- Screenshots light + dark (local Go `CYCLE_ENGINE=v6`, `local-dev` skill, headless CDP): Today with the overdue 15+
  and ask_last_period banners, the `unknown` phase ring, the overdue banner button landing on the pregnancy-test chip,
  calendar for pill and onboarding-only personas.
- Also the T-M9-43 visual QA: prediction card light + dark in three states (calendar-estimated ovulation, BBT/LH-confirmed
  ovulation, fertile window null / suppressed) with the regularity badge; save under `docs/engine-v6/qa/`.
- Fix any v6 display bug they reveal inside `touches`; otherwise file a follow-up task.

## Out of scope
- Backend changes.

## Acceptance
- All screenshots above exist and are reviewed in the main session; verify green.
