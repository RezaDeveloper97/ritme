---
id: B-N3-03
title: Log sheet v2 — quick tiles, accordion, detail panels, body map
milestone: N3
type: frontend
status: done
depends_on: [B-N3-01,B-N3-02]
parallel_group: N3-C
touches: [frontend/src/screens/log,frontend/src/features/log-day,frontend/src/entities/health-log,frontend/src/widgets/body-map]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N3-03 — Log sheet v2 — quick tiles, accordion, detail panels, body map

## Why
The core daily action.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Sheet_Cycle.dc.html` (+ `nbd_Log_Sheet_Cycle`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Bleeding.dc.html` (+ `nbd_Log_Bleeding`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Pain.dc.html` (+ `nbd_Log_Pain`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Measure.dc.html` (+ `nbd_Log_Measure`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Cycle_Log.dc.html` (+ `nbd_Cycle_Log`)

## Scope
- Date strip, «ثبت دستی / ثبت با صدا» tabs, quick tiles that open their accordion section, all-categories accordion, search, summary footer + save; detail panels: bleeding, pain (+ SVG body map), weight & BBT & tests.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
