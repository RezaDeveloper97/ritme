---
id: CB-DIR-12
title: Desktop web: for-businesses landing + join + done
epic: DIR
type: frontend
status: todo
depends_on: [CB-DIR-10, CB-DIR-11]
parallel_group: DIR-I
touches: [frontend/src/screens/web-city-services-business, frontend/src/app/[locale]/(web)/business]
skills: [new-fsd-slice]
boards: [W_Dir_Business.dc.html, W_Dir_Join.dc.html, W_Dir_JoinDone.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-DIR-12 — Desktop web: for-businesses landing + join + done

## Why
Acquire businesses from the web.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `W_Dir_Business.dc.html`
- `W_Dir_Join.dc.html`
- `W_Dir_JoinDone.dc.html`

## Scope
1. Landing (hero, why, 4 steps, requirements, cooperation terms as admin-editable text, FAQ), desktop wizard on the app draft API (save & exit), done page.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
