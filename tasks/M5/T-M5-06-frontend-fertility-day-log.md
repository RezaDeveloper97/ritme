---
id: T-M5-06
title: Frontend — «ثبت روز» fertility day log screen
milestone: M5
type: frontend
status: done
depends_on: [T-M5-04]
parallel_group: M5-B
touches: [frontend/src/screens/fertility-log, frontend/src/app/[locale]/fertility/log]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M5-06 — Frontend — «ثبت روز» fertility day log screen

## Why
Artboard `v19_TTC_Log` / `nb2_TTC_Log`.

## Scope
1. `/fertility/log?date=YYYY-MM-DD&focus=` (default today); header «ثبت روز» + «۲۹ شهریور · روز ۱۲».
2. Chance card (gradient, target icon, level, 5 bars) → tap opens Insights.
3. Sections: LH chips (ثبت نشه/منفی/کم‌رنگ/مثبت), BBT field (Lalezar input, °C, −/+ 0.01, clamps 35.00–38.50,
   hint «صبح، قبل از بلندشدن از رختخواب»), cervical mucus chips, intercourse chips, symptom multi-chips, note.
   `focus` scrolls to and highlights that section.
4. «ذخیره» → `log-fertility-day` (only changed fields), toast, back. Dirty-state guard on leaving.

## Acceptance
- Light/dark × fa/en screenshots; round-trip test (load → edit → save payload); build green.
