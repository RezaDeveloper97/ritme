---
id: L3-09
title: FAQ context, /faq page and reusable FAQ blocks + admin
milestone: L3
type: fullstack
status: todo
depends_on: [L3-01,L1-04,L1-08]
parallel_group: L3-E
touches: [resources/views/pages/home.blade.php,resources/views/pages/plus.blade.php,app/Domain/Faq,database/migrations,database/seeders/FaqSeeder.php,resources/views/pages/faq.blade.php,resources/views/components/faq,app/Http/Controllers/FaqController.php,app/Filament/Resources/Faq,tests/Feature/Faq]
skills: []
verify: composer verify && node tools/shot.mjs --design faq.html --route /faq
---

# L3-09 — FAQ context, /faq page and reusable FAQ blocks + admin

## Scope
- `faq_groups` (slug, title, sort) + `faq_items` (group, question, answer rich text sanitised, sort, is_published);
  seeded from every FAQ in the design (faq, home «قبل از نصب», plus, directory-business, contact «شاید جوابت اینجا
  باشد»). Cached repository (`faq` ns).
- `/faq` grouped by category with in-page anchor nav; `x-faq :group="'plus'"` reusable block; FAQPage JSON-LD emitted
  once per page from all visible items.
- Swap the static FAQ placeholders in L3-02/L3-06 for the component.
- Filament: groups + items with drag-sort, publish toggle, preview.

## Acceptance
- Diff < 3%; FAQ schema valid; editing an item in admin updates `/faq` (cache bumped); tests green.
