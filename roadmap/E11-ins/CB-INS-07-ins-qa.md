---
id: CB-INS-07
title: INS QA
epic: INS
type: qa
status: todo
depends_on: [CB-INS-04, CB-INS-05, CB-INS-06]
parallel_group: INS-E
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbl_Ins_Home.dc.html, nbl_Ins_Coverage.dc.html, nbl_Ins_Claims.dc.html, nbl_Ins_ClaimNew.dc.html, nbl_Ins_ClaimDetail.dc.html, nbl_Ins_Health.dc.html, nbl_Ins_Status.dc.html, nbl_Ins_Centers.dc.html]
verify: test -s docs/qa/canvas/ins.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-INS-07 — INS QA

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Ins_Home.dc.html`
- `nbl_Ins_Coverage.dc.html`
- `nbl_Ins_Claims.dc.html`
- `nbl_Ins_ClaimNew.dc.html`
- `nbl_Ins_ClaimDetail.dc.html`
- `nbl_Ins_Health.dc.html`
- `nbl_Ins_Status.dc.html`
- `nbl_Ins_Centers.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/ins.md` (board · route · light · dark · verdict · fix). Journey: policy → claim with record doc → docs requested → attach → paid → coverage updates.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/ins.md, no open ✘.
- `verify` green.
