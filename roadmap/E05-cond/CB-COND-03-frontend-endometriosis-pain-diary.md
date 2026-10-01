---
id: CB-COND-03
title: Frontend: endometriosis pain diary
epic: COND
type: frontend
status: todo
depends_on: [CB-COND-02, B-N3-03, CB-CORE-02]
parallel_group: COND-C
touches: [frontend/src/screens/pain-diary, frontend/src/app/[locale]/programs/pain]
skills: [new-fsd-slice]
boards: [nbl_Cond_Endo.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-COND-03 — Frontend: endometriosis pain diary

## Why
Daily pain with location, severity, type, impact, medication.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Cond_Endo.dc.html`

## Scope
1. Body-map/location chips (bloom body map), NumericScale 0–10, type + associated chips, impact toggle, analgesic row + effect.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
