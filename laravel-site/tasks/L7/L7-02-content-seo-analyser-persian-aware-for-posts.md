---
id: L7-02
title: Content SEO analyser (Persian-aware) for posts, products, places, pages
milestone: L7
type: backend
status: todo
depends_on: [L7-01]
parallel_group: L7-B
touches: [app/Domain/Seo/Analysis,app/Filament/Components/Seo,tests/Unit/Seo/Analysis]
skills: []
verify: composer verify
---

# L7-02 — Content SEO analyser (Persian-aware) for posts, products, places, pages

## Scope
- `SeoAnalyzer` returns a score + checklist (pass/warn/fail with Persian messages): title length & pixel width,
  description length, focus keyword in title / description / slug / H1 / first paragraph / subheadings / image alt,
  keyword density (0.5–2.5%), Persian normalisation (ی/ي، ک/ك، ZWNJ, diacritics) before matching, content length
  per type, heading hierarchy (no skipped levels, single H1), images missing alt, internal links ≥ 2, outbound links
  `rel` check, slug length/stopwords, readability (avg sentence length, paragraph length), duplicate title/description
  across site, cornerstone flag.
- Live analysis panel in `SeoFields` (Livewire, debounced) + score badge column in list tables + filter
  "needs work".
- Unit tests per check with Persian fixtures.

## Acceptance
- Analyser covers every check with tests; panel updates while typing; `composer verify` green.
