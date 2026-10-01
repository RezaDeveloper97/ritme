---
id: CB-DIR-08
title: Frontend: place page
epic: DIR
type: frontend
status: todo
depends_on: [CB-DIR-06]
parallel_group: DIR-F
touches: [frontend/src/screens/city-services-place, frontend/src/app/[locale]/services/mother-child/place]
skills: [new-fsd-slice]
boards: [nbl_Dir_Place.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-DIR-08 — Frontend: place page

## Why
Everything about one place.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Dir_Place.dc.html`

## Scope
1. Gallery, verified badge, rating, open state, actions (call, directions → Neshan link, website), about, amenities, services & prices, nearest slots, health note, reviews with sub-scores, address/hours/rules/cancel policy, owner/report links, sticky choose-time bar.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
