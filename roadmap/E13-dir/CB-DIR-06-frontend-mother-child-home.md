---
id: CB-DIR-06
title: Frontend: mother & child home
epic: DIR
type: frontend
status: todo
depends_on: [CB-DIR-02, B-N7-01]
parallel_group: DIR-E
touches: [frontend/src/screens/city-services, frontend/src/entities/city-service, frontend/messages/fa/city-services.json, frontend/messages/en/city-services.json, frontend/src/app/[locale]/(app)/services/mother-child, frontend/src/app/message-scopes.ts]
skills: [new-fsd-slice]
boards: [nbl_Dir_Home.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-DIR-06 — Frontend: mother & child home

## Why
Services › مادر و کودک.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Dir_Home.dc.html`

## Scope
1. Location header, child chips, search, next booking, age-matched carousel (toggle in settings), nearby + map link, join CTA, ranking + age notes.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
