---
id: CB-NAV-04
title: NAV QA
epic: NAV
type: qa
status: todo
depends_on: [CB-NAV-03]
parallel_group: NAV-D
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbd_Nav_Today.dc.html, nbd_Nav_Stage.dc.html, nbd_Nav_Plus.dc.html, nbd_Nav_Services.dc.html, nbd_Nav_Me.dc.html, nbd_Nav_Mode.dc.html, nbd_Nav_Search.dc.html]
verify: test -s docs/qa/canvas/nav.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-NAV-04 — NAV QA

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbd_Nav_Today.dc.html`
- `nbd_Nav_Stage.dc.html`
- `nbd_Nav_Plus.dc.html`
- `nbd_Nav_Services.dc.html`
- `nbd_Nav_Me.dc.html`
- `nbd_Nav_Mode.dc.html`
- `nbd_Nav_Search.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/nav.md` (board · route · light · dark · verdict · fix). Walk every mode's tabs.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/nav.md, no open ✘.
- `verify` green.
