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

## Catalog API (CB-SHOP-02)

Package `backend-go/internal/shop/catalog` (sqlc queries `db/queries/shop/catalog_api.sql`), routes in
`internal/http/routes_shop.go` next to bloom's. All `auth:api`; a teen-mode account gets **403
`teen_commercial_blocked`** (bloom's link-out routes keep their 404 / empty list).

| route | answers |
|---|---|
| `GET /api/v1/shop/catalog/home?root=layette\|beauty` | root tabs, sub-categories + counts, ≤ 6 sections × 4 best sellers |
| `GET /api/v1/shop/catalog/products?category&q&size&color&volume&material&skin_type&price_min&price_max&sort&page&per_page` | filtered, sorted, paged cards + `filters` (choices with counts, price range) |
| `GET /api/v1/shop/catalog/products/:id` | detail: variants + stock, seller terms, attributes, beauty fields, 3 newest reviews, bought-together |
| `GET /api/v1/shop/catalog/products/:id/reviews` | published reviews, newest first, paged |

- **Catalog product** = active, active category, ≥ 1 active variant, seller null or active. Link-out rows are not
  catalog products (404 on the detail); bloom's `/shop/products*` is unchanged. bloom's public `/shop/categories` now
  lists **root** categories only, so sub-categories never leak into its two-tab strip.
- A category's list covers its whole subtree. Card price = cheapest in-stock variant (else cheapest).
- **Filter values are the default-language text** of the variant `size` / `color` / `volume` and of
  `attributes.material` (labels are localized), so a chosen filter survives a UI language switch. Size, color, volume
  and the price bounds must hold on the same variant. `skin_type` matches keys in `skin_types` (the client labels them).
- Data shapes the admin (CB-SHOP-05) writes: `attributes` = `{"material": {…}, "wash": {…}, "size_guide": {…},
  "ingredients": {…}, …}` (translatable values or plain strings; the board rows come first, the rest alphabetically);
  `skin_types` = `["oily","combination",…]`; `badges` = array of translatable objects or strings; `images` = array of
  storage paths (`image_path` stays the cover).
- Sort: `popular` (default, purchase_count), `rating`, `newest`, `price_asc`, `price_desc`. Filtering runs in memory
  over at most 1000 products per subtree (MVP catalog size).
- Bought-together = products sharing orders (`shop_order_items`, no user column read), topped up with the category's
  best sellers.
