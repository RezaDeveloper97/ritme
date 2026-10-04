---
id: L4-04
title: RSS feed, site search, WebSite SearchAction
milestone: L4
type: backend
status: done
depends_on: [L4-03]
parallel_group: L4-C
touches: [app/Http/Controllers/FeedController.php,app/Http/Controllers/SearchController.php,app/Domain/Search,resources/views/pages/search.blade.php,resources/views/feed,routes/web.php,tests/Feature/FeedSearchTest.php]
skills: []
verify: composer verify
---

# L4-04 — RSS feed, site search, WebSite SearchAction

## Scope
- `/blog/feed` (RSS 2.0, last 30 posts, full excerpt + cover enclosure), `link rel=alternate` in head on blog pages.
- `SearchProvider` contract; posts + FAQ providers now (directory and shop register theirs in L5-01 / L6-01).
- `/search?q=` across all registered providers (simple `LIKE` + MySQL FULLTEXT when available, normalising
  Persian/Arabic ی/ك and ZWNJ), `noindex,follow`, rate-limited, results cached briefly (cache-aside keyed by
  normalised query).
- `WebSite` schema gets `potentialAction` `SearchAction`; search box in header/mobile menu (progressive).
- Search terms logged (anonymised) for the admin «جستجوهای بی‌نتیجه» report (L7-05).

## Acceptance
- Feed validates (W3C feed rules checked in test); Persian normalisation tests; tests green.
