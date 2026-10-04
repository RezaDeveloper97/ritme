---
id: L1-05
title: Routing, clean URLs, legacy 301s, trailing slash, error pages
milestone: L1
type: backend
status: done
depends_on: [L1-02,L1-03]
parallel_group: L1-D
touches: [routes/web.php,app/Http/Controllers,app/Http/Middleware,resources/views/errors,bootstrap/app.php,tests/Feature/Routing]
skills: []
verify: composer verify
---

# L1-05 — Routing, clean URLs, legacy 301s, trailing slash, error pages

## Why
URL hygiene is SEO hygiene: one canonical URL per resource and 301s from every old URL.

## Scope
- Named routes for every page in the audit URL map (thin invokable controllers returning placeholder views until
  the page task lands); no closures (route cache must work).
- Middleware: canonical host + https redirect (config-driven, off in local), trailing-slash removal (301), duplicate
  slashes, uppercase → lowercase for static paths, `*.html` → new route (301) from the audit map, WordPress-style
  `/cycle/` slugs (301).
- Error views `errors/404|410|419|429|500|503.blade.php` in the site design, `noindex`, helpful links (search once
  L4-04 exists, home, popular pages); 404 does not hit the DB except the redirect lookup (L7-03 later).
- Tests for every redirect rule and status code.

## Out of scope
- DB-driven redirects (L7-03).

## Acceptance
- `php artisan route:cache` works; all audit URLs return 200 (placeholder) or 301 → 200; tests green.
