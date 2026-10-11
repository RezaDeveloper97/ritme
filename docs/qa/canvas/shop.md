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
