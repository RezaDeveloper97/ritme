---
id: L7-06
title: Bulk SEO editor + Organization/SEO defaults settings
milestone: L7
type: admin
status: todo
depends_on: [L7-01,L5-06,L6-06]
parallel_group: L7-C
touches: [app/Filament/Pages/Seo/BulkEditor.php,app/Filament/Pages/Settings/SeoSettings.php,app/Filament/Pages/Settings/OrganizationSettings.php,tests/Feature/Admin/BulkSeoTest.php]
skills: []
verify: composer verify
---

# L7-06 — Bulk SEO editor + Organization/SEO defaults settings

## Scope
- Bulk editor: one table across posts/products/places/categories/static pages with inline-editable meta
  title/description, counters, duplicate highlighting, filters (missing, too long, duplicate, noindex), CSV
  export/import.
- Settings pages for `SeoDefaults` (title template, separator, default description, default OG image, twitter
  handle) and `OrganizationSettings` (legal name, logo, sameAs list, contact point, founding date) feeding JSON-LD.

## Acceptance
- Bulk save of 50 rows in one action bumps caches once; tests green.
