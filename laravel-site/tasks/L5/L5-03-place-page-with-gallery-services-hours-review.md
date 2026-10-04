---
id: L5-03
title: Place page with gallery, services, hours, reviews, LocalBusiness schema
milestone: L5
type: frontend
status: todo
depends_on: [L5-02]
parallel_group: L5-B
touches: [resources/views/pages/directory/show.blade.php,resources/views/components/directory,app/Http/Controllers/Directory/ShowPlaceController.php,resources/js/modules/gallery.js,tests/Feature/Directory/PlaceTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design directory-place.html --route /directory/place/ab-pari
---

# L5-03 — Place page with gallery, services, hours, reviews, LocalBusiness schema

## Scope
- `/directory/place/{slug}`: gallery (first image priority, rest lazy, lightweight lightbox module), about,
  amenities, services & prices (`Money`), opening hours table (today highlighted), reviews (approved only, paginated)
  + review form (moderated, rate-limited, honeypot), address with **no embedded map** — deep links (`geo:` URI,
  Neshan, Balad, Google Maps URLs) + local static illustration, rules & cancellation.
- Schema: `LocalBusiness` subtype from category, address `PostalAddress`, `geo`, `openingHoursSpecification`,
  `priceRange`, `image`, `aggregateRating` + `review` only from real approved reviews.

## Acceptance
- Diff < 3% vs `directory-place.html`; schema valid; tests green.
