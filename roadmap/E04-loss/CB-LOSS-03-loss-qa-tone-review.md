---
id: CB-LOSS-03
title: LOSS QA + tone review
epic: LOSS
type: qa
status: todo
depends_on: [CB-LOSS-02]
parallel_group: LOSS-C
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbl_Loss_Start.dc.html, nbl_Loss_Care.dc.html, nbl_Loss_Next.dc.html]
verify: test -s docs/qa/canvas/loss.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-LOSS-03 — LOSS QA + tone review

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Loss_Start.dc.html`
- `nbl_Loss_Care.dc.html`
- `nbl_Loss_Next.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/loss.md` (board · route · light · dark · verdict · fix). Verify pregnancy content/reminders stop; companion sees one line only.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/loss.md, no open ✘.
- `verify` green.
