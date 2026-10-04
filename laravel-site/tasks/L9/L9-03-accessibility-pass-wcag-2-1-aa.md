---
id: L9-03
title: Accessibility pass (WCAG 2.1 AA)
milestone: L9
type: quality
status: todo
depends_on: [L3-11,L4-03,L5-03,L6-05]
parallel_group: L9-C
touches: [resources/views,resources/css,resources/js,docs/qa/a11y]
skills: [design:accessibility-review]
verify: composer verify
---

# L9-03 — Accessibility pass (WCAG 2.1 AA)

## Scope
- Landmarks, heading order, link purpose, color contrast of the token palette (fix tokens if < 4.5:1), focus-visible
  on every interactive element, form labels/errors (`aria-describedby`, `aria-invalid`), menu/accordion/tabs/gallery
  keyboard support, `prefers-reduced-motion`, touch targets ≥ 44 px, `lang`/`dir` correctness for mixed Latin text.
- Report `docs/qa/a11y/README.md` per template with fixes.

## Acceptance
- Lighthouse a11y ≥ 95 everywhere; keyboard walkthrough documented; tests green.
