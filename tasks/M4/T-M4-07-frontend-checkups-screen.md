---
id: T-M4-07
title: Frontend — Checkups screen (/checkups) and custom checkup form
milestone: M4
type: frontend
status: todo
depends_on: [T-M4-05]
parallel_group: M4-B
touches: [frontend/src/screens/checkups, frontend/src/screens/checkup-custom-form, frontend/src/app/[locale]/checkups/page.tsx, frontend/src/app/[locale]/checkups/custom]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M4-07 — Frontend — Checkups screen (/checkups) and custom checkup form

## Why
Artboard `v14_Checkups`.

## Scope
1. Header: back, «چکاپ‌های دوره‌ای» + «بر اساس سن N سال», plan-settings button (sheet listing types with
   enabled/remind switches → settings endpoint).
2. Status card: «N از M به‌روز» (Lalezar), status pill for the worst state, stacked bar (green/amber/rose, LTR),
   legend.
3. Tabs همه / نیاز به اقدام / انجام‌شده (URL-synced) → `filter`.
4. Sections in server order with titles (این ماه, عقب‌افتاده, سالانه, هر ۶ ماه, بر اساس سن, سفارشی); row = tinted
   icon tile, title, interval/timing line, «آخرین: … · بعدی: …» line, status pill (به‌روز / موعدش رسیده /
   عقب‌افتاده / به‌زودی) → detail.
5. Info note + outline button «افزودن چکاپ سفارشی» → `/checkups/custom/new` (title, interval chips
   ۱ ماه/۶ ماه/۱ سال/۲ سال/۳ سال, performed_by, last-done date optional, note) with edit/delete.
6. Empty and error states.

## Acceptance
- Light/dark × fa/en screenshots; build green.
