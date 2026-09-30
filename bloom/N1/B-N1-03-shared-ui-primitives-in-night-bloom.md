---
id: B-N1-03
title: Shared UI primitives in Night & Bloom
milestone: N1
type: frontend
status: todo
depends_on: [B-N1-02]
parallel_group: N1-C
touches: [frontend/src/shared/ui]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-03 — Shared UI primitives in Night & Bloom

## Why
Every screen in the canvas is assembled from ~15 repeated parts; build them once.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Sheet_Cycle.dc.html` (+ `nbd_Log_Sheet_Cycle`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_Settings.dc.html` (+ `nbd_Cycle_Settings`)
- `docs/design/night-bloom/a-start-plus/nbl_Onb_Cycle.dc.html` (+ `nbd_Onb_Cycle`)

## Scope
- Per N1-01 inventory: ScreenHeader (round 44px back/action, title + subtitle), DateStrip (week, Jalali), Card, SectionTitle, PillChip (toggle/multi), SegmentedTabs, NumberStepper (big Lalezar value, −/+), StatusPill, ListRow, Accordion, BottomSheet, PrimaryButton/SecondaryButton (54px pill), IconCircle, InfoNote, Skeleton, EmptyState, ProgressRing, simple SVG LineChart/BarChart (RTL-safe, `direction:ltr` plot).
- Stories/usage page in the existing ui-kit route (or a dev-only route) showing each in light + dark.
- Replace old primitives in place when the API matches; otherwise add alongside and mark old ones deprecated.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Each primitive has a unit test for its a11y contract (role/label)
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- `verify` green
