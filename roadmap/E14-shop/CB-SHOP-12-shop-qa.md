---
id: CB-SHOP-12
title: SHOP QA
epic: SHOP
type: qa
status: todo
depends_on: [CB-SHOP-05, CB-SHOP-08, CB-SHOP-09, CB-SHOP-11]
parallel_group: SHOP-J
touches: [docs/qa/canvas]
skills: [verify-all]
boards: [nbd_Shop_Home.dc.html, nbd_Shop_Beauty.dc.html, nbd_Shop_List.dc.html, nbd_Shop_Product.dc.html, nbd_Shop_ProductBeauty.dc.html, nbd_Shop_Checklist.dc.html, nbd_Shop_Cart.dc.html, nbd_Shop_Checkout.dc.html, nbd_Shop_Order.dc.html, W_Shop_Home.dc.html, W_Shop_List.dc.html, W_Shop_Product.dc.html, W_Shop_Cart.dc.html, W_Shop_Checkout.dc.html, W_Shop_Done.dc.html]
verify: test -s docs/qa/canvas/shop.md && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-SHOP-12 — SHOP QA

## Why
Close the epic.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbd_Shop_Home.dc.html`
- `nbd_Shop_Beauty.dc.html`
- `nbd_Shop_List.dc.html`
- `nbd_Shop_Product.dc.html`
- `nbd_Shop_ProductBeauty.dc.html`
- `nbd_Shop_Checklist.dc.html`
- `nbd_Shop_Cart.dc.html`
- `nbd_Shop_Checkout.dc.html`
- `nbd_Shop_Order.dc.html`
- `W_Shop_Home.dc.html`
- `W_Shop_List.dc.html`
- `W_Shop_Product.dc.html`
- `W_Shop_Cart.dc.html`
- `W_Shop_Checkout.dc.html`
- `W_Shop_Done.dc.html`

## Scope
1. Run locally (local-dev), screenshot every listed board's screen in light AND dark next to the board render, fix small drift here (bigger → a `b` follow-up task), write `docs/qa/canvas/shop.md` (board · route · light · dark · verdict · fix). Journey: admin adds sellers/products → order from 2 sellers → ship/deliver → checklist updates → return.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- docs/qa/canvas/shop.md, no open ✘.
- `verify` green.
