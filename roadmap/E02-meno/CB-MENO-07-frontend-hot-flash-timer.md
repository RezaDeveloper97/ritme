---
id: CB-MENO-07
title: Frontend: hot-flash timer
epic: MENO
type: frontend
status: todo
depends_on: [CB-MENO-05, CB-CORE-02]
parallel_group: MENO-D
touches: [frontend/src/screens/menopause-hot-flash, frontend/src/app/[locale]/menopause/hot-flash]
skills: [new-fsd-slice]
boards: [nbl_Meno_HotFlash.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-MENO-07 — Frontend: hot-flash timer

## Why
One tap now, tap again when over.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_HotFlash.dc.html`

## Scope
1. CountdownRing timer (start persisted server-side), severity chips, cause chips, today tiles (count, avg duration, night), today list, breathing tip.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
