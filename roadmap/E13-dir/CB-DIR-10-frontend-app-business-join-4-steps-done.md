---
id: CB-DIR-10
title: Frontend (app): business join 4 steps + done
epic: DIR
type: frontend
status: todo
depends_on: [CB-DIR-04, CB-DIR-06]
parallel_group: DIR-G
touches: [frontend/src/screens/city-services-join, frontend/src/app/[locale]/services/mother-child/join]
skills: [new-fsd-slice]
boards: [nbl_Dir_Join.dc.html, nbl_Dir_JoinForm.dc.html, nbl_Dir_JoinDocs.dc.html, nbl_Dir_JoinDone.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-DIR-10 — Frontend (app): business join 4 steps + done

## Why
Owners apply from the app.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Dir_Join.dc.html`
- `nbl_Dir_JoinForm.dc.html`
- `nbl_Dir_JoinDocs.dc.html`
- `nbl_Dir_JoinDone.dc.html`

## Scope
1. Stepper with auto-saved draft: intro, place & photos (Neshan pin), services & booking mode, documents + terms; done screen with tracking timeline.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
