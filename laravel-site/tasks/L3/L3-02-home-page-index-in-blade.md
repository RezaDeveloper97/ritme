---
id: L3-02
title: Home page (index) in Blade
milestone: L3
type: frontend
status: todo
depends_on: [L3-01,L1-04]
parallel_group: L3-B
touches: [resources/views/pages/home.blade.php,resources/views/pages/home,app/Http/Controllers/HomeController.php,lang/fa/home.php,tests/Feature/Pages/HomeTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design index.html --route /
---

# L3-02 — Home page (index) in Blade

## Why
Highest-traffic, highest-stakes page for SEO and first impression.

## Scope
- Convert `design/html/index.html` section by section using components; copy into `lang/fa/home.php` (or inline Blade
  when one-off) — no copy in controllers.
- «خواندنی‌های این هفته» pulls latest posts once L4 exists: build with a `HomeReadings` view model returning an empty
  state / design sample until then (no fake hard-coded posts in the view).
- «قبل از نصب» FAQ → `x-faq` with the FAQ group `home` once L3-09 lands (static items until then, swap in L3-09).
- SEO: unique title/description from design `<title>`/meta, JSON-LD Organization + WebSite + MobileApplication +
  WebPage; LCP hero element identified and prioritised (no lazy on it); all illustrations with width/height.
- Fragment-cache heavy static sections (cache-aside `pages` ns).

## Acceptance
- Diff < 3% at 390/1440 (or explained); `seo:audit` passes for `/`; tests green.
