---
id: L1-06
title: Sitemaps index + robots.txt (cached)
milestone: L1
type: backend
status: todo
depends_on: [L1-05]
parallel_group: L1-D
touches: [app/Domain/Seo/Sitemap,app/Http/Controllers/Seo,routes/web.php,tests/Feature/Seo/SitemapTest.php]
skills: []
verify: composer verify
---

# L1-06 — Sitemaps index + robots.txt (cached)

## Why
Crawlers need a complete, fresh, fast sitemap; robots must never let staging get indexed.

## Scope
- `/sitemap.xml` (sitemapindex) → `/sitemaps/pages.xml` now; providers interface `SitemapProvider` so blog,
  products, places, categories register their own later (`/sitemaps/{provider}-{page}.xml`, 5,000 URLs per file).
- Entries: loc, lastmod (real `updated_at`), image:image for primary media; only indexable URLs (respects
  `seo_meta.robots`, `sitemap_include`).
- Rendered via cache-aside (`sitemap` ns, bumped by content observers); proper `Content-Type`, gzip handled by server.
- `/robots.txt` dynamic: production → configured rules (admin-editable in L7-04) + `Sitemap:` line; non-production
  → `Disallow: /`. Delete Laravel's static `public/robots.txt`.
- Tests: XML validity, noindex exclusion, non-prod robots.

## Out of scope
- IndexNow (L7-04).

## Acceptance
- Both endpoints valid and cached (second request runs zero queries); tests green.
