---
id: L7-01
title: Static pages SEO manager with SERP + social previews
milestone: L7
type: admin
status: todo
depends_on: [L4-05,L3-11]
parallel_group: L7-A
touches: [app/Filament/Resources/Seo/StaticPageSeo,app/Domain/Seo,tests/Feature/Admin/StaticPageSeoTest.php]
skills: []
verify: composer verify
---

# L7-01 — Static pages SEO manager with SERP + social previews

## Why
Marketing pages are Blade, but their SEO must be editable by the SEO manager without a developer.

## Scope
- Filament resource listing every entry of the `StaticPage` registry (route, URL, current title/description, status
  icons: length ok, unique, OG image set, indexable) with edit form using `SeoFields` (SERP desktop/mobile preview,
  OG/Telegram/WhatsApp card preview, pixel-width counters, focus keyword, robots, canonical override, OG image
  upload → auto `og` crop variant).
- "View live" and "reset to defaults" actions; saving bumps `seo` + `pages` namespaces.

## Acceptance
- Editing `/cycle` title in admin changes the live `<title>` immediately; tests green.
