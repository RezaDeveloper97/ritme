# Shop / Catalog (L6-01)

Products (+ simple size/colour variants, slug history, gallery, cross-sells), categories (tree), brands and moderated
reviews. Single seller (Ritme): product cards show the brand; no seller model (a `seller_id` can be added later).

- Money: `App\Support\Money\Money` — **integer rials** in the database (`MoneyCast`), tomans on screen, IRR in schema.
- Reads: `ProductRepository`, `CatalogRepository` → Eloquent + Cached decorators (`shop` namespace).
- Queries: `ProductsInCategory` (filters, sorts, facets), `BestSellers`, `FrequentlyBoughtWith`, `SitemapProducts`,
  `SitemapCategories`.
- Observers bump `shop`, `sitemap` (+ `pages`). Pivot writes go through the `Sync*` actions; stock changes through
  `AdjustStock`.
- Sitemaps `shop-products`, `shop-categories`; site search provider `products`. Demo rows (`is_demo`) never count
  towards ratings and never reach a sitemap.
