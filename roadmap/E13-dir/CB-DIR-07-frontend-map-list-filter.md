---
id: CB-DIR-07
title: Frontend: map + list/filter
epic: DIR
type: frontend
status: todo
depends_on: [CB-DIR-06, CB-CORE-06]
parallel_group: DIR-F
touches: [frontend/src/screens/city-services-map, frontend/src/screens/city-services-list, frontend/src/app/[locale]/(app)/services/mother-child/map, frontend/src/app/[locale]/(app)/services/mother-child/list]
skills: [new-fsd-slice]
boards: [nbl_Dir_Map.dc.html, nbl_Dir_List.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-DIR-07 — Frontend: map + list/filter

## Why
Map or list browsing.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Dir_Map.dc.html`
- `nbl_Dir_List.dc.html`

## Scope
1. Map (pins, search-this-area, category chips, bottom cards, list toggle). List (search, filter chips, sort, cards with verified badge, rating, price-from, next slot).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
