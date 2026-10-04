---
id: L3-05
title: Postpartum, menopause and teen stage pages
milestone: L3
type: frontend
status: todo
depends_on: [L3-03]
parallel_group: L3-C
touches: [lang/fa/stages/postpartum.php,lang/fa/stages/menopause.php,lang/fa/stages/teen.php,resources/views/pages/stages/partials/postpartum,resources/views/pages/stages/partials/menopause,resources/views/pages/stages/partials/teen,app/Domain/Content/Stages/Postpartum.php,app/Domain/Content/Stages/Menopause.php,app/Domain/Content/Stages/Teen.php,tests/Feature/Pages/StagesOtherTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design postpartum.html --route /postpartum && node tools/shot.mjs --design menopause.html --route /menopause && node tools/shot.mjs --design teen.html --route /teen
---

# L3-05 — Postpartum, menopause and teen stage pages

## Scope
- `/postpartum`, `/menopause`, `/teen` on the template (teen: «مادر در جریان است، نه ناظر» parent section).
- SEO per page; teen copy must stay age-appropriate (red lines).

## Acceptance
- Diff < 3% at 390/1440 for all three; `seo:audit` passes; tests green.
