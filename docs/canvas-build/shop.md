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
## Layette checklist + supply reminder API (CB-SHOP-04)

Package `internal/shop/checklist`, routes `internal/http/routes_shop_checklist.go` (group `shop_checklist`), contract
group `shop-checklist`. auth:api, teen-mode → 404.

| route | role |
|---|---|
| `GET /api/v1/shop/checklist[?for_user_id=]` | template (active) + her state + custom rows, grouped `clothing`/`sleep`/`feeding`/`bath`/`outing` (unknown template groups after); `ready` = `have`, `total` excludes `not_needed`; `due_date`/`weeks_left` from the active pregnancy. Family share: a companion with a live `pregnancy` grant reads it (`read_only: true`, audited) |
| `PUT /api/v1/shop/checklist/items/:id` | her state (+quantity) on a template item (template rows are never deleted, only `not_needed`) |
| `POST/PATCH/DELETE /api/v1/shop/checklist/custom[/:id]` | her own rows (≤ 200); a body naming another account is 403 (share is read only) |
| `GET/PUT /api/v1/shop/supply-reminder` | opt-in, **off by default**. On: the predicted next period start (cycle engine, cycle/TTC/no life mode only) → one care reminder in `reminders` (type `custom`, `meta.source = shop_supply`, discreet title «نگاهی به وسایل شخصی‌ات بینداز», `lead_days` before at 09:00 Tehran, not before today, none when the period is due/late). Off → the row is deleted and none is ever created. GET re-syncs it to the current prediction (rolls to the next cycle) |

Health inputs (due date, prediction) and the companion check are injected interfaces wired in the routes file; the
package imports no health package and logs no health data. `Service.SyncSupply` is exported for a future daily job.
The «in transit → have» moves on order / delivery belong to CB-SHOP-03.

## Cart, COD checkout, orders, returns (CB-SHOP-03)

Package `backend-go/internal/shop/order` (sqlc queries `db/queries/shop/order_api.sql`, migration
`00087_shop_checkout.sql` + Laravel twin), routes in `internal/http/routes_shop.go`. All `auth:api`, owner-scoped
(someone else's order / address → 404, someone else's `address_id` at checkout → 422), teen mode → 403
`teen_commercial_blocked`. Errors: `{success:false, message, error_code[, errors][, variant_ids]}`.

| route | does |
|---|---|
| `GET /shop/cart` | lines per seller (one shipment each) + fee + free-shipping remainder; unavailable lines apart; applied code re-evaluated |
| `POST /shop/cart/items` · `PUT/DELETE /shop/cart/items/:variant_id` | add (≤ stock, ≤ 10 per variant, ≤ 30 lines) / set / remove |
| `PUT/DELETE /shop/cart/discount` | admin codes `shop_discount_codes` (percent/fixed, cap, minimum, window, global + per-user limits) |
| `GET/POST /shop/addresses` · `PUT/DELETE /shop/addresses/:id` | her addresses (mobile + 10-digit postal code, digits folded; first = default; ≤ 20) |
| `GET /shop/checkout` | addresses, per-seller shipment (fee, COD ceiling, slots from tomorrow), plain packaging, `payment_methods: [cod]`, totals, `blockers` |
| `POST /shop/orders` | place (COD only), `idempotency_key` required: same key → 200 + `replayed: true` |
| `GET /shop/orders` · `GET /shop/orders/:id` | cards · detail with shipments, slot, items, timeline, `return_until`, `checklist_on_the_way` |
| `POST /shop/orders/:id/cancel` | while `placed` and every shipment `awaiting_seller`: stock, slots, code use and checklist given back |
| `POST /shop/orders/:id/returns` · `GET /shop/returns` | item of a delivered shipment within the seller's `return_days` (default 7) |

- **Checkout transaction**: lock the cart row (`FOR UPDATE`; serialises one user's checkouts so the idempotency key is
  race-free, plus `UNIQUE (user_id, idempotency_key)`), take stock with `UPDATE … WHERE stock >= qty` in variant-id
  order, then the code use (`used_count < max_uses`), then each chosen slot (`booked < capacity`). Any guard missing →
  the whole order rolls back (409 `out_of_stock` / `discount_exhausted` / `slot_full`). Deadlock → 409 `busy`.
- **Snapshots**: unit price, title, variant label (all languages present in the data), recipient and address text.
- **Statuses** written here: order `placed` → `cancelled` / `delivered`; shipment `awaiting_seller` → `cancelled` /
  `delivered`. CB-SHOP-05's admin moves shipments through confirm / ship and calls
  `order.Service.MarkShipmentDelivered` (shipment + timeline, order delivered when all are, checklist «دارم»).
- **Layette checklist link**: placing moves her existing checklist rows whose template category is an ordered product's
  category (or its parent) from todo / in_cart / later to `on_the_way`; delivery → `have`; cancel → `todo`.
- **SMS**: `order.Notifier` adapter, fake provider only (logs the order id). The copy is discreet: order code + «پرداخت
  هنگام تحویل», never a product name. No phone number or address reaches a log line.
- Open: encryption at rest for `shop_addresses` / order recipient snapshots; a real SMS provider for shop messages.

## Admin (CB-SHOP-05)

Package `backend-go/internal/shop/admin` (sqlc queries `db/queries/shop/admin.sql`), registered in
`routes_admin_shop.go` after bloom's link-out CRUD (one `/shop` section). admin-web: `/shop` (products, with a
«commerce» tab per product for seller, fields, gallery and variants), `/shop/sellers`, `/shop/stock`, `/shop/slots`,
`/shop/shipments` (+ `/shop/orders/:id`), `/shop/returns`, `/shop/reviews`, `/shop/discount-codes`, `/shop/layette`.
API reference: `docs/go-migration/admin-api.md` §25.

- **Capabilities**: catalog, reviews, discount codes and the layette template are `content`; the shipment queue, order
  detail and returns show the recipient's name, phone and address, so `httpadmin.capabilityOf` maps
  `/shop/shipments|orders|returns` to `support`. Audit lines carry ids and statuses only.
- **Shipment machine**: `awaiting_seller → confirmed → shipped → delivered` (confirmed may go straight to delivered).
  Delivered calls `order.Service.MarkShipmentDelivered`; other moves write an order event with the shipment id.
- **Admin cancel**: `order.Service.CancelByAdmin` — placed and no shipment shipped/delivered (the user's own cancel
  still needs every shipment awaiting the seller).
- **Returns**: `requested → approved | rejected`; approve may restock the variant. Cash is settled offline (COD).
- **Reviews**: moderation and delete refresh `rating_avg` / `rating_count`.
- **Slots**: generated from a weekly template (weekdays + windows) over ≤ 62 days; existing rows are kept
  (`INSERT IGNORE` on the seller/day/start key); booked slots cannot be deleted or shrunk below the bookings.

