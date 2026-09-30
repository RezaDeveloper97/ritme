---
id: B-N1-06
title: Cycle home redesign — normal / near period / during period
milestone: N1
type: fullstack
status: todo
depends_on: [B-N1-04]
parallel_group: N1-F
touches: [frontend/src/screens/home,frontend/src/widgets/banner-slideshow,frontend/src/widgets/today-challenge,frontend/src/widgets/day-tasks,frontend/src/widgets/smart-tip,frontend/src/widgets/week-summary,frontend/src/widgets/home-cycle,frontend/src/entities/cycle,backend-go/internal/home,backend-go/internal/cycle,backend-go/api]
skills: [new-endpoint,new-fsd-slice,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-06 — Cycle home redesign — normal / near period / during period

## Why
Home is the most-seen screen; the design adds states, predictions and streaks.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_Home.dc.html` (+ `nbd_Cycle_Home`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_Home_Near.dc.html` (+ `nbd_Cycle_Home_Near`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_Home_During.dc.html` (+ `nbd_Cycle_Home_During`)

## Scope
- Greeting + DateStrip; hero countdown (next period / period day N / period in 1 day) with the three state variants and their prompts («پریودم شروع شد / هنوز نه», «پریودم تموم شد / هنوز ادامه دارد»).
- Phase card with pregnancy-chance pill + «بیشتر درباره این فاز» → phase sheet.
- Today-log summary card with logging streak («۶ روز پشت‌سرهم») and category chips.
- Predictions card: next period range, PMS window, fertile window, ovulation, median cycle length ± variability, regularity.
- Keep existing banner slideshow, recommendations and challenges widgets, restyled.
- Backend: expose any missing fields (streak, PMS window, period range, regularity label) on `/home` or `/cycle/today` with OpenAPI + tests; engine math stays in the cycle engine.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- Three states reachable with the test clock
- `verify` green
