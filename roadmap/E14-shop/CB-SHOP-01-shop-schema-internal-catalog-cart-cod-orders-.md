---
id: CB-SHOP-01
title: Shop schema: internal catalog, cart, COD orders, checklist (supersedes link-out)
epic: SHOP
type: backend
status: todo
depends_on: [B-N10-03, CB-CORE-05, CB-CORE-03]
parallel_group: SHOP-A
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/shop, backend-go/sqlc.yaml, docs/canvas-build/shop.md]
skills: [new-endpoint]
boards: [nbd_Shop_Product.dc.html, nbd_Shop_ProductBeauty.dc.html, nbd_Shop_Cart.dc.html, nbd_Shop_Checkout.dc.html, nbd_Shop_Order.dc.html, nbd_Shop_Checklist.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && make schema-diff
---

# CB-SHOP-01 — Shop schema: internal catalog, cart, COD orders, checklist (supersedes link-out)

## Why
DECISIONS: internal shop with pay-on-delivery replaces bloom's partner link-out purchase path (B-N10-03 keeps the entry + categories).

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbd_Shop_Product.dc.html`
- `nbd_Shop_ProductBeauty.dc.html`
- `nbd_Shop_Cart.dc.html`
- `nbd_Shop_Checkout.dc.html`
- `nbd_Shop_Order.dc.html`
- `nbd_Shop_Checklist.dc.html`

## Scope
1. Extend `shop`: sellers (verified, shipping days, discreet packaging), category tree per shop (baby, beauty), products (badges, rating cache, material/wash/size guide or IRC code/ingredients/expiry/skin types, public images), variants (size/color/volume, price, compare-at, stock), verified-purchase reviews, carts, addresses, orders (per-seller shipments, delivery slot, status timeline, COD, totals, discreet flag), returns, layette templates (catalog) + user checklist states, supply-reminder opt-in.
2. Shop tables never join health data.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- schema-diff green; link-out products still render until migrated.
- `verify` green.
