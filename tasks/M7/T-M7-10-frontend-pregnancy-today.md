---
id: T-M7-10
title: Frontend — pregnancy «امروز» (Today) v2 screen
milestone: M7
type: frontend
status: done
depends_on: [T-M7-08, T-M7-02]
parallel_group: M7-C
touches: [frontend/src/screens/pregnancy, frontend/src/widgets/pregnancy-week-carousel, frontend/src/widgets/pregnancy-care-checklist]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M7-10 — Frontend — pregnancy Today v2

## Scope
Rebuild `screens/pregnancy` from `Main.dc.html`: date strip; swipeable week carousel (prev/current/next,
«بازگشت به امروز»); due-date card with range; 40-week progress with trimester segments; 4 quick actions (log, weekly
checkup → existing weekly form, weeks → `/pregnancy/weeks/[current]`, alerts with badge); next-visit card; smart tip
card (API tip); «مراقبت‌های این هفته» checklist (week state PUT, done count); disclaimer. Old v1 sections removed
(AlertsCard/WeekContent move to the new screens). Skeleton/empty/error states.

## Acceptance
- Light/dark × fa/en screenshots; build green.
