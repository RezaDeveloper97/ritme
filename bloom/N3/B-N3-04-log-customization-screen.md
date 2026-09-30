---
id: B-N3-04
title: Log customization screen
milestone: N3
type: frontend
status: todo
depends_on: [B-N3-03]
parallel_group: N3-D
touches: [frontend/src/screens/log-customize,frontend/src/features/customize-log]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N3-04 — Log customization screen

## Why
Gear icon on the log sheet.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Customize.dc.html` (+ `nbd_Log_Customize`)

## Scope
- Drag to reorder (pointer + keyboard accessible), pin ≤8, hide, add custom item.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
