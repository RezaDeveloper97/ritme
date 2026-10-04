---
id: L3-03
title: Life-stage page template + cycle page
milestone: L3
type: frontend
status: todo
depends_on: [L3-01,L1-04]
parallel_group: L3-B
touches: [resources/views/pages/stages,app/Domain/Content/Stages,app/Http/Controllers/StagePageController.php,lang/fa/stages,tests/Feature/Pages/StagesTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design cycle.html --route /cycle
---

# L3-03 — Life-stage page template + cycle page

## Why
cycle, ttc, pregnancy, postpartum, menopause and teen share one skeleton (hero, 3 feature sections, «کارهای کوچک»,
«وقتی کمک بیشتری لازم داری», «برای همین مرحله», CTA). One template + data per stage = DRY.

## Scope
- `StageDefinition` data objects (one class or content file per stage) consumed by `pages/stages/show.blade.php`
  with per-stage section partials where the design differs (KISS: shared where identical, partial where not).
- Implement **cycle** fully; template ready for the others.
- SEO: title/description from design, WebPage + BreadcrumbList (خانه › مرحله‌ها › پیگیری چرخه), FAQ schema only if
  the page has visible FAQs; internal links to tools/blog category for the stage.

## Acceptance
- `/cycle` diff < 3% at 390/1440; `seo:audit` passes; tests green.
