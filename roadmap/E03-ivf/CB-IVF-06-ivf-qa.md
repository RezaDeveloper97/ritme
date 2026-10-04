---
id: CB-IVF-06
title: IVF QA
epic: IVF
type: qa
status: done
depends_on: [CB-IVF-03, CB-IVF-04, CB-IVF-05]
parallel_group: IVF-D
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbl_IVF_Home.dc.html, nbl_IVF_Meds.dc.html, nbl_IVF_Scan.dc.html, nbl_IVF_TWW.dc.html]
verify: test -s docs/qa/canvas/ivf.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-IVF-06 — IVF QA

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_IVF_Home.dc.html`
- `nbl_IVF_Meds.dc.html`
- `nbl_IVF_Scan.dc.html`
- `nbl_IVF_TWW.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/ivf.md` (board · route · light · dark · verdict · fix). Journey: ttc → IVF on → meds → scan → TWW → both outcomes.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/ivf.md, no open ✘.
- `verify` green.
