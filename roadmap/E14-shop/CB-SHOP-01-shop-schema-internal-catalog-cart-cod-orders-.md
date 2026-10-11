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
0. Build ON bloom B-N10-03 (migration 00071, `internal/shop`): existing `shop_categories` (layette, beauty seeded), `shop_products` (title JSON, image_path, price_text, badge, nullable `external_url`, partner_label, sort_order, is_active) and `shop_click_counts` are EXTENDED with `ALTER TABLE` (nullable new columns: seller_id, rating cache, attributes JSON, etc.) and new `shop_*` tables — never recreated/renamed. Migrations numbered 00080+ (+ Laravel twin). Rows with `external_url` set stay link-out (price_text, click counter, interstitial) until migrated; a product with variants is internal. The `ShopOrders` partner adapter stub and the Me «سفارش‌ها» partner row are dropped/unused (DECISIONS #15).
1. Extend `shop`: sellers (verified, shipping days, discreet packaging), category tree per shop (extend `shop_categories` with parent_id for sub-categories under the two seeded roots), products (badges, rating cache, material/wash/size guide or IRC code/ingredients/expiry/skin types, public images), variants (size/color/volume, price, compare-at, stock), verified-purchase reviews, carts, addresses, orders (per-seller shipments, delivery slot, status timeline, COD, totals, discreet flag), returns, layette templates (catalog) + user checklist states, supply-reminder opt-in.
2. Shop tables never join health data.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- schema-diff green; link-out products still render until migrated.
- `verify` green.
