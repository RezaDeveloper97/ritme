# Shop — design fidelity (canvas)

## CB-SHOP-06

Screens render the current light/dark tokens, not the board hex. Shots at 390 wide, Persian, API mocked at the
network layer (catalog data shaped as `GET /shop/catalog/home`).

| board | route | light | dark | verdict | fix |
|---|---|---|---|---|---|
| `nbd_Shop_Home` (`docs/qa/canvas/boards/nbd_Shop_Home.png`) | `/shop` (baby tab) | `CB-SHOP-06/fa_shop-baby.light.png` | `CB-SHOP-06/fa_shop-baby.dark.png` | matches: root tabs, category chips, best sellers per sub-category, trust row, separation note | layette checklist card, curated set, city/cart header and search deferred to CB-SHOP-09 / 08 / 07 (APIs not yet built) |
| `nbd_Shop_Beauty` (`docs/qa/canvas/boards/nbd_Shop_Beauty.png`) | `/shop` (beauty tab) | `CB-SHOP-06/fa_shop-beauty.light.png` | `CB-SHOP-06/fa_shop-beauty.dark.png` | matches: IRC trust card, sections, trust row | period-supply reminder card deferred to CB-SHOP-04 API / CB-SHOP-09 |

Behaviour: empty internal catalog (404) falls back to the bloom link-out categories page; teen account (403) shows an
unavailable state; no online-payment wording (pay on delivery note).

## CB-SHOP-07

Shots at 390 wide, Persian, API mocked at the network layer (shapes of `GET /shop/catalog/products[/:id]`). The PWA
install prompt visible on the list shot is the app shell, not this screen.

| board | route | light | dark | verdict | fix |
|---|---|---|---|---|---|
| `nbd_Shop_List` (`docs/qa/canvas/boards/nbd_Shop_List.png`) | `/shop/[category]` (catalog category) | `CB-SHOP-07/fa_list.light.png` | `CB-SHOP-07/fa_list.dark.png` | matches: search, filters button + removable active chips (filter sheet), sub-category tabs, result count, sort, 2-col grid, pager, size note | product images need real uploads (placeholder icon when none) |
| `nbd_Shop_Product` (`docs/qa/canvas/boards/nbd_Shop_Product.png`) | `/shop/product/[id]` (baby) | `CB-SHOP-07/fa_product-baby.light.png` | `CB-SHOP-07/fa_product-baby.dark.png` | matches: gallery, seller, rating, color/size choosers with guide, material/wash/shipping/return, reviews, bought-together, sticky bar | add-to-cart is disabled with a hint until the cart (CB-SHOP-03/08) lands |
| `nbd_Shop_ProductBeauty` (`docs/qa/canvas/boards/nbd_Shop_ProductBeauty.png`) | `/shop/product/[id]` (beauty) | `CB-SHOP-07/fa_product-beauty.light.png` | `CB-SHOP-07/fa_product-beauty.dark.png` | matches: volume, skin types, IRC + expiry, ingredients, pregnancy/breastfeeding caution | none |

Behaviour: non-catalog slug (404) keeps bloom's link-out list; teen (403) shows the unavailable state; pay on delivery
only, no online-payment wording.
## CB-SHOP-09

Shots at 390 wide, Persian, API mocked at the network layer (shapes from `GET /shop/checklist`, `GET/PUT /shop/supply-reminder`).

| board | route | light | dark | verdict | fix |
|---|---|---|---|---|---|
| `nbd_Shop_Checklist` (`docs/qa/canvas/boards/nbd_Shop_Checklist.png`) | `/shop/checklist` | `CB-SHOP-09/fa_shop-checklist.light.png` | `CB-SHOP-09/fa_shop-checklist.dark.png` | matches: progress ring, due date + weeks left, family share, add item, filter chips, per-group rows with state lines and «see» links, safe-sleep note on the row, "general list, more stuff is not more ready" note | state/qty edit via sheet (pen icon); family view (`?for=`) is read only |
| `nbd_Shop_Home` | `/shop` (baby) | `CB-SHOP-09/fa_shop-baby-card.light.png` | `CB-SHOP-09/fa_shop-baby-card.dark.png` | layette-checklist card with progress | none |
| `nbd_Shop_Beauty` | `/shop` (beauty) | `CB-SHOP-09/fa_shop-beauty-reminder.light.png` | `CB-SHOP-09/fa_shop-beauty-reminder.dark.png` | discreet supply-reminder toggle (copy never names the reason; opt-in, off by default) | none |

