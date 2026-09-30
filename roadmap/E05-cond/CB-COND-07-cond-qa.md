---
id: CB-COND-07
title: COND QA
epic: COND
type: qa
status: todo
depends_on: [CB-COND-03, CB-COND-04, CB-COND-05, CB-COND-06]
parallel_group: COND-D
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbl_Cond_Hub.dc.html, nbl_Cond_Endo.dc.html, nbl_Cond_PMDD.dc.html, nbl_Cond_Bleed.dc.html]
verify: test -s docs/qa/canvas/cond.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-COND-07 — COND QA

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Cond_Hub.dc.html`
- `nbl_Cond_Endo.dc.html`
- `nbl_Cond_PMDD.dc.html`
- `nbl_Cond_Bleed.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/cond.md` (board · route · light · dark · verdict · fix). Enrol each program, log 3 days, see summaries.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/cond.md, no open ✘.
- `verify` green.
