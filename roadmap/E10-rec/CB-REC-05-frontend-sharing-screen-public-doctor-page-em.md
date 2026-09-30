---
id: CB-REC-05
title: Frontend: sharing screen, public doctor page, emergency card
epic: REC
type: frontend
status: todo
depends_on: [CB-REC-03, CB-REC-04]
parallel_group: REC-D
touches: [frontend/src/screens/record-share, frontend/src/screens/record-emergency, frontend/src/screens/doctor-view, frontend/src/app/[locale]/(app)/record/share, frontend/src/app/[locale]/(app)/record/emergency, frontend/src/app/[locale]/(web)/d]
skills: [new-fsd-slice]
boards: [nbl_Rec_Share.dc.html, nbl_Rec_Emergency.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-REC-05 — Frontend: sharing screen, public doctor page, emergency card

## Why
Who sees my record + emergency card.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Rec_Share.dc.html`
- `nbl_Rec_Emergency.dc.html`

## Scope
1. Share per board (doctor code + QR, insurer rows, family row, history). Public /d/[token] read-only page (noindex, no nav). Emergency card + lock-screen toggle.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
