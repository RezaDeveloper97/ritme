---
id: CB-VOICE-03
title: VOICE QA
epic: VOICE
type: qa
status: done
depends_on: [CB-VOICE-02]
parallel_group: VOICE-C
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbl_Voice_Entry.dc.html, nbl_Voice_Record.dc.html, nbl_Voice_Review.dc.html, nbl_Voice_Saved.dc.html]
verify: test -s docs/qa/canvas/voice.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-VOICE-03 — VOICE QA

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Voice_Entry.dc.html`
- `nbl_Voice_Record.dc.html`
- `nbl_Voice_Review.dc.html`
- `nbl_Voice_Saved.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/voice.md` (board · route · light · dark · verdict · fix). 10 Persian fixtures (cycle + menopause) → accuracy table.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/voice.md, no open ✘.
- `verify` green.
