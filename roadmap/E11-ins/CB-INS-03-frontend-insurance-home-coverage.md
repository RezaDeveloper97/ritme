---
id: CB-INS-03
title: Frontend: insurance home + coverage
epic: INS
type: frontend
status: todo
depends_on: [CB-INS-01]
parallel_group: INS-C
touches: [frontend/src/screens/insurance, frontend/src/screens/insurance-coverage, frontend/src/entities/insurance, frontend/messages/fa/insurance.json, frontend/messages/en/insurance.json, frontend/src/app/[locale]/insurance, frontend/src/app/message-scopes.ts]
skills: [new-fsd-slice]
boards: [nbl_Ins_Home.dc.html, nbl_Ins_Coverage.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-INS-03 — Frontend: insurance home + coverage

## Why
Replaces the info page entry with the tracker.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Ins_Home.dc.html`
- `nbl_Ins_Coverage.dc.html`

## Scope
0. Build ON bloom: `/insurance` is B-N10-06's info page route/screen (`frontend/src/screens/insurance`). Extend that screen in place: when no policy exists show setup form + a link to the info content; when one exists show the tracker and keep «how it works» reachable (e.g. a link/section). Do not create a second `/insurance` route or duplicate the info content. The services tile (hidden while info is unpublished) must also show once a policy exists.
1. Home (+ setup form when empty): policy card, action card, remaining coverage bars, recent claims. Coverage: deductible tiles, caps, waiting notes, not-covered note, policy document link.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
