---
id: CB-SHOP-08
title: Frontend: cart, checkout, order placed, my orders
epic: SHOP
type: frontend
status: todo
depends_on: [CB-SHOP-07, CB-SHOP-03]
parallel_group: SHOP-G
touches: [frontend/src/screens/shop-cart, frontend/src/screens/shop-checkout, frontend/src/screens/shop-order, frontend/src/screens/my-orders, frontend/src/app/[locale]/(app)/services/shop/cart, frontend/src/app/[locale]/(app)/services/shop/checkout, frontend/src/app/[locale]/(app)/me/orders]
skills: [new-fsd-slice]
boards: [nbd_Shop_Cart.dc.html, nbd_Shop_Checkout.dc.html, nbd_Shop_Order.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-SHOP-08 — Frontend: cart, checkout, order placed, my orders

## Why
Per-seller cart, slots, COD, tracking.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbd_Shop_Cart.dc.html`
- `nbd_Shop_Checkout.dc.html`
- `nbd_Shop_Order.dc.html`

## Scope
1. Cart (seller groups, qty, discount, totals, free-shipping hint if configured). Checkout (stepper, address, recipient, per-seller slots, discreet packaging, pay on delivery only). Order placed (per-seller timelines, checklist link, returns). My orders in Me.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- fa + en message keys; RTL correct; no inline colors (lint:styles, lint:dark green).
- Bottom-nav visibility follows IA_Nav rules (hidden on forms / flows / sensitive paths).
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