## CB-SHOP-08

Shots at 390 wide, Persian, API mocked at the network layer (shapes of `GET /shop/cart`, `/shop/checkout`, `/shop/orders[/:id]`).
The PWA install prompt on the cart shot is the app shell, not this screen. Pay on delivery only: no bank-gateway row
(DECISIONS #13); the board's «درگاه بانکی» is intentionally left out.

| board | route | light | dark | verdict | fix |
|---|---|---|---|---|---|
| `nbd_Shop_Cart` | `/shop/cart` | `CB-SHOP-08/fa_cart.light.png` | `CB-SHOP-08/fa_cart.dark.png` | matches: per-seller groups with ship days, free-shipping hint, qty stepper, discount code, totals («محاسبه در مرحله بعد» for shipping), sticky total + continue | product images are placeholders until uploaded |
| `nbd_Shop_Checkout` | `/shop/checkout` | `CB-SHOP-08/fa_checkout.light.png` | `CB-SHOP-08/fa_checkout.dark.png` | matches: stepper, address (sheet to pick/add/edit), masked recipient phone, slot chips per seller, plain packaging switch, COD card, totals, place order | gateway option removed (COD only); one idempotency key per attempt, reused on retry |
| `nbd_Shop_Order` | `/shop/order/[id]?placed=1` | `CB-SHOP-08/fa_order-placed.light.png` | `CB-SHOP-08/fa_order-placed.dark.png` | matches: code + discreet SMS note, per-seller timeline (confirm / ship / deliver), checklist «on the way» link, returns after delivery | cancel (sheet) while nothing is confirmed |
| Me → orders | `/profile/orders` | `CB-SHOP-08/fa_my-orders.light.png` | `CB-SHOP-08/fa_my-orders.dark.png` | order cards (code, sellers, count, date, total) | none |

## CB-SHOP-10

Shots at 1440 wide, Persian, guest (logged out), light + dark. The catalog API is mocked by a local stand-in server (shapes of `GET /shop/catalog/home`, `/products`, `/products/:id`), so SSR also runs. Pay on delivery only, no online-payment wording. Board renders: `docs/qa/canvas/boards/W_Shop_{Home,List,Product}.png`.

| board | route | light | dark | verdict | fix |
|---|---|---|---|---|---|
| `W_Shop_Home` | `/store` | `CB-SHOP-10/home-light.png` | `CB-SHOP-10/home-dark.png` | matches: hero + the two root cards (baby, beauty), category strip, best sellers per root, trust strip | the board's wishlist hearts, search/cart icons, magazine nav and the layette-checklist / reminder cards are app features and not on the public page; category art is the icon until images are uploaded |
| `W_Shop_List` | `/store/[category]` | `CB-SHOP-10/list-light.png` | `CB-SHOP-10/list-dark.png` | matches: breadcrumb, title + count, filter sidebar (sub-categories, size, material, color, price), active chips, sort pills, 4-col grid, numbered pager | color swatches are checkboxes (labels come from the API, no hex); no seller / in-stock facet (not in the API) |
| `W_Shop_Product` | `/store/product/[id]` | `CB-SHOP-10/product-light.png` | `CB-SHOP-10/product-dark.png` | matches: breadcrumb, gallery, seller, rating, price, color/size choosers, add-to-cart panel (login link when logged out), shipping/returns, specifications, reviews, bought-together | no quantity stepper or wishlist (cart takes qty 1); size table shown as the size guide line |

Behaviour: logged out → «ورود یا ثبت‌نام» link; logged in → `useAddToCart`, then a link to the app cart. Signed-in teen (403) and unknown product (404) get a plain state. `<1024px` signed-in visitors are redirected to the app `/shop` screens by the web shell. SSR metadata (title, description, OpenGraph, canonical) comes from the public catalog API.

## CB-SHOP-11

Desktop 1440w, mocked API (3 items from 2 sellers). Routes sit under the public `store` segment, signed-in only (page-level login prompt `CB-SHOP-11/gate-dark.png`), noindex + robots disallow.

| board | route | light | dark | verdict | fix |
|---|---|---|---|---|---|
| `W_Shop_Cart` | `/store/cart` | `CB-SHOP-11/cart-light.png` | `CB-SHOP-11/cart-dark.png` | matches: title, lines per seller with free-shipping bar, quantity stepper + remove, discount code, sticky order summary + «ادامه خرید», privacy note | no «شاید لازم داشته باشی» strip (no recommendations endpoint); no cart/search icons in the header (cart is a nav link) |
| `W_Shop_Checkout` | `/store/checkout` | `CB-SHOP-11/checkout-light.png` | `CB-SHOP-11/checkout-dark.png` | matches: 3-step bar, numbered address / delivery-time / payment cards, per-seller slot grid, plain-packaging switch, summary with pay-on-delivery button | payment is ONE fixed «پرداخت در محل» card — the board's bank-gateway option is out of MVP scope; address map thumbnail omitted; phone masked; add/edit address inline |
| `W_Shop_Done` | `/store/order/[id]` | `CB-SHOP-11/done-light.png` | `CB-SHOP-11/done-dark.png` | matches: success mark, title, order code, one card per seller with items, status chip and slot, «پیگیری سفارش» + «ادامه خرید», return note | per-seller timeline reduced to status chip + slot (full timeline lives on the app order page, linked); app-download strip not rendered (no real store links yet, same as the shell footer) |

Behaviour: `<1024px` signed-in visitors go to the app `/shop/cart|checkout|order/[id]`. One `crypto.randomUUID()` idempotency key per checkout page, reused on retry; discount refusals show the generic «not valid» text, 429 shows a try-later line; `too_many_open_orders` (422) is shown from `shop.errors`.

## CB-SHOP-12

Real end-to-end pass (no mocks): Go API (`:8731`) against a private DB `shop12_dev` (goose 87), frontend dev server and
admin-web dev server on their own ports, own headless Chrome (Playwright-core, `bypassCSP` because the app CSP allows
`img-src https:` only and the local API is http). Seed: admin API (2 sellers, root + sub-categories, 4 products with
7 images and 5 variants, 21 delivery slots, 2 discount codes, 3 layette template rows) plus one seller created through the
admin-web form. Users: Alice, Bob, Cara (adult), Teen (life_mode teen). Shots in `CB-SHOP-12/`
(`app-*` = 390 wide, `web-*` = 1440, `admin-*` = 1440; `-fa`/`-en`, `-light`/`-dark`).

### Journey

| step | how | result | evidence |
|---|---|---|---|
| admin creates seller, sub-category, product + variants + images + stock, slots, discount code, layette template | admin API; seller also via admin-web form | ok (variant `is_active` defaults to false when omitted, see F-7) | `admin-sellers-light`, `admin-seller-created-light`, `admin-commerce-p1-light`, `admin-stock/slots/discounts/layette-light` |
| browse app (fa/en) | `/shop`, beauty tab, `/shop/[category]` | ok | `app-home-fa-*`, `app-home-beauty-fa-*`, `app-home-en-*`, `app-list-fa-*` |
| browse desktop logged out | `/store`, category, product, cart, checkout | ok; cart/checkout show a login card, product shows a login CTA instead of add-to-cart | `web-home-guest-*`, `web-list-guest-*`, `web-product-guest-*`, `web-cart-guest-*` |
| filters | color "pink" sheet, active chip, header count | ok after F-1 and F-2 | `app-filters-fa-light`, `app-list-filtered-fa-light` |
| product, add to cart (2 sellers) | app + desktop | ok; cart thumbnails fixed (F-3) | `app-product-fa-*`, `app-cart-fa-*`, `web-cart-fa-*` |
| discount: valid / invalid / limit | `WELCOME10`, `NOPE`, 10 attempts per hour | valid -50k; invalid is the generic 422 `discount_invalid` (same copy for unknown, expired, fixed-limit); 11th attempt 429 `throttled` (a valid code is also refused while throttled); another user is not throttled | `app-cart-discount-invalid-fa-light`, `app-cart-discount-ok-fa-light` |
| checkout: address, slot per seller, discreet packaging, COD | app (390) and desktop (1440) | ok; order placed once; shipping per seller (30k + free over 400k) | `app-checkout-*`, `app-checkout-bottom-fa-light`, `web-checkout-fa-*` |
| done | `/shop/order/[id]?placed=1`, `/store/order/[id]` | ok | `app-order-placed-fa-light`, `web-order-done-fa-*` |
| admin moves shipments to delivered | `awaiting_seller -> confirmed -> shipped -> delivered`, skip refused 422 `invalid_transition` | ok | `admin-shipments-light`, `admin-order-1-light` |
| layette rows on_the_way -> have | Bob had the row touched before ordering | `on_the_way` on place, `have` on delivery (see F-6: rows never touched stay `todo`) | `app-checklist-*` |
| return request, admin approves with restock | Bob, reason `wrong_size` | requested; approve restock stock 8 -> 9; second approve 422 `not_pending`; second request over quantity 422 | `admin-returns-light` |
| review, admin moderates, rating shows | no user endpoint exists (F-5), review row seeded by SQL; admin `moderate published` | catalog `rating_avg 4`, `rating_count 1`, review listed with `verified_purchase` | `admin-reviews-light`, `app-product-en-*` |

### Matrix

| check | result |
|---|---|
| cancel restores stock, slot booking, discount use | pass (stock 3 -> 2 -> 3, slot booked 0 -> 1 -> 0, code `used_count` 1 -> 2 -> 1); second cancel 422 `not_cancellable` |
| oversell, 2 users race for the last unit | pass: one 201, the other 409 `out_of_stock`, stock 0, one order item |
| idempotent double submit | pass: API same key twice answers 201 then 200 with the same order (1 row); UI double click sent 1 to 2 POSTs, always exactly 1 order, stock moved once |
| open-order cap | pass: 5 open orders accepted, 6th and 7th 422 (cancelled orders not counted) |
| teen account | catalog, cart, checkout, orders, addresses 403 with the unavailable copy; checklist and supply reminder 404 (by design); both app screens render the unavailable state (`app-teen-shop-fa-light`) |
| fa + en, RTL + LTR | pass: `dir`/`lang` rtl/fa and ltr/en on every screen; one bug F-4 |
| light + dark | pass on every screen listed (`lint:dark` green) |
| 390 and 1440 | pass; F-1 was the only overflow |
| PII in logs | API log (426 lines), frontend and admin logs: 0 hits for phones, names, addresses; OTP SMS logged masked; browser consoles had no errors apart from the expected 422 of the invalid discount code |
| online-payment wording | no payment gateway, card or "pay now" copy anywhere. Two strings name it only to deny it: `shop.checkout.codBody` ("There is no online payment") fa + en. Kept (board wording), listed as F-8 |
| supply reminder discreet copy | reminder title "Check your personal supplies" / "نگاهی به وسایل شخصی‌ات بینداز", no period or product word (also asserted by `checklist_int_test`); card copy says nothing reaches the seller |

### Security

| test | result |
|---|---|
| guest: catalog home/product/reviews | 200 (by design); cart, orders, checklist 401 |
| IDOR Bob on Alice: `GET /orders/1`, cancel, return | 404 x3 |
| return on own order with Alice's item id | 422 (item not in order) |
| `PUT`/`DELETE /addresses/1` of Alice | 404 / 404; Bob's address list holds only his own |
| order with Alice's `address_id` | 422 |
| returns list | each user only their own |
| checklist: Bob `PATCH`/`DELETE` Alice's custom item | 404 / 404; checklist state rows are per user |
| idempotency key of another user's order | not replayed (processed as a new request) |
| validation: `payment_method=card`, quantity 0 and 9999, bad phone | 422 |
| SQL-ish discount code | generic 422 |
| discount brute force | 429 after 10 attempts per hour per user |
| open-order cap, oversell | see matrix |
| admin writes | audit rows carry ids only; order data (name, phone, address) only under the `support` capability |

### Findings

Fixed in this commit:

| id | severity | finding | fix |
|---|---|---|---|
| F-1 | medium | filter sheet: the "to" price input overflowed the viewport (grid `1fr` + input min width) | `.shp-price-range` `minmax(0,1fr)`, input `inline-size:100%; min-inline-size:0` |
| F-2 | medium | list header always said "0 items": category schema defaulted the absent `product_count` to 0 so the `?? meta.total` fallback never ran (app + desktop) | schema field optional, desktop sub-category count `?? 0` |
| F-3 | medium | cart, checkout and order lines had no thumbnail for admin-uploaded products: queries read only the legacy `image_path`, not the `images` array | `ListCartLines` / `ListProductImages` also select `images`; `coverPath` helper (+ unit test) |
| F-4 | low | en checklist ring showed the Persian percent sign `50٪` | `%` unless locale is fa |
| F-9 | low | en copy "More things does not mean more ready" | grammar fixed in frontend, backend copy and golden |

Open:

| id | severity | finding | suggestion |
|---|---|---|---|
| F-5 | medium | no user endpoint to write a review: only moderation and display exist, so the review -> moderate -> rating journey cannot start from the app | add `POST /shop/orders/:id/reviews` (delivered items only, one per order item) in a follow-up; the table (`order_item_id` unique, `status pending`) is ready |
| F-6 | low | checklist link only moves rows the user already has; a user who never touched the layette list gets no `on_the_way` / `have` after buying (documented in the query comment) | create the row on purchase, or show "bought" from orders |
| F-7 | low | admin variant created through the API without `is_active` is inactive (form checkbox convention); easy to ship an invisible variant | default true on create, or required |
| F-8 | low | two checkout strings mention "online payment" while denying it | reword to "pay on delivery" only if the copy owner wants zero mentions |
| F-10 | low | Lighthouse SEO on `/fa/store/product/1` (production build): 0.92, only failure "no meta description": Next 15.5 streams metadata into `<body>` for browser UAs (title, description, canonical, OG are in the HTML, outside `<head>`). Googlebot renders JS; social crawlers get blocking metadata | optional `htmlLimitedBots: /.*/` in `next.config.ts` (site-wide TTFB cost), not applied |
| F-11 | low | admin-web dashboard logs `MISSING_MESSAGE overview.mode.teen` for fa (teen user in the data) | add the key (not a shop screen) |
| F-12 | info | tag line "Shop · enter" / "خرید · ورود" in the shop header reads like a login prompt | copy owner |

Environment notes: the app CSP (`img-src https:`) blocks images of an http API, so local screenshots need `bypassCSP`;
Lighthouse numbers come from `next start` of a production build with `NEXT_PUBLIC_API_BASE_URL` pointing at the local API.

### Follow-up fixes (QA follow-ups commit)

| id | resolution |
|---|---|
| F-5 | `POST /api/v1/shop/orders/:id/items/:item_id/review` {rating 1–5, body?}: delivered items only (422 `not_reviewable`), one per order item (409 `already_reviewed`), `pending` until moderation, author = initial of the profile name; owner-scoped 404, teen 403, shop write throttle. Order detail items carry `reviewable` + `review_status`; `/shop/order/[id]` shows «ثبت نظر» and a sheet (stars + optional text, "shows after a check"). OpenAPI, contract case `shop/review_refusals`, `shop_review_int_test.go` |
| F-6 | placing an order also inserts the active layette rows of the ordered categories she never touched as `on_the_way` (`InsertChecklistOnTheWay`); delivery moves them to `have`, cancel reverts them to `todo` under the existing other-open-orders rule |
| F-7 | admin variant create: `is_active` omitted = active (explicit false kept); update unchanged |
| F-11 | admin-web `overview.mode.teen` added (fa + en) |
