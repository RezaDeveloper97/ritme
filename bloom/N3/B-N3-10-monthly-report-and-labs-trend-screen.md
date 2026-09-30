---
id: B-N3-10
title: Monthly report and labs trend screen
milestone: N3
type: frontend
status: todo
depends_on: [B-N3-08]
parallel_group: N3-J
touches: [frontend/src/screens/analysis-monthly,frontend/src/screens/analysis-labs]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N3-10 — Monthly report and labs trend screen

## Why
Monthly digest.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Monthly.dc.html` (+ `nbd_An_Monthly`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Labs.dc.html` (+ `nbd_An_Labs`)

## Scope
- Monthly report table + top symptoms + next-month suggestion + «ساخت PDF برای پزشک» (links B-N6-04). Labs trend shows an empty state until B-N6-06 data exists.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
