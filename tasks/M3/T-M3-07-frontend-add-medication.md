---
id: T-M3-07
title: Frontend — Add/Edit medication screen
milestone: M3
type: frontend
status: todo
depends_on: [T-M3-05]
parallel_group: M3-C
touches: [frontend/src/screens/reminder-medication-form, frontend/src/app/[locale]/reminders/medication]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M3-07 — Frontend — Add/Edit medication screen

## Why
Artboard `v13_AddMedication`. Routes `/reminders/medication/new` and `/reminders/medication/[id]` (edit, with delete).

## Scope
1. Card 1: name input (pill icon), dose + unit (grid 1.2fr/1fr), form chips قرص/کپسول/شربت/آمپول/قطره.
2. Card 2: times per day segmented ۱–۴ بار → that many time rows «نوبت n · صبح/ظهر/عصر/شب · [HH:MM]» with the
   app's time-wheel picker and sensible defaults (08:00 / 20:00 / 14:00 / 23:00); weekday circles ش..ج (all on =
   «هر روز»); amount stepper with the form's unit label.
3. Card 3: start date (date picker, default today, locale calendar), duration row with «تغییر» (ongoing / until
   date / until end of pregnancy — last only when in pregnancy mode), notification switch, note input.
4. «ذخیره یادآور» → `manage-medication`; zod form validation mirroring the API; server 422 mapped to fields;
   on success invalidate + return to origin. Edit mode adds «حذف» with confirm.

## Out of scope
Push delivery.

## Acceptance
- Light/dark + fa/en screenshots; form unit tests (times ↔ count, weekday summary text); build green.
