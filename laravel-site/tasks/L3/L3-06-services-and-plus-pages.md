---
id: L3-06
title: Services and Plus pages
milestone: L3
type: frontend
status: done
depends_on: [L3-01,L1-04]
parallel_group: L3-C
touches: [resources/views/pages/services.blade.php,resources/views/pages/plus.blade.php,app/Http/Controllers/ServicesController.php,app/Http/Controllers/PlusController.php,lang/fa/services.php,lang/fa/plus.php,tests/Feature/Pages/ServicesPlusTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design services.html --route /services && node tools/shot.mjs --design plus.html --route /plus
---

# L3-06 — Services and Plus pages

## Scope
- `/services` (links into directory + shop), `/plus` (free vs Plus table, FAQ «جواب سؤال‌هایی که حق داری بپرسی»
  → FAQ group `plus` after L3-09). Prices (if any) come from settings, not markup.
- SEO: services → `CollectionPage`; plus → `WebPage` + `FAQPage` (visible FAQs only); no fake offers/ratings.

## Acceptance
- Diff < 3% at 390/1440; `seo:audit` passes; tests green.
