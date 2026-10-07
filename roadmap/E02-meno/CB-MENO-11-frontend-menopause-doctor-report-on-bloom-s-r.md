---
id: CB-MENO-11
title: Frontend: menopause doctor report (on bloom's report builder)
epic: MENO
type: frontend
status: done
depends_on: [CB-MENO-03, CB-MENO-05, B-N6-04]
parallel_group: MENO-D
touches: [frontend/src/screens/menopause-report, frontend/src/app/[locale]/menopause/report, frontend/src/screens/record-export]
skills: [new-fsd-slice]
boards: [nbl_Meno_Report.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-MENO-11 — Frontend: menopause doctor report (on bloom's report builder)

## Why
1/3/6-month report for the doctor.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_Report.dc.html`

## Scope
1. Board layout (range control, summary rows, top-symptom bars, meds, editable questions) feeding B-N6-04's PDF + share link; entry from menopause home and Alert.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
