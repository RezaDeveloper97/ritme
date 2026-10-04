---
id: L4-02
title: Magazine list, category, tag, author pages + newsletter signup
milestone: L4
type: frontend
status: done
depends_on: [L4-01,L3-01]
parallel_group: L4-B
touches: [resources/views/pages/blog,app/Http/Controllers/Blog,app/Domain/Newsletter,database/migrations,lang/fa/blog.php,tests/Feature/Blog/ListingTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design blog.html --route /blog
---

# L4-02 — Magazine list, category, tag, author pages + newsletter signup

## Scope
- `/blog` (featured hero post, grid, category chips), `/blog/category/{slug}`, `/blog/tag/{slug}` (noindex,follow
  when < N posts — configurable), `/blog/author/{slug}` (Person schema, credentials).
- Pagination `?page=n` with self-canonical, page number in title, out-of-range → 404.
- `CollectionPage` + `ItemList` + BreadcrumbList schema.
- Newsletter context: «هر هفته یک خواندنی کوتاه» form → `newsletter_subscribers` (email, source, consent at,
  unsubscribed_at, token), honeypot + rate limit, double-opt-in mail (log driver default), unsubscribe link route;
  Filament list + CSV export (L4-05 may host it).
- Home «خواندنی‌های این هفته» now uses `LatestPosts`.

## Acceptance
- Diff < 3% vs `blog.html`; `seo:audit` passes on list/category/author; tests green.
