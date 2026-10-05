---
id: L7-03
title: Redirect manager, 404 monitor, auto-redirect on slug change
milestone: L7
type: admin
status: done
depends_on: [L7-01,L1-05]
parallel_group: L7-B
touches: [app/Domain/Seo/Redirects,database/migrations,app/Http/Middleware/ApplyRedirects.php,app/Filament/Resources/Seo/Redirects,app/Filament/Resources/Seo/NotFoundLogs,tests/Feature/Seo/RedirectsTest.php]
skills: []
verify: composer verify
---

# L7-03 — Redirect manager, 404 monitor, auto-redirect on slug change

## Scope
- `redirects` (from_path, to_url, code 301|302|307|410, is_regex, hits, last_hit_at, note); map cached
  (cache-aside `seo` ns, exact-match hash map + small regex list); applied before the 404 response only (zero cost
  for normal pages); loop/chain detection on save (A→B→C collapses to A→C).
- `not_found_logs` (path, referer, user agent class (bot/human), hits, last_seen) written via a batched/deduped
  insert (no write per request storms), bots ignored by default; "create redirect" action; purge older than 90 days
  (scheduler).
- Observers on Post/Product/Place/Category slug change create 301s automatically.
- CSV import/export of redirects (migration from the old WordPress site).

## Acceptance
- Chain collapse, regex, 410, auto-redirect tested; 404 monitor dedupes; tests green.
