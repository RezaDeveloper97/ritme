---
id: L9-02
title: Lighthouse / GTmetrix sweep on every template
milestone: L9
type: quality
status: done
depends_on: [L9-01,L8-02]
parallel_group: L9-B
touches: [tools/lighthouse.mjs,docs/qa/perf,resources,app/Http/Middleware]
skills: []
verify: node tools/lighthouse.mjs --all
---

# L9-02 — Lighthouse / GTmetrix sweep on every template

## Scope
- Run the app production-like (`APP_ENV=production APP_DEBUG=false`, `php artisan optimize`, page cache on, real
  build) and `npx lighthouse` (mobile + desktop presets) on one URL per template (home, stage, tools, faq, contact,
  blog list, article, directory list, place, shop, category, product, cart, 404).
- `tools/lighthouse.mjs --all` writes JSON + HTML reports to `docs/qa/perf/<date>/` and a summary table; fails when
  below targets (Perf ≥ 95 mobile, SEO 100, BP 100, A11y ≥ 95, CLS < 0.1, LCP < 2.5 s, TBT < 150 ms).
- Fix regressions found (lazy/eager images, CLS from fonts/icons, long tasks, uncompressed responses, cache headers).
- Optional: run the same URLs through GTmetrix manually after stage deploy (needs the user's account) — document.

- From L9-01: product page lab CLS 0.159 (category chip row in the product header wraps on Vazirmatn 800 swap — fix
  with `flex-nowrap` + overflow, then drop the `BUDGET_OVERRIDES` entry); home FCP 1.3 s lab (`rel=expect` on the
  SVG-heavy `<main>`); optional Arabic font subset incl. ASCII punctuation (~80 KB/page).

## Acceptance
- Summary table all green (or justified); reports committed as JSON only (HTML ignored).
