---
id: CB-COND-02
title: Frontend: programs hub (real programs behind bloom's services tiles)
epic: COND
type: frontend
status: todo
depends_on: [CB-COND-01, B-N7-01, CB-COND-06b]
parallel_group: COND-B
touches: [frontend/src/screens/conditions, frontend/src/entities/condition, frontend/messages/fa/conditions.json, frontend/messages/en/conditions.json, frontend/src/app/[locale]/programs, frontend/src/app/message-scopes.ts, frontend/src/screens/services]
skills: [new-fsd-slice]
boards: [nbl_Cond_Hub.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-COND-02 — Frontend: programs hub (real programs behind bloom's services tiles)

## Why
Services › برنامه‌های مراقبتی now opens real programs.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Cond_Hub.dc.html`

## Scope
1. Program cards with what-you-log line and active state; enrol/leave; not-a-diagnosis note; PCOS card links to analysis. Services tiles route here.
2. (CB-COND-06b) Show active nudges from `GET /api/v1/messages/nudges` as a soft card on the hub (and on the cycle home if bloom's home has a slot for it — otherwise note it for CB-COND-07).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
