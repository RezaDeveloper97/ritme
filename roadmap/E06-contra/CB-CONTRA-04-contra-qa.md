---
id: CB-CONTRA-04
title: CONTRA QA
epic: CONTRA
type: qa
status: done
depends_on: [CB-CONTRA-03]
parallel_group: CONTRA-D
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbl_Contra_Setup.dc.html, nbl_Contra_Pill.dc.html, nbl_Contra_Missed.dc.html, nbl_Contra_Other.dc.html]
verify: test -s docs/qa/canvas/contra.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-CONTRA-04 — CONTRA QA

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Contra_Setup.dc.html`
- `nbl_Contra_Pill.dc.html`
- `nbl_Contra_Missed.dc.html`
- `nbl_Contra_Other.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/contra.md` (board · route · light · dark · verdict · fix). Pill + injection reminders fire locally.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/contra.md, no open ✘.
- `verify` green.
