---
id: B-N1-14
title: Restyle pregnancy v2 screens to Night & Bloom
milestone: N1
type: frontend
status: todo
depends_on: [B-N1-03]
parallel_group: N1-N
touches: [frontend/src/screens/pregnancy,frontend/src/screens/pregnancy-setup,frontend/src/screens/pregnancy-week,frontend/src/screens/pregnancy-log,frontend/src/screens/pregnancy-calendar,frontend/src/screens/pregnancy-alerts,frontend/src/widgets/pregnancy-week-carousel,frontend/src/widgets/pregnancy-care-checklist]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-14 — Restyle pregnancy v2 screens to Night & Bloom

## Why
M7 shipped pregnancy v2; the canvas re-skins it (PregFull set).

## Design
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_PregFull_Setup.dc.html` (+ `nbd_PregFull_Setup`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_PregFull_Main.dc.html` (+ `nbd_PregFull_Main`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_PregFull_Week.dc.html` (+ `nbd_PregFull_Week`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_PregFull_Log.dc.html` (+ `nbd_PregFull_Log`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_PregFull_Calendar.dc.html` (+ `nbd_PregFull_Calendar`)
- `docs/design/night-bloom/b2-ttc-pregnancy-postpartum-child/nbl_PregFull_Alerts.dc.html` (+ `nbd_PregFull_Alerts`)
- `docs/design/night-bloom/c-health-record/nbl_v13_Preg_Home.dc.html` (+ `nbd_v13_Preg_Home`)

## Scope
- Setup (4 steps, dating basis, history), Today, Week-by-week, Log, Calendar & visits, Alerts (4 levels). Also the reminders block from `v13_Preg_Home`.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
