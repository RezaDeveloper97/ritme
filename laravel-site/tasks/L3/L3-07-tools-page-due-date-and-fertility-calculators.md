---
id: L3-07
title: Tools page: due-date and fertility calculators (JS + no-JS fallback)
milestone: L3
type: fullstack
status: done
depends_on: [L3-01,L1-04]
parallel_group: L3-D
touches: [resources/views/pages/tools.blade.php,app/Http/Controllers/ToolsController.php,app/Domain/Content/Tools,app/Support/Jalali,resources/js/modules/calculators.js,resources/js/lib/jalali.js,lang/fa/tools.php,tests/Feature/Pages/ToolsTest.php,tests/Unit/Tools,tests/js]
skills: []
verify: composer verify && node --test tests/js && node tools/shot.mjs --design tools.html --route /tools
---

# L3-07 — Tools page: due-date and fertility calculators (JS + no-JS fallback)

## Why
Calculators are classic SEO magnets («محاسبه تاریخ زایمان», «محاسبه روزهای باروری»). They must work instantly and
without JS for crawlers and slow phones.

## Scope
- Extract the Jalali + calculator logic from `design/html/assets/ritme.js` into `resources/js/lib/jalali.js` (ES
  module, `node:test` unit tests incl. leap years 1403/1408) loaded only on `/tools` via `data-module`.
- PHP twin in `App\Domain\Content\Tools` (`DueDateCalculator`, `FertilityWindowCalculator`) using
  `App\Support\Jalali` — same results as JS (shared test vectors file used by both test suites).
- Forms work with JS off: GET `/tools?calc=due&lmp=1405/02/12&cycle=28` renders the result server-side (`noindex`
  for parameterised results, canonical → `/tools`). Validation messages in Persian, «روش پیشگیری نیست» warning kept
  above the result.
- Consider separate indexable landing URLs `/tools/due-date` and `/tools/fertility-window` with their own titles
  (anchors on the same template) — implement if the audit confirms the design supports it; otherwise note for later.
- SEO: `WebApplication` schema (free, no ratings), FAQ if visible.
- Audit decision on landing URLs (`docs/AUDIT.md` §8): the design is one page with both calculators, 3 checklists
  and 4 guides. Ship stable anchors `#due-date`, `#fertility`, `#hospital-bag`, `#sisemoni` (footer links target
  them) and defer separate `/tools/due-date` / `/tools/fertility-window` URLs to a later SEO task. Remove the
  design's English `aria-label="lmp"/"cycle"`; inputs get proper Persian `<label>`s. Checklists are non-persistent
  checkboxes (no storage).

## Acceptance
- Same output in JS and PHP for the shared vectors; works with JS disabled; diff < 3%; tests green.
