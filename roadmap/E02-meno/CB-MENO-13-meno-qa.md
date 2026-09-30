---
id: CB-MENO-13
title: MENO QA
epic: MENO
type: qa
status: todo
depends_on: [CB-MENO-04, CB-MENO-06, CB-MENO-07, CB-MENO-08, CB-MENO-09, CB-MENO-10, CB-MENO-11, CB-MENO-12]
parallel_group: MENO-E
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbl_Meno_Home.dc.html, nbl_Meno_Stage.dc.html, nbl_Meno_Log.dc.html, nbl_Meno_HotFlash.dc.html, nbl_Meno_Alert.dc.html, nbl_Meno_Score.dc.html, nbl_Meno_Checkups.dc.html, nbl_Meno_Treatment.dc.html, nbl_Meno_Report.dc.html]
verify: test -s docs/qa/canvas/meno.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-MENO-13 — MENO QA

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_Home.dc.html`
- `nbl_Meno_Stage.dc.html`
- `nbl_Meno_Log.dc.html`
- `nbl_Meno_HotFlash.dc.html`
- `nbl_Meno_Alert.dc.html`
- `nbl_Meno_Score.dc.html`
- `nbl_Meno_Checkups.dc.html`
- `nbl_Meno_Treatment.dc.html`
- `nbl_Meno_Report.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/meno.md` (board · route · light · dark · verdict · fix). Journey: switch to menopause → stage → log → hot flash → score → report PDF.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/meno.md, no open ✘.
- `verify` green.
