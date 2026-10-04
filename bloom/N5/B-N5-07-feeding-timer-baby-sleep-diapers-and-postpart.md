---
id: B-N5-07
title: Feeding timer, baby sleep/diapers and postpartum analysis hub
milestone: N5
type: frontend
status: done
depends_on: [B-N5-03,B-N3-08]
parallel_group: N5-G
touches: [frontend/src/screens/log-feed,frontend/src/screens/analysis-postpartum,frontend/src/features/baby-log]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N5-07 — Feeding timer, baby sleep/diapers and postpartum analysis hub

## Why
Postpartum logging + analysis.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Feed.dc.html` (+ `nbd_Log_Feed`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_An_Hub_Post.dc.html` (+ `nbd_An_Hub_Post`)

## Scope
- Feed timer (L/R, bottle, pump), last feed, today totals; An_Hub_Post (EPDS trend, bleeding trend, feeds L/R split, mother/baby sleep, growth percentiles, recovery weight).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
