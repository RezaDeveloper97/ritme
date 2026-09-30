---
id: CB-INS-05
title: Frontend: health questionnaire + status
epic: INS
type: frontend
status: todo
depends_on: [CB-INS-03]
parallel_group: INS-D
touches: [frontend/src/screens/insurance-questionnaire, frontend/src/screens/insurance-status, frontend/src/app/[locale]/(app)/insurance/questionnaire, frontend/src/app/[locale]/(app)/insurance/status]
skills: [new-fsd-slice]
boards: [nbl_Ins_Health.dc.html, nbl_Ins_Status.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-INS-05 — Frontend: health questionnaire + status

## Why
Prefilled questionnaire and a status page.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Ins_Health.dc.html`
- `nbl_Ins_Status.dc.html`

## Scope
1. Section stepper, yes/no rows with 'from record' tags, explanation, warning, PDF export; status timelines.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
