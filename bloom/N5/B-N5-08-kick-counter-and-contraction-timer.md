---
id: B-N5-08
title: Kick counter and contraction timer
milestone: N5
type: frontend
status: todo
depends_on: [B-N5-03]
parallel_group: N5-H
touches: [frontend/src/screens/log-kick,frontend/src/screens/log-contraction,frontend/src/features/pregnancy-tools]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N5-08 — Kick counter and contraction timer

## Why
Live pregnancy tools.

## Design
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Kick.dc.html` (+ `nbd_Log_Kick`)
- `docs/design/night-bloom/b1-cycle-log-analysis/nbl_Log_Contraction.dc.html` (+ `nbd_Log_Contraction`)

## Scope
- Kick counter (tap per movement, 10 target, elapsed, 2-hour guidance); contraction timer (tap start/stop, averages, 5-1-1 guidance with call CTA). Survive page reload (persist running session).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
