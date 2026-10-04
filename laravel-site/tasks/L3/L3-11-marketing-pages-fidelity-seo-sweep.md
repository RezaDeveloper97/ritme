---
id: L3-11
title: Marketing pages fidelity + SEO sweep
milestone: L3
type: quality
status: done
depends_on: [L3-02,L3-03,L3-04,L3-05,L3-06,L3-07,L3-08,L3-09,L3-10]
parallel_group: L3-F
touches: [docs/qa/L3,resources/views,resources/css,lang/fa]
skills: []
verify: composer verify && node tools/shot.mjs --all --out docs/qa/L3 && php artisan seo:audit
---

# L3-11 — Marketing pages fidelity + SEO sweep

## Scope
- Run `tools/shot.mjs --all` at 390/768/1440; fix every diff > 3% or document why it's intentional.
- `seo:audit` across all marketing routes: unique titles/descriptions, one h1, internal links resolve (no `#`).
- Check all `href="#download"`/store links resolve to settings; no external request in any page.
- Write `docs/qa/L3/README.md` with the table (page, diff per width, audit status).

## Acceptance
- All rows green or justified; tests green.
