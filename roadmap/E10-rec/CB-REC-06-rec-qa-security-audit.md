---
id: CB-REC-06
title: REC QA + security audit
epic: REC
type: qa
status: todo
depends_on: [CB-REC-05]
parallel_group: REC-E
touches: [docs/qa/canvas, docs/security]
skills: [verify-all]
boards: [nbl_Rec_Home.dc.html, nbl_Rec_Timeline.dc.html, nbl_Rec_Doc.dc.html, nbl_Rec_Share.dc.html, nbl_Rec_Emergency.dc.html]
verify: test -s docs/qa/canvas/rec.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-REC-06 — REC QA + security audit

## Why
Health documents are the most sensitive data.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Rec_Home.dc.html`
- `nbl_Rec_Timeline.dc.html`
- `nbl_Rec_Doc.dc.html`
- `nbl_Rec_Share.dc.html`
- `nbl_Rec_Emergency.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/rec.md` (board · route · light · dark · verdict · fix). Run security-auditor on files/healthrecord/sharelinks; write docs/security/canvas-record-review.md.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- No open high findings; docs/qa/canvas/rec.md, no open ✘.
- `verify` green.
