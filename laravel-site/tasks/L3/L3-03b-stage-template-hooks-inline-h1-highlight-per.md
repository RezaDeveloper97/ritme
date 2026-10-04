---
id: L3-03b
title: Stage template hooks: inline h1 highlight, per-stage mock copy, shared companion screen
milestone: L3
type: frontend
status: done
depends_on: [L3-04,L3-05]
parallel_group: L3-C
touches: [app/Domain/Content/Stages,resources/views/pages/stages,lang/fa/stages,lang/fa/home.php,resources/views/pages/home,tests/Feature/Pages]
skills: []
verify: composer verify
---

# L3-03b — Stage template hooks: inline h1 highlight, per-stage mock copy, shared companion screen

## Why
L3-04/L3-05 found template gaps they could not change while working in parallel:
- `partials/hero` always renders `<span highlight>` before the title, but most stage designs highlight a phrase in the
  middle/end of the h1 — copy was reworded to fit.
- `StagePageBuilder::screenCopy()` only reads `stages/common.mock.<screen>`; stage-specific screens call `__()` directly
  and their fragment-cache key ignores lang changes.
- The companion phone screen is duplicated (home mock, `ttc-companion`, `postpartum-partner`, teen).

## Scope
- Support a `:highlight` placeholder (or `title_before`/`title_after`) in `hero.title`; restore the design's original h1
  wording for ttc, pregnancy, postpartum, menopause, teen.
- `screenCopy()` reads `<stage>.mock.<screen>` then falls back to common; include it in the mock fragment cache key.
- One shared `mock/screens/companion` used by home and all stages.

## Acceptance
- h1 text matches the design on all six stage pages; shots not worse than the L3-03/04/05 numbers.
- Stage tests updated; `composer verify` green.
