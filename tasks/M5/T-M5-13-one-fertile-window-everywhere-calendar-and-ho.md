---
id: T-M5-13
title: One fertile window everywhere — calendar and home timeline use the v1.1 anchors
milestone: M5
type: frontend
status: in_progress
depends_on: [T-M5-12]
parallel_group: M5-D
touches: [frontend/src/screens/calendar,frontend/src/widgets/cycle-timeline,frontend/src/widgets,frontend/src/entities/cycle,frontend/src/screens/home,docs/fertility-ttc]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M5-13 — One fertile window everywhere — calendar and home timeline use the v1.1 anchors

## Why
T-M5-12 aligned the home "fertile window" text with `/fertility/bbt` (task.md §19: `max(O−5, period_end+1) … O`), but
the calendar day markers (`is_fertile_window`, legacy calculation) and the home `CycleTimelineBar` still draw the
biological O−5…O+1 window, so screens disagree by a day.

## Scope
- Make the calendar (month grid markers, legend, day detail) and the home cycle timeline bar use the same window as
  `entities/cycle/model/schedule.ts` (from `cycle_view.anchors`), for current and predicted cycles. Where the API
  only gives the legacy per-day flag for past/future months, derive the window from the anchors/predictions with the
  same §19 rule in one shared helper (entities/cycle), not per screen. Frontend only.
- Unit tests for the helper incl. the cycle-day-16 case and a short cycle where period end overlaps O−5.

## Acceptance
- Calendar, home timeline, home text, insights and BBT all show the same window days for the same user (checked
  locally with one TTC user; screenshots in docs/fertility-ttc/screenshots/window-*.png); verify green.
