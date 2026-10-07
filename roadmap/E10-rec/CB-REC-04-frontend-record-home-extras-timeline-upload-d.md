---
id: CB-REC-04
title: Frontend: record home extras, timeline, upload, document detail
epic: REC
type: frontend
status: done
depends_on: [CB-REC-02, B-N6-03]
parallel_group: REC-C
touches: [frontend/src/screens/health-record, frontend/src/screens/record-timeline, frontend/src/screens/record-document, frontend/src/entities/health-record, frontend/messages/fa/record.json, frontend/messages/en/record.json, frontend/src/app/[locale]/record, frontend/src/app/message-scopes.ts]
skills: [new-fsd-slice]
boards: [nbl_Rec_Home.dc.html, nbl_Rec_Timeline.dc.html, nbl_Rec_Doc.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-REC-04 — Frontend: record home extras, timeline, upload, document detail

## Why
Adds the canvas parts to bloom's record screen.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Rec_Home.dc.html`
- `nbl_Rec_Timeline.dc.html`
- `nbl_Rec_Doc.dc.html`

## Scope
1. Home: category grid with counts, allergy card, insurance links (after INS), pending-document card. Timeline: chips, month groups, claim badges. Upload sheet (camera/file) + detail (preview, editable extracted fields or manual form for free users, where-used rows).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
