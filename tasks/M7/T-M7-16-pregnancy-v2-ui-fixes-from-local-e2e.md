---
id: T-M7-16
title: Pregnancy v2 UI fixes from local e2e
milestone: M7
type: frontend
status: done
depends_on: [T-M7-10,T-M7-11,T-M7-12,T-M7-14,T-M7-09]
parallel_group: M7-E
touches: [frontend/src/widgets/pregnancy-week-carousel,frontend/src/screens/pregnancy,frontend/src/screens/pregnancy-onboarding,frontend/src/screens/pregnancy-week,frontend/src/screens/pregnancy-log,frontend/src/screens/onboarding-setting-up,frontend/src/screens/profile,frontend/src/entities/pregnancy,backend-go/internal/messages/pregnancyalerts,docs/pregnancy-v2]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../backend-go && go vet ./... && go test ./internal/messages/...
---

# T-M7-16 — Pregnancy v2 UI fixes from local e2e

## Why
The local e2e for T-M7-15 (docs/pregnancy-v2/README.md § Local e2e (T-M7-15)) found bugs 1–8 (+ minor 9a–9j).
File:line details and repro steps are there.

## Scope
1. Week carousel on Today collapses to 32px — add `shrink-0` (PregnancyWeekCarousel / PregnancyPage column).
2. Trimester bars fill wrongly: API `trimesters[].percent` is the start position on the 40-week bar (service.go +
   OpenAPI). Keep the API contract; fix the frontend to compute fill from the current week/the documented fields,
   and fix the type comment in the entity. Unit test.
3. Setup v2 step bar renders as a segmented control (`.seg`) — use a proper stepper look per design.
4. Week stats in Latin digits (PregnancyWeekPage) → locale number formatting.
5. `week_entered` alert text uses `strconv.Itoa` → Persian digits for fa (backend-go pregnancyalerts); Go test.
6. Log save status stays «queued» after the outbox syncs, and the button says «ذخیره شد» while only queued —
   show queued vs saved correctly (use-save-day `onSent`, DayLogPage).
7. SettingUpPage swallows every error and can save a pregnant signup as a cycle profile: don't save with a
   missing `intention`, surface failures (retry UI/toast), and make the pregnant path always call activate +
   onboarding. Unit test for the decision logic.
8. Profile page Latin digits (ProfilePage) → locale formatting.
9. Minor, only if trivial: 9a appointment category for NT prefill = scan; 9f saved weight shown with Latin digits.

## Out of scope
- 9b–9e, 9g–9j (design/product or clinical-content questions; listed for the human reviewer).

## Acceptance
- Items 1–8 fixed with unit tests for logic; verify green; light/dark screenshots of Today, Setup, Week, Log,
  Alerts and Profile re-taken locally into `docs/pregnancy-v2/screenshots/` (same names, same compression).
