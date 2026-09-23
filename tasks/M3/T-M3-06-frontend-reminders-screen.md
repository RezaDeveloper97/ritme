---
id: T-M3-06
title: Frontend — Reminders screen (/reminders) with tabs, dose strip and lists
milestone: M3
type: frontend
status: todo
depends_on: [T-M3-05]
parallel_group: M3-C
touches: [frontend/src/screens/reminders, frontend/src/app/[locale]/reminders/page.tsx, frontend/src/screens/profile/ui/ProfilePage.tsx]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M3-06 — Frontend — Reminders screen (/reminders) with tabs, dose strip and lists

## Why
Artboard `v13_Reminders` (light `nbl_`, dark `nbd_`): the hub the «همه» link and profile point to.

## Scope
1. Header: back, title «یادآورها» + subtitle «داروها و نوبت‌ها», notification-settings button (opens the existing
   notifications sheet).
2. Segmented tabs همه / داروها / نوبت‌ها (real `role="tab"`), filtering the sections below; tab kept in the URL.
3. «امروز» card: badge «N از M مصرف شد», horizontal scroll of dose cards (time, check, name) — tick via `log-intake`.
4. «داروها و مکمل‌ها» list: tinted icon tile per form, name, «frequency · time(s) · amount form», active switch
   (`manage-medication` toggle); row tap → edit. «+ افزودن» → AddMedication.
5. «نوبت‌ها و مشاوره‌ها» list: date tile (amber for in-person, teal for online/phone), title, «time · kind · with»,
   chevron → detail. «+ افزودن» → AddAppointment.
6. Primary CTA «افزودن یادآور جدید» opens the `reminders-add` sheet. Empty state per section.
7. Point the profile's reminders row to `/reminders` (keep the old sheet registered).

## Out of scope
Forms and detail (T-M3-07, T-M3-08).

## Acceptance
- Matches the artboard in light and dark, fa and en; screenshots attached to PROGRESS.
- Build green.
