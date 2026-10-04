---
id: L5-01
title: Directory context: places, categories, cities, amenities, reviews
milestone: L5
type: backend
status: done
depends_on: [L2-01,L1-03,L4-04]
parallel_group: L5-A
touches: [app/Domain/Directory,database/migrations,database/factories,database/seeders/DirectorySeeder.php,tests/Feature/Directory,tests/Unit/Directory]
skills: []
verify: composer verify
---

# L5-01 — Directory context: places, categories, cities, amenities, reviews

## Why
«خدمات مادر و کودک» is local-SEO content (city × category landing pages) and needs structured data from day one.

## Scope
- `Place` (name, slug + history, category, city, district, address, lat/lng, phone(s), website, description,
  amenities M2M, age ranges, opening hours JSON (per weekday, ranges), services + prices (name, duration, price),
  rules/cancellation text, gallery media (ordered, cover), status draft|published|suspended, is_verified,
  rating_avg/count cached), `PlaceCategory` (schema.org type mapping, e.g. استخر → `SportsActivityLocation`,
  مهدکودک → `ChildCare`), `City`, `District`, `Amenity` (icon name), `PlaceReview` (moderated).
- Cached repos (`directory` ns), query object `SearchPlaces` (city, category, age, amenities, sort), observers bump
  `directory`, `pages`, `sitemap`; sitemap providers (places, city × category combos with ≥ 1 place); search provider for places.
- Demo seeder from the design (آب‌پری pool etc.).

## Acceptance
- Factories + tests for search filters, opening-hours helpers (open now, Persian weekday names), rating aggregate.
