---
id: L6-03
title: Product page: gallery, variants, specs, size chart, reviews, Product schema
milestone: L6
type: frontend
status: done
depends_on: [L6-02]
parallel_group: L6-B
touches: [resources/views/pages/shop/product.blade.php,app/Http/Controllers/Shop/ProductController.php,resources/js/modules/product.js,tests/Feature/Shop/ProductTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design shop-product.html --route /shop/product/body-long-sleeve-3
---

# L6-03 — Product page: gallery, variants, specs, size chart, reviews, Product schema

## Scope
- `/shop/product/{slug}`: gallery (LCP priority on first), variant picker (no-JS: radio inputs inside the add-to-cart
  form), price/compare price, stock state, add-to-cart form (works without JS; JS enhances with fetch + toast),
  specs table, size chart, reviews + form (moderated), «معمولاً با این می‌خرند».
- Schema: `Product` (name, image[], sku, brand, description, offers `Offer`/`AggregateOffer` with price, priceCurrency
  `IRR`, availability, url, priceValidUntil), `aggregateRating`/`review` from real approved reviews only; OG
  `product:price:amount/currency`.

## Acceptance
- Diff < 3% vs `shop-product.html`; schema valid; tests green.
