---
id: CB-MENO-10
title: Frontend: treatment & care
epic: MENO
type: frontend
status: done
depends_on: [CB-MENO-05, CB-MENO-03]
parallel_group: MENO-D
touches: [frontend/src/screens/menopause-treatment, frontend/src/app/[locale]/menopause/treatment]
skills: [new-fsd-slice]
boards: [nbl_Meno_Treatment.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-MENO-10 — Frontend: treatment & care

## Why
HRT, supplements, lifestyle.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_Treatment.dc.html`

## Scope
1. Sections per board: HRT items with intake toggles + weekly dots + review date, side-effect chips + spotting note, supplements, lifestyle progress, 'only with your doctor' note.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
