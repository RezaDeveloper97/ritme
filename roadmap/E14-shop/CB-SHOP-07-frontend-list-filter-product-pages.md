---
id: CB-SHOP-07
title: Frontend: list/filter + product pages
epic: SHOP
type: frontend
status: todo
depends_on: [CB-SHOP-06]
parallel_group: SHOP-F
touches: [frontend/src/screens/shop-list, frontend/src/screens/shop-product, frontend/src/app/[locale]/(app)/services/shop/list, frontend/src/app/[locale]/(app)/services/shop/product]
skills: [new-fsd-slice]
boards: [nbd_Shop_List.dc.html, nbd_Shop_Product.dc.html, nbd_Shop_ProductBeauty.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-SHOP-07 — Frontend: list/filter + product pages

## Why
Browse and pick variants.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbd_Shop_List.dc.html`
- `nbd_Shop_Product.dc.html`
- `nbd_Shop_ProductBeauty.dc.html`

## Scope
1. List (search, filters, sub-tabs, sort, grid). Baby product (gallery, seller, rating, color/size + guide, material/wash/shipping/return, reviews, bought-together, sticky add). Beauty product (volume, skin type, IRC + expiry, ingredients, pregnancy/breastfeeding caution, reviews).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
