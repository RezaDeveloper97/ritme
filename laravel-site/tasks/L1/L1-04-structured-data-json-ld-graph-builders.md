---
id: L1-04
title: Structured data (JSON-LD graph) builders
milestone: L1
type: backend
status: todo
depends_on: [L1-03]
parallel_group: L1-C
touches: [app/Domain/Seo/Schema,tests/Unit/Seo/Schema]
skills: []
verify: composer verify
---

# L1-04 — Structured data (JSON-LD graph) builders

## Why
Rich results (FAQ, breadcrumbs, product, article, organization, app) drive CTR; one graph per page avoids conflicts.

## Scope
- `spatie/schema-org` (or hand-rolled small builders if lighter — decide and note in ARCHITECTURE.md).
- `SchemaGraph` collected through the request; builders: `Organization` (logo, sameAs, contactPoint from settings),
  `WebSite` (+ `SearchAction` once L4-04 ships), `WebPage`/`AboutPage`/`ContactPage`/`CollectionPage`,
  `BreadcrumbList` (from `x-ui.breadcrumbs`), `MobileApplication` (app links, operatingSystem, offers price 0,
  no fake ratings), `FAQPage`, `BlogPosting` (+ author `Person`, `reviewedBy`/`medicalReviewer` hooks for YMYL),
  `Product`/`Offer`/`AggregateRating` (only from real reviews), `LocalBusiness` subtypes / `ChildCare` /
  `SportsActivityLocation` with `OpeningHoursSpecification`, `ItemList`.
- Single `<script type="application/ld+json">` with `@graph` and stable `@id`s (`{url}#organization`, `#website`,
  `#webpage`, `#breadcrumb`).
- Unit tests assert valid JSON + required properties per Google docs; snapshot of a sample graph.

## Out of scope
- Page wiring (each page task adds its schema nodes).

## Acceptance
- Graph renders in the layout; tests green; Rich Results requirements documented in `docs/SEO.md` (new, short).
