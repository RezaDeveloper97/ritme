---
id: CB-IVF-02
title: Frontend: IVF home = stage tab 'درمان'
epic: IVF
type: frontend
status: todo
depends_on: [CB-IVF-01, CB-CORE-02]
parallel_group: IVF-B
touches: [frontend/src/screens/ivf, frontend/src/entities/ivf, frontend/messages/fa/ivf.json, frontend/messages/en/ivf.json, frontend/src/app/[locale]/(app)/ivf, frontend/src/app/message-scopes.ts]
skills: [new-fsd-slice]
boards: [nbl_IVF_Home.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-IVF-02 — Frontend: IVF home = stage tab 'درمان'

## Why
Treatment timeline + today's injections.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_IVF_Home.dc.html`

## Scope
1. StepTimeline (6 stages), today's injections (done/log), next appointment, companion reminder toggle (hidden without a linked companion).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
