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
- Audit corrections (`docs/AUDIT.md` §2): the design has **no JS tabs**. Every tab-like UI (blog/directory categories,
  shop sort, shop subnav, faq side nav, contact topics) is a link list, so build `x-ui.chip-nav` (links, no JS)
  instead of a tabs module. Also build: `x-ui.alert-emergency` (۱۱۵ as `tel:`), `x-ui.qr` (server-side SVG QR of the
  app link via a local PHP library, cached; never an external QR service), `x-ui.success-hero`, `x-ui.stepper`,
  `x-ui.timeline`, `x-ui.toggle-row`, `x-ui.promise-banner`, `x-ui.app-cta` (+ aside variant), `x-ui.page-intro`,
  `x-ui.promo-split`, `x-cards.value`, `x-cards.review`. Single-use blocks (quote, pricing, team, checklist,
  category tile) stay with their page tasks.

## Acceptance
- `/_components` renders all components; spot-checked against design; no inline styles; tests for helpers.
