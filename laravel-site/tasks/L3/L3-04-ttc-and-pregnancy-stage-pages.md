---
id: L3-04
title: TTC and pregnancy stage pages
milestone: L3
type: frontend
status: done
depends_on: [L3-03]
parallel_group: L3-C
touches: [lang/fa/stages/ttc.php,lang/fa/stages/pregnancy.php,resources/views/pages/stages/partials/ttc,resources/views/pages/stages/partials/pregnancy,app/Domain/Content/Stages/Ttc.php,app/Domain/Content/Stages/Pregnancy.php,tests/Feature/Pages/StagesTtcPregnancyTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design ttc.html --route /ttc && node tools/shot.mjs --design pregnancy.html --route /pregnancy
---

# L3-04 — TTC and pregnancy stage pages

## Scope
- `/ttc` (design `ttc.html`, incl. IVF/IUI section) and `/pregnancy` (`pregnancy.html`, 40-week, appointments,
  hospital bag/birth plan/sisemoni) on the L3-03 template; stage-specific partials only where needed.
- SEO per page; links to the due-date / fertility calculators on `/tools`.

## Acceptance
- Diff < 3% at 390/1440 for both; `seo:audit` passes; tests green.
