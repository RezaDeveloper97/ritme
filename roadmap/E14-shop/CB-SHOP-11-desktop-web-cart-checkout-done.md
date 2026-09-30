---
id: CB-SHOP-11
title: Desktop web: cart, checkout, done
epic: SHOP
type: frontend
status: todo
depends_on: [CB-SHOP-10, CB-SHOP-08]
parallel_group: SHOP-I
touches: [frontend/src/screens/web-shop-checkout, frontend/src/app/[locale]/(web)/shop/checkout]
skills: [new-fsd-slice]
boards: [W_Shop_Cart.dc.html, W_Shop_Checkout.dc.html, W_Shop_Done.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-SHOP-11 — Desktop web: cart, checkout, done

## Why
Desktop checkout (COD).

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `W_Shop_Cart.dc.html`
- `W_Shop_Checkout.dc.html`
- `W_Shop_Done.dc.html`

## Scope
1. Cart, checkout, order-placed pages on the app APIs.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Screens match the listed boards (layout, hierarchy, copy, components, flows) using the app's CURRENT light and dark tokens — never the board hex colors; light + dark screenshots next to the board render are in `docs/qa/canvas/<epic>.md` (skill §5).
- `verify` green.
