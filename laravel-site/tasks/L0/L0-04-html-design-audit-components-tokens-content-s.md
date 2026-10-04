---
id: L0-04
title: HTML design audit: components, tokens, content, SEO gaps, URL map
milestone: L0
type: investigate
status: todo
depends_on: [L0-01]
parallel_group: L0-B
touches: [docs/AUDIT.md]
skills: []
verify: test -s docs/AUDIT.md
---

# L0-04 — HTML design audit: components, tokens, content, SEO gaps, URL map

## Why
29 pages of inline-styled HTML (~7,000 inline style attributes). Before converting, we need one reference of what
repeats, what is data, and what is broken for SEO, so every page task reuses the same components.

## Scope
Read every file in `design/html/` and write `docs/AUDIT.md` with:
1. **Page inventory**: file → new route (see `tasks/README.md` URL decision) → sections (h1/h2 outline) → owning task.
2. **Shared components** with the pages that use them: header (dark hero variant vs light), nav + active state,
   mobile menu, footer, breadcrumbs, buttons (primary/secondary/ghost/store badges), pills/badges, section header,
   cards (stage, feature, article, product, place, service), accordion (`details`), app-download CTA, «کارهای کوچک،
   آمادگی بیشتر» / «وقتی کمک بیشتری لازم داری» / «برای همین مرحله» stage blocks, forms/inputs, steps, rating stars,
   price, gallery, tabs, quote/testimonial, newsletter box.
3. **Design tokens**: every distinct color (with usage counts), font sizes/weights/line-heights, radii, shadows,
   gradients (dark hero radial glows), spacing scale (120px gutters → 20px mobile), breakpoints (1180/1024/700) and
   the responsive rules in `assets/ritme.css` mapped to intent.
4. **Icons & illustrations**: unique inline SVGs (dedupe by path data) with suggested names; big illustrations list.
5. **Dynamic data** per page (posts, products, places, FAQs, settings like app-store links, prices) vs static copy;
   placeholders (`[اینماد]`, `[ایمیل مسئول داده]`, `#download`) to become settings.
6. **SEO/perf gaps** in the export: missing canonical/OG/JSON-LD, heading order issues, links to `#`, contrast
   concerns, forms without labels, JS needs (menu, calculators, accordions).
7. **URL map** old → new, including `*.html` and WordPress-snippet slugs (`/cycle/` …) for 301s.
8. Corrections: if any later task's scope is wrong after the audit, edit that task file and list the change here.

## Out of scope
- Code.

## Acceptance
- `docs/AUDIT.md` has all 8 sections; every page appears in the inventory; components list names the Blade
  component each becomes (`x-ui.button`, `x-stage.help-block`, …).
