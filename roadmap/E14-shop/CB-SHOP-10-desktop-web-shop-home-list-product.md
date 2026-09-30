---
id: CB-SHOP-10
title: Desktop web: shop home, list, product
epic: SHOP
type: frontend
status: todo
depends_on: [CB-SHOP-07, CB-DIR-11]
parallel_group: SHOP-H
touches: [frontend/src/screens/web-shop, frontend/src/app/[locale]/(web)/shop]
skills: [new-fsd-slice]
boards: [W_Shop_Home.dc.html, W_Shop_List.dc.html, W_Shop_Product.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-SHOP-10 — Desktop web: shop home, list, product

## Why
Public desktop shop on the web shell from CB-DIR-11.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `W_Shop_Home.dc.html`
- `W_Shop_List.dc.html`
- `W_Shop_Product.dc.html`

## Scope
1. Home, list with filter sidebar, product; SSR metadata; add-to-cart needs login.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Lighthouse SEO ≥ 90 on product; Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
