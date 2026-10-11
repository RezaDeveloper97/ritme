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
