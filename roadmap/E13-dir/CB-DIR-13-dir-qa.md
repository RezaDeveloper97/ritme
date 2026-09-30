---
id: CB-DIR-13
title: DIR QA
epic: DIR
type: qa
status: todo
depends_on: [CB-DIR-05, CB-DIR-09, CB-DIR-12]
parallel_group: DIR-J
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbl_Dir_Home.dc.html, nbl_Dir_Map.dc.html, nbl_Dir_List.dc.html, nbl_Dir_Place.dc.html, nbl_Dir_Book.dc.html, nbl_Dir_Booked.dc.html, nbl_Dir_MyBookings.dc.html, nbl_Dir_Join.dc.html, nbl_Dir_JoinForm.dc.html, nbl_Dir_JoinDocs.dc.html, nbl_Dir_JoinDone.dc.html, W_Dir_Search.dc.html, W_Dir_Place.dc.html, W_Dir_Booked.dc.html, W_Dir_Business.dc.html, W_Dir_Join.dc.html, W_Dir_JoinDone.dc.html]
verify: test -s docs/qa/canvas/dir.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-DIR-13 — DIR QA

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Dir_Home.dc.html`
- `nbl_Dir_Map.dc.html`
- `nbl_Dir_List.dc.html`
- `nbl_Dir_Place.dc.html`
- `nbl_Dir_Book.dc.html`
- `nbl_Dir_Booked.dc.html`
- `nbl_Dir_MyBookings.dc.html`
- `nbl_Dir_Join.dc.html`
- `nbl_Dir_JoinForm.dc.html`
- `nbl_Dir_JoinDocs.dc.html`
- `nbl_Dir_JoinDone.dc.html`
- `W_Dir_Search.dc.html`
- `W_Dir_Place.dc.html`
- `W_Dir_Booked.dc.html`
- `W_Dir_Business.dc.html`
- `W_Dir_Join.dc.html`
- `W_Dir_JoinDone.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/dir.md` (board · route · light · dark · verdict · fix). Web boards in light + dark too. Journey: apply → approve → slots → book → confirm → reminder → review.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/dir.md, no open ✘.
- `verify` green.
