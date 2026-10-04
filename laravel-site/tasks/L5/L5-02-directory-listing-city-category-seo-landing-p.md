---
id: L5-02
title: Directory listing + city/category SEO landing pages
milestone: L5
type: frontend
status: todo
depends_on: [L5-01,L3-01]
parallel_group: L5-B
touches: [resources/views/pages/directory/index.blade.php,resources/views/pages/directory/partials,app/Http/Controllers/Directory/ListPlacesController.php,lang/fa/directory.php,tests/Feature/Directory/ListingTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design directory.html --route /directory
---

# L5-02 — Directory listing + city/category SEO landing pages

## Scope
- `/directory`, `/directory/{city}`, `/directory/{city}/{category}` (indexable, unique title/description/h1 templates
  editable in admin, intro text per combo); other filter combos via query string → `noindex,follow` + canonical to
  the nearest indexable URL. Filters are plain GET forms (crawlable links for city/category, JS only enhances).
- Place cards with `<x-picture>` cover, rating, age range, open-now pill; pagination self-canonical.
- `ItemList` + `CollectionPage` + BreadcrumbList schema; «مجموعه‌ای برای مادر و کودک داری؟» CTA → business page.
- Audit correction (`docs/AUDIT.md` §4.2): the design's right-hand map column (560×1180 SVG with price pins, zoom ±,
  «با جابه‌جایی نقشه جست‌وجو کن») conflicts with the no-maps decision. Replace it with a static local illustration
  plus crawlable city/district links, or drop the column; results go full width on mobile. No map JS.

## Acceptance
- Diff < 3% vs `directory.html`; `seo:audit` passes on 3 URL shapes; tests green.
