---
id: CB-IVF-05
title: Frontend: two-week wait + outcome
epic: IVF
type: frontend
status: done
depends_on: [CB-IVF-02, CB-LOSS-02]
parallel_group: IVF-C
touches: [frontend/src/screens/ivf-tww, frontend/src/app/[locale]/ivf/tww]
skills: [new-fsd-slice]
boards: [nbl_IVF_TWW.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-IVF-05 — Frontend: two-week wait + outcome

## Why
Waiting, mood, meds, danger signs, result.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_IVF_TWW.dc.html`

## Scope
1. Countdown to beta, mood chips, early-test note, progesterone doses, OHSS danger card, result buttons: positive → pregnancy setup; negative → LOSS start or back to cycle.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
