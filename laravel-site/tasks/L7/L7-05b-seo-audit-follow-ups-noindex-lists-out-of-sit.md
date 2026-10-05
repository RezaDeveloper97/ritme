---
id: L7-05b
title: SEO audit follow-ups: noindex lists out of sitemap, reviewer profile placeholder
milestone: L7
type: backend
status: todo
depends_on: [L7-05]
parallel_group: L7-D
touches: [app/Domain/Seo/Sitemap/PagesSitemapProvider.php,app/Http/Controllers/Blog/BlogListingController.php,tests/Feature/Seo/AuditTest.php,tests/Feature/Seo/SitemapTest.php,tests/Feature/Blog/ListingTest.php]
skills: []
verify: composer verify && php artisan seo:audit
---

# L7-05b — SEO audit follow-ups: noindex lists out of sitemap, reviewer profile placeholder

## Why
L7-05's audit is red on the dev site with 3 real errors: `/directory` and `/shop` (and `/blog` on a clean seed) are
`noindex` while they only contain demo/no items, yet `PagesSitemapProvider` lists them (`sitemap.noindex`); and
`/blog/author/medical-reviewer` is an indexable placeholder profile with a 26-char description. The audit test excludes
those three listings with a `sitemap_include=false` workaround.

## Scope
- `PagesSitemapProvider` skips static listing pages whose controller would answer noindex (empty or demo-only lists:
  `blog.index`, `directory.index`, `shop.index`) — reuse the same rule the controllers use (no duplicated logic).
- Author pages without a real bio (placeholder `[...]` / too short) are noindex even for reviewers.
- Remove the workaround in `AuditTest`; add tests for both rules.

## Acceptance
- `php artisan seo:audit` on the dev DB: 0 errors (warnings for og:image / thin demo posts are fine).
- `composer verify` green.
