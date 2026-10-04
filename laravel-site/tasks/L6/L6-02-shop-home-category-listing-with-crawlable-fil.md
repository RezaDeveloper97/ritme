---
id: L6-02
title: Shop home + category listing with crawlable filters
milestone: L6
type: frontend
status: todo
depends_on: [L6-01,L3-01]
parallel_group: L6-B
touches: [resources/views/pages/shop/index.blade.php,resources/views/pages/shop/category.blade.php,resources/views/components/shop,app/Http/Controllers/Shop/ShopHomeController.php,app/Http/Controllers/Shop/CategoryController.php,lang/fa/shop.php,tests/Feature/Shop/ListingTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design shop.html --route /shop && node tools/shot.mjs --design shop-list.html --route /shop/category/baby-clothes
---

# L6-02 — Shop home + category listing with crawlable filters

## Scope
- `/shop` (all design sections; «لیست سیسمونی» and «یادآور خرید قبل از پریود» as app CTAs), `/shop/category/{slug}`
  (filters: price range, brand, size, in stock; sort: popular/newest/price) via GET; indexable: category + page;
  filtered → `noindex,follow` + canonical to category. Product cards with `<x-picture>` square variant.
- Schema: `CollectionPage`, `ItemList` of products, BreadcrumbList.
- Cart badge in header **not** baked into cached HTML (read from a cookie by a tiny module).

## Acceptance
- Diffs < 3% vs `shop.html` and `shop-list.html`; `seo:audit` passes; tests green.
