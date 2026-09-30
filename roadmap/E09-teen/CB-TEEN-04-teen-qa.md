---
id: CB-TEEN-04
title: TEEN QA
epic: TEEN
type: qa
status: todo
depends_on: [CB-TEEN-03]
parallel_group: TEEN-D
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbl_Teen_Onb.dc.html, nbl_Teen_Home.dc.html, nbl_Teen_Parent.dc.html]
verify: test -s docs/qa/canvas/teen.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-TEEN-04 — TEEN QA

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Teen_Onb.dc.html`
- `nbl_Teen_Home.dc.html`
- `nbl_Teen_Parent.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/teen.md` (board · route · light · dark · verdict · fix). No shop/banners anywhere in teen mode; two-account scope-leak check.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/teen.md, no open ✘.
- `verify` green.
