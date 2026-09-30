---
id: B-N3-08
title: Analysis hub and tab routing
milestone: N3
type: frontend
status: todo
depends_on: [B-N3-07,B-N1-04]
parallel_group: N3-H
touches: [frontend/src/screens/analysis,frontend/src/entities/analysis,frontend/src/app/[locale]/analysis]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N3-08 — Analysis hub and tab routing

## Why
Entry to all reports.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Hub.dc.html` (+ `nbd_An_Hub`)

## Scope
- Hub with range + category filters, top finding, cards per category, Plus locks; mode-specific hub chosen by mode.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
