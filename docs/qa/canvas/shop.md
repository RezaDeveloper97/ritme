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
