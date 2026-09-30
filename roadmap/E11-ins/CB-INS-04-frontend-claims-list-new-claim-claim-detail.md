---
id: CB-INS-04
title: Frontend: claims list, new claim, claim detail
epic: INS
type: frontend
status: todo
depends_on: [CB-INS-03, CB-REC-04]
parallel_group: INS-D
touches: [frontend/src/screens/insurance-claims, frontend/src/screens/insurance-claim-new, frontend/src/screens/insurance-claim, frontend/src/app/[locale]/(app)/insurance/claims]
skills: [new-fsd-slice]
boards: [nbl_Ins_Claims.dc.html, nbl_Ins_ClaimNew.dc.html, nbl_Ins_ClaimDetail.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-INS-04 — Frontend: claims list, new claim, claim detail

## Why
Self-tracked claims with record documents.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Ins_Claims.dc.html`
- `nbl_Ins_ClaimNew.dc.html`
- `nbl_Ins_ClaimDetail.dc.html`

## Scope
1. List (year stats, search, chips, month groups). New (member chips, type chips, centre, date, amount, docs from record + camera, masked IBAN). Detail (payable breakdown, note, StepTimeline she advances, attach missing doc, dispute note).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
