---
id: CB-CONTRA-02
title: Frontend: method setup + pill pack
epic: CONTRA
type: frontend
status: done
depends_on: [CB-CONTRA-01]
parallel_group: CONTRA-B
touches: [frontend/src/screens/contraception, frontend/src/screens/contraception-setup, frontend/src/entities/contraception, frontend/messages/fa/contraception.json, frontend/messages/en/contraception.json, frontend/src/app/[locale]/contraception, frontend/src/app/message-scopes.ts, frontend/src/screens/mode]
skills: [new-fsd-slice]
boards: [nbl_Contra_Setup.dc.html, nbl_Contra_Pill.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-CONTRA-02 — Frontend: method setup + pill pack

## Why
The switch in the mode screen opens setup; pill users get a pack view.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Contra_Setup.dc.html`
- `nbl_Contra_Pill.dc.html`

## Scope
1. Setup (method chips, pack type, start, reminder time). Pill (today card, 'I missed one', pack grid taken/today/placebo, streak + next-pack tiles, refill row).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
