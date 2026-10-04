---
id: L6-01
title: Shop catalog context: products, categories, brands, reviews, Money
milestone: L6
type: backend
status: todo
depends_on: [L2-01,L1-03,L4-04]
parallel_group: L6-A
touches: [app/Domain/Shop/Catalog,app/Support/Money,database/migrations,database/factories,database/seeders/ShopSeeder.php,tests/Feature/Shop/Catalog,tests/Unit/Shop]
skills: []
verify: composer verify
---

# L6-01 — Shop catalog context: products, categories, brands, reviews, Money

## Scope
- `Product` (title, slug + history, sku, brand, short/long description (sanitised), specs key-value, size chart
  (table JSON), price, compare_at_price, currency (Toman display, Rial stored as integer), stock qty, stock_status,
  weight, gallery media + cover, categories M2M with primary category, life_stage tags, rating cache, is_published,
  is_featured, sort), optional simple variants (size/color → own sku/price/stock) — only if the design needs them
  (the product page has sizes: yes → `ProductVariant`).
- `ProductCategory` (parent/children, cover media, intro text, SEO), `Brand`, `ProductReview` (moderated,
  verified-purchase flag later).
- `Money` value object (integer Rial, `toToman()`, Persian formatting `۱۲۹٬۰۰۰ تومان`), never floats.
- Cached repos (`shop` ns), query objects (`ProductsInCategory` with filters/sort, `BestSellers`,
  `FrequentlyBoughtWith`), observers bump `shop`, `pages`, `sitemap`; sitemap providers; search provider.
- Demo seeder from design (بادی آستین‌بلند نخی ۳ عدد …).
- Audit finding (`docs/AUDIT.md` §8): the design is **multi-seller**. Product cards show a seller («پوشاک
  پنبه‌ریز», «بهداشتی بانو», «خانه سیسمونی ماه‌نو») and «فروشنده بررسی‌شده», and cart/checkout/done group lines per
  seller with their own delivery estimates and status. Confirm with the user, then add a minimal `Seller` (name, slug,
  is_verified, shipping_days_min/max; products belong to one seller), or simplify to single-seller and drop the
  per-seller grouping in L6-04/05.

## Acceptance
- Factories/tests for Money, filters, variants stock; `composer verify` green.
