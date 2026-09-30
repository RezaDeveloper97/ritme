---
id: CB-DIR-11
title: Desktop web: search+map, place, booked
epic: DIR
type: frontend
status: todo
depends_on: [CB-DIR-07, CB-DIR-08, CB-DIR-09]
parallel_group: DIR-H
touches: [frontend/src/screens/web-city-services, frontend/src/widgets/web-shell, frontend/src/app/[locale]/(web)]
skills: [new-fsd-slice]
boards: [W_Dir_Search.dc.html, W_Dir_Place.dc.html, W_Dir_Booked.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-DIR-11 — Desktop web: search+map, place, booked

## Why
Public SEO pages in the same frontend (DECISIONS: web only, no native app).

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `W_Dir_Search.dc.html`
- `W_Dir_Place.dc.html`
- `W_Dir_Booked.dc.html`

## Scope
1. Desktop web shell (header nav, login, footer; store badges only if links exist). Search: filter sidebar + results + sticky map. Place: gallery, details, booking panel (login to send). Booked page. SSR metadata; <1024px falls back to the app screens.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Lighthouse SEO ≥ 90 on place page; Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
