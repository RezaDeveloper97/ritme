---
id: B-N2-08
title: Trial banner, trial sheet and Plus locks across the app
milestone: N2
type: frontend
status: todo
depends_on: [B-N2-06,B-N2-07,B-N1-06]
parallel_group: N2-H
touches: [frontend/src/widgets/plus-trial-banner,frontend/src/widgets/plus-trial-sheet,frontend/src/shared/ui/plus-gate,frontend/src/screens/home]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N2-08 — Trial banner, trial sheet and Plus locks across the app

## Why
Conversion surfaces.

## Design
- `docs/design/night-bloom/a-start-plus/nbl_Prem_TrialHome.dc.html` (+ `nbd_Prem_TrialHome`)
- `docs/design/night-bloom/a-start-plus/nbl_Prem_TrialSheet.dc.html` (+ `nbd_Prem_TrialSheet`)

## Scope
- Home banner with live timer, sheet with usage list + offer + free-vs-plus note; `<PlusGate>` badge/lock component used by later Plus features.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
