---
id: L1-03
title: SEO core: SeoMeta, SeoManager, head component, basic seo:audit
milestone: L1
type: backend
status: todo
depends_on: [L1-01]
parallel_group: L1-C
touches: [app/Domain/Seo,resources/views/components/seo,database/migrations,app/Console/Commands/SeoAudit.php,tests/Feature/Seo,tests/Unit/Seo]
skills: []
verify: composer verify
---

# L1-03 — SEO core: SeoMeta, SeoManager, head component, basic seo:audit

## Why
SEO is the project's #1 goal. One service must decide every head tag so no page can ship without them.

## Scope
- `seo_meta` morph table: title, description, canonical_url, robots (index/noindex, follow/nofollow, max-image-preview
  etc.), og_title, og_description, og_media_id, og_type, twitter_card, focus_keyword, schema_overrides JSON,
  sitemap_include, sitemap_priority, sitemap_changefreq. `HasSeo` trait for models; static pages keyed by route name.
- `SeoManager` (request-scoped, injectable): layered resolution defaults (settings) → page/model meta → controller
  overrides; title template; description fallback from excerpt (trimmed to ~155 chars at a word boundary, Persian
  aware); absolute canonical normalised (https, host, no tracking params, no trailing slash, page param kept only
  when > 1); `noindex` automatic on non-production, search, cart/checkout/done pages, filtered listings.
- `<x-seo.head/>`: `<title>`, description, canonical, robots, OG (`og:locale=fa_IR`, site_name, type, url, title,
  description, image + width/height/alt/type), Twitter card, `link rel=alternate` RSS (when blog exists),
  verification metas, `prev/next` NOT emitted (Google ignores), JSON-LD slot (L1-04).
- `php artisan seo:audit` v1: renders every registered route in-process and checks title (30–60 chars, unique),
  description (70–160, unique), exactly one `<h1>`, canonical present + absolute, OG complete, images have alt +
  width + height, no `href="#"`; exits non-zero on errors. Extended in L7-05.
- Tests: resolution order, canonical normalisation, robots on non-prod, audit command on fixtures.

## Out of scope
- Admin UI (L7-01), JSON-LD builders (L1-04).

## Acceptance
- Any view using the layout gets a complete head with zero per-view boilerplate; tests green.
