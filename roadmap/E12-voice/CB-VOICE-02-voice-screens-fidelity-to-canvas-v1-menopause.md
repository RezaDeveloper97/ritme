---
id: CB-VOICE-02
title: Voice screens fidelity to canvas-v1 + menopause entry
epic: VOICE
type: frontend
status: todo
depends_on: [CB-VOICE-01, CB-MENO-06]
parallel_group: VOICE-B
touches: [frontend/src/features/voice-log, frontend/src/screens/log, frontend/src/screens/menopause-log]
skills: [new-fsd-slice]
boards: [nbl_Voice_Entry.dc.html, nbl_Voice_Record.dc.html, nbl_Voice_Review.dc.html, nbl_Voice_Saved.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-VOICE-02 — Voice screens fidelity to canvas-v1 + menopause entry

## Why
Align bloom's voice UI with this canvas where they differ.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Voice_Entry.dc.html`
- `nbl_Voice_Record.dc.html`
- `nbl_Voice_Review.dc.html`
- `nbl_Voice_Saved.dc.html`

## Scope
1. Diff bloom voice screens vs the 4 boards (examples, today's status, live text, quote under each item, ambiguity chooser, saved list, nightly 21:00 reminder opt-in) and close gaps; menopause 'say it by voice' card opens it.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
