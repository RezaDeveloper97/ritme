---
id: L3-01
title: Shared UI component kit from the audit
milestone: L3
type: frontend
status: todo
depends_on: [L1-02,L2-02]
parallel_group: L3-A
touches: [resources/views/components/ui,resources/views/components/stage,resources/views/components/cards,resources/js/modules,docs/COMPONENTS.md]
skills: []
verify: composer verify
---

# L3-01 — Shared UI component kit from the audit

## Why
Page tasks must assemble pages from components, not paste converter output, to stay DRY and consistent.

## Scope
- Build every component listed in `docs/AUDIT.md` §2 that is used by ≥ 2 pages: section header, buttons + store
  badges, pills, cards (feature, stage, article, product, place, service), accordion (`<details>` — no JS), steps,
  rating, price (`Money` formatting, Persian digits), app-download CTA (links from settings), stage blocks («کارهای
  کوچک…», «وقتی کمک بیشتری لازم داری», «برای همین مرحله»), quote, newsletter box (form wired in L4-02), tabs (JS
  module, progressive), gallery.
- `docs/COMPONENTS.md`: name, props, example, which design pages use it.
- A hidden `/_components` route (local only) rendering every component for review + screenshots.
- Persian digits helper (`fa_digits()`), Jalali date helper (`App\Support\Jalali`, tested) for dates in cards.

## Acceptance
- `/_components` renders all components; spot-checked against design; no inline styles; tests for helpers.
