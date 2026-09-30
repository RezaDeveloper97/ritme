---
id: CB-COND-05
title: Frontend: heavy bleeding (PBAC) chart
epic: COND
type: frontend
status: todo
depends_on: [CB-COND-02]
parallel_group: COND-C
touches: [frontend/src/screens/pbac, frontend/src/app/[locale]/(app)/programs/bleeding]
skills: [new-fsd-slice]
boards: [nbl_Cond_Bleed.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-COND-05 — Frontend: heavy bleeding (PBAC) chart

## Why
Pictorial blood-loss chart.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Cond_Bleed.dc.html`

## Scope
1. NumberStepper per soak level with points, clot chips, flooding toggle, today/period tiles, ≥100 warning.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
