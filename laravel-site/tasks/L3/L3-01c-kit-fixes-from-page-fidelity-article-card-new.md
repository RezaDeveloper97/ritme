---
id: L3-01c
title: Kit fixes from page fidelity: article card, newsletter input, store badges, app-cta
milestone: L3
type: frontend
status: done
depends_on: [L3-02,L3-03,L4-02]
parallel_group: L3-A
touches: [resources/views/components/ui,resources/views/components/cards,resources/views/components/layout,resources/views/pages/stages/show.blade.php,resources/views/pages/blog/index.blade.php,resources/views/pages/home,resources/views/pages/blog/partials,tests/Feature/Components]
skills: []
verify: composer verify
---

# L3-01c — Kit fixes from page fidelity: article card, newsletter input, store badges, app-cta

## Why
Page tasks L3-02, L3-03 and L4-02 hit the same component-kit deviations and patched them locally with arbitrary
selectors (e.g. `[&_a>span:first-child]:box-content` / `h-54` on `x-stage.readings` / the blog grid).

## Scope
- `x-cards.article`: cover renders like the design (180 px + padding, content-box ⇒ 216 px); featured variant mobile
  padding (`ps-0`); remove the page-level workarounds in stages/show, blog index, home.
- `x-ui.newsletter`: accept `value` (old input) and `aria-describedby`/error slot; keep design padding on mobile (`p-9`).
- Footer store badges in a row like the design (L1-02 footer); `x-ui.app-cta` height at 390 matches the design.
- Replace local `strtr` digit maps in blog pagination and error views with `fa_digits()`.

## Out of scope
- New components.

## Acceptance
- Workarounds removed; component tests updated; home/cycle/blog shots not worse than before (record numbers).
- `composer verify` green.
