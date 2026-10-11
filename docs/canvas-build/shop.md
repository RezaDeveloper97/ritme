# Shop — internal catalog, cart, COD orders (CB-SHOP-01)

Schema: `backend-go/db/migrations/00080_shop_commerce.sql` (Laravel twin `2026_10_11_000080_create_shop_commerce_tables.php`),
built on bloom's `00071_shop_catalog.sql` (B-N10-03). Typed store: `backend-go/internal/shop/store` (queries in
`db/queries/shop/`). bloom's link-out code in `internal/shop` keeps its own SQL.

## Rules

- DECISIONS #13/#15: internal shop, **pay on delivery only** (`shop_orders.payment_method = 'cod'`); no gateway.
- Link-out vs internal: a product with active rows in `shop_product_variants` is internal (cart, stock, COD). A product
  with `external_url` and no variants stays link-out (price_text, click counter, interstitial) until an admin migrates it.
- Money is integer toman (`bigint unsigned`). Translatable columns are JSON keyed by language code.
- No shop table references a health table and no query joins one. The period-supply reminder reads cycle predictions
  through a Go interface (CB-SHOP-04), never SQL.
- Orders keep a **snapshot** of recipient/address and item title/variant label; user-owned rows cascade on account deletion.

## Tables

| table | role |
|---|---|
| `shop_sellers` | verified flag, ship min/max days, fee, free-shipping threshold, COD ceiling, discreet packaging, return days |
| `shop_categories` +`parent_id` | sub-categories under the seeded `layette` / `beauty` roots |
| `shop_products` +columns | `seller_id`, `subtitle`, `description`, `images`, `attributes` (material, wash, size guide, ingredients…), `irc_code`, `expires_on`, `skin_types`, `badges`, `discreet_packaging`, `rating_avg`/`rating_count` cache, `purchase_count` |
| `shop_product_variants` | size / color / volume, price, compare-at price, stock |
| `shop_carts`, `shop_cart_items` | one cart per user, one line per variant |
| `shop_addresses` | user delivery addresses |
| `shop_delivery_slots` | per-seller dated windows, optional capacity (`booked < capacity`) |
| `shop_orders` | code, address snapshot, `payment_method`/`payment_status`, `status`, `discreet`, totals |
| `shop_order_shipments` | one per seller per order: slot snapshot, status, fee, tracking, delivered_at |
| `shop_order_items` | snapshot lines |
| `shop_order_events` | status timeline (order or shipment) |
| `shop_returns` | per item; window enforced by the service from `shop_sellers.return_days` |
| `shop_product_reviews` | verified purchase (unique `order_item_id`), moderated `status` |
| `shop_layette_items` | admin checklist template: group, essential / first-days flags, «see» category |
| `shop_user_checklist_items` | user state per template item (`todo`, `have`, `in_cart`, `on_the_way`, `not_needed`, `later`) or a custom row |
| `shop_supply_reminders` | period-supply reminder opt-in (lead days, optional product) |

Status strings (`placed`, `awaiting_seller`, `requested`, `pending`…) are plain varchar; their sets are owned by the
CB-SHOP-03/05 services.
