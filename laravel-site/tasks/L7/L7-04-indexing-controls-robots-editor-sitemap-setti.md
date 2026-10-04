---
id: L7-04
title: Indexing controls: robots editor, sitemap settings, IndexNow, verification, head code
milestone: L7
type: admin
status: todo
depends_on: [L7-01,L1-06]
parallel_group: L7-C
touches: [app/Filament/Pages/Seo,app/Domain/Seo/Indexing,tests/Feature/Admin/IndexingTest.php]
skills: []
verify: composer verify
---

# L7-04 — Indexing controls: robots editor, sitemap settings, IndexNow, verification, head code

## Scope
- robots.txt editor (validated, preview, reset; production only takes effect), sitemap settings (include/exclude
  types, priorities, ping list), "regenerate sitemap" action, per-type default robots (e.g. tags noindex).
- Search engine verification codes (Google, Bing, Yandex) → meta tags.
- IndexNow (Bing/Yandex): key file route, submit on publish/update via queued job — **off by default** (it is a
  server-side outbound call; the public site still makes zero external requests).
- "Custom head code" field restricted to super-admin with a visible warning that external scripts cost GTmetrix
  points and break the strict CSP (CSP must be adjusted explicitly in the same form).

## Acceptance
- robots change reflects at `/robots.txt` (prod env test); IndexNow job tested with HTTP fake; tests green.
