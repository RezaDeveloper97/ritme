---
id: L0-06
title: style2tw: inline-style → Tailwind converter for design pages
milestone: L0
type: frontend
status: done
depends_on: [L0-05]
parallel_group: L0-C
touches: [tools/style2tw.mjs,tools/README.md,tests/tools]
skills: []
verify: node --test tests/tools
---

# L0-06 — style2tw: inline-style → Tailwind converter for design pages

## Why
Hand-translating ~7,000 inline styles is slow and error-prone. A converter gives a faithful first pass per page;
humans then extract components.

## Scope
- `node tools/style2tw.mjs design/html/cycle.html [--section N] > /tmp/out.blade.php` (no npm deps beyond what's
  already installed; use a tolerant HTML tokenizer — e.g. `parse5` as a devDependency is fine).
- Maps CSS declarations to Tailwind v4 utilities using the project theme tokens (colors → token names when exact
  match, else arbitrary `bg-[#hex]` flagged in a report), spacing/size → scale or arbitrary values, flex/grid,
  typography, borders/radius, gradients → arbitrary `bg-[...]`; `left/right/margin-left/right` → logical
  `start/end/ms/me` (RTL!).
- Encodes the responsive rules of `design/html/assets/ritme.css` as variants (e.g. `padding:… 120px` →
  `px-[120px] max-lg:px-5`; `repeat(4,…)` grids → `max-lg:grid-cols-2 max-md:grid-cols-1`; hero font sizes → the
  same step-downs) so the converted page is responsive without attribute selectors.
- Strips `style=""`, keeps semantic markup, replaces `href="x.html"` with `{{ route('…') }}` from the audit URL map,
  replaces known SVG icons with `<x-icon name="…"/>` (after L0-07; until then leaves them).
- Report to stderr: unmapped declarations, colors outside the palette, arbitrary-value counts.
- `node:test` unit tests for the declaration mapper (padding shorthand, RTL mapping, gradients, responsive rules).

## Out of scope
- Component extraction (done by hand in page tasks).

## Acceptance
- Converting `index.html` produces Blade with zero `style=` attributes that renders visually close to the original
  (checked in L0-08); tests green.
