---
id: L6-04
title: Cart (session) with progressive enhancement
milestone: L6
type: fullstack
status: done
depends_on: [L6-03]
parallel_group: L6-C
touches: [app/Domain/Shop/Cart,app/Http/Controllers/Shop/CartController.php,resources/views/pages/shop/cart.blade.php,resources/js/modules/cart.js,tests/Feature/Shop/CartTest.php,tests/Unit/Shop/CartTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design shop-cart.html --route /shop/cart
---

# L6-04 — Cart (session) with progressive enhancement

## Scope
- `Cart` aggregate (lines: product/variant id, qty, unit price snapshot) behind `CartRepository` (session impl);
  actions `AddToCart`, `UpdateCartLine`, `RemoveFromCart`, `ClearCart`; price/stock revalidated on every read.
- `/shop/cart` (noindex, never page-cached) matching `shop-cart.html` incl. «شاید لازم داشته باشی»; forms POST
  (PRG) and JSON responses for the JS module; `rt_cart_count` cookie for the header badge.
- Shipping rule (flat / free over X from settings) shown as estimate.

## Acceptance
- Add/update/remove with and without JS; stock overflow handled; diff < 3%; tests green.
