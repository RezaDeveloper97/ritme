---
id: CB-PELV-03
title: PELV QA
epic: PELV
type: qa
status: todo
depends_on: [CB-PELV-02]
parallel_group: PELV-C
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbl_Pelvic_Plan.dc.html, nbl_Pelvic_Kegel.dc.html]
verify: test -s docs/qa/canvas/pelv.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-PELV-03 — PELV QA

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Pelvic_Plan.dc.html`
- `nbl_Pelvic_Kegel.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/pelv.md` (board · route · light · dark · verdict · fix).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/pelv.md, no open ✘.
- `verify` green.
