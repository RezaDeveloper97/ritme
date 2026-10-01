---
id: CB-PELV-02
title: Frontend: plan, bladder diary, Kegel trainer
epic: PELV
type: frontend
status: todo
depends_on: [CB-PELV-01, CB-COND-02, CB-CORE-02]
parallel_group: PELV-B
touches: [backend-go/resources/translations, frontend/src/screens/pelvic, frontend/src/screens/pelvic-kegel, frontend/src/entities/pelvic, frontend/messages/fa/pelvic.json, frontend/messages/en/pelvic.json, frontend/src/app/[locale]/programs/pelvic, frontend/src/app/message-scopes.ts]
skills: [new-fsd-slice]
boards: [nbl_Pelvic_Plan.dc.html, nbl_Pelvic_Kegel.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-PELV-02 — Frontend: plan, bladder diary, Kegel trainer

## Why
Plan card, week dots, diary, guided timer.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Pelvic_Plan.dc.html`
- `nbl_Pelvic_Kegel.dc.html`

## Scope
1. Plan per board; Kegel trainer full-screen with CountdownRing hold/relax, rep/set counters, stop, instruction; reduced-motion; wake lock where supported; listed in the programs hub.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
