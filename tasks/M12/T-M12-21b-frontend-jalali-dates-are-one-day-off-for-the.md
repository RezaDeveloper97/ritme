---
id: T-M12-21b
title: Frontend Jalali dates are one day off for the whole year 1404 (jalaliday leap rule)
milestone: M12
type: frontend
status: done
depends_on: []
parallel_group: M12-TEST
touches: [frontend/src/shared/lib/date,frontend/src/screens/analysis-monthly/model,frontend/package.json,frontend/package-lock.json]
skills: []
verify: cd frontend && npm run typecheck && npx vitest run src/shared/lib/date src/screens/analysis-monthly
---

# T-M12-21b — Frontend Jalali dates are one day off for the whole year 1404 (jalaliday leap rule)

**Model:** opus. Found in T-M12-21 (2026-11-18), confirmed with node in `frontend/node_modules` (jalaliday 2.3.0).

## Why
jalaliday treats 1404 as the leap year; the official calendar (and the backend `jalaali` converter) has 1403 leap.
Result in the frontend: 2025-03-20 renders 1404-01-01 (official 1403-12-30), 2025-03-21 renders 1404-01-02 (official
1404-01-01) — every date from Nowruz 1404 to 2026-03-20 is shown one Jalali day off, and 2026-03-20 renders
1404-12-30 (official 1404-12-29). Calendar history, logs, reports and the monthly analysis (frontend vs
`analysis/monthly.go`) disagree for that whole year. 1405 renders correctly.

## Scope
1. Confirm which code paths format/parse Jalali (`shared/lib/date/calendar.ts`, `screens/analysis-monthly/model/monthly.ts`,
   any other `calendar('jalali')` / `{ jalali: true }` use — grep all).
2. Fix: upgrade jalaliday if a release follows the official (33-year arithmetic / Iranian solar) rule, otherwise swap
   the conversion to `jalaali-js` (same algorithm as the backend) behind the existing `shared/lib/date` API.
3. Table test against the backend's facts: 2025-03-20 = 1403-12-30, 2025-03-21 = 1404-01-01, 2026-03-20 = 1404-12-29,
   2026-03-21 = 1405-01-01, plus 1399/1403/1408 leap checks and round-trips; a parity test that frontend and backend
   month boundaries agree for 1400–1410.
4. Mind the jalaliday setter pitfall (memory: `.year()/.month()/.date()` write Gregorian fields).

## Out of scope
- Android.

## Acceptance
- Table + parity tests green and red on the old library; calendar and monthly analysis screens show 1404 dates equal to
  the backend (screenshot a 1404 month).
