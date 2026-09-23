---
id: T-M7-09
title: Frontend — re-enable pregnancy entry points, mode-aware nav, Setup v2 flow
milestone: M7
type: frontend
status: todo
depends_on: [T-M7-08, T-M7-02]
parallel_group: M7-B
touches: [frontend/src/entities/user/model/steps.ts, frontend/src/widgets/bottom-nav, frontend/src/screens/profile/ui/ProfilePage.tsx, frontend/src/screens/onboarding-intention, frontend/src/screens/onboarding-pregnancy-basis, frontend/src/screens/onboarding-setting-up, frontend/src/screens/pregnancy-onboarding, frontend/src/app/[locale]/pregnancy/setup]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M7-09 — Frontend — pregnancy entry points + Setup v2

## Why
The pregnancy option existed in signup and profile but is commented out («TEMPORARILY»). The user wants it back with
the v2 Setup artboard.

## Scope
1. Re-enable: intention + pregnancy-basis steps in `steps.ts`, pregnancy-aware `nav-items.ts` (امروز · تقویم ·
   gradient + FAB → log · بارداری · پروفایل), profile app-mode section (switch to pregnancy → setup; switch back →
   deactivate with confirm).
2. Setup v2 (`/pregnancy/setup`, replaces `pregnancy-onboarding` UI; signup's pregnancy-basis step reuses the same
   step components): welcome → 3-step progress (dating source segmented with LMP month calendar / ultrasound date +
   week + day / manual week + day and per-source hint; optional history with exclusive «هیچ‌کدام», blood group,
   Rh, «فعلاً رد می‌شم»; result from `dating-preview` with «مبنای محاسبه رو عوض می‌کنم») → submits v1
   `/pregnancy/onboarding` + activate → `/pregnancy`.
3. The history disclaimer must be truthful: use the corrected copy from `pregnancy_setup` messages (server storage)
   unless the user decides otherwise (docs/pregnancy-v2/README.md open point 1).
4. Existing cycle users unaffected; pregnancy users land on `/pregnancy`.

## Acceptance
- Signup (pregnant) and profile switch both reach the pregnancy home; light/dark × fa/en screenshots of all 4 setup
  states; build green.
