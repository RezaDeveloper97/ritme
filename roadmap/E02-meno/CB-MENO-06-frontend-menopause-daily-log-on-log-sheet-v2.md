---
id: CB-MENO-06
title: Frontend: menopause daily log on log sheet v2
epic: MENO
type: frontend
status: todo
depends_on: [CB-MENO-05, B-N3-03]
parallel_group: MENO-D
touches: [frontend/src/screens/menopause-log, frontend/src/app/[locale]/menopause/log, frontend/src/screens/log]
skills: [new-fsd-slice]
boards: [nbl_Meno_Log.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-MENO-06 — Frontend: menopause daily log on log sheet v2

## Why
The board is a full-page log; bloom's log sheet v2 renders taxonomy items.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_Log.dc.html`

## Scope
1. Menopause preset of log sheet v2 + full-page variant per board: grouped SeverityScale rows, bleeding choice with post-menopause note (→ Alert), trigger chips, 'say it by voice' card (bloom voice log), save/outbox.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
