---
id: T-M3-08
title: Frontend — Add/Edit appointment and appointment detail screens
milestone: M3
type: frontend
status: done
depends_on: [T-M3-05]
parallel_group: M3-C
touches: [frontend/src/screens/reminder-appointment-form, frontend/src/screens/reminder-appointment-detail, frontend/src/app/[locale]/reminders/appointment, frontend/src/shared/lib/ics]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M3-08 — Frontend — Add/Edit appointment and appointment detail screens

## Why
Artboards `v13_AddAppointment` and `v13_AppointmentDetail`.

## Scope
1. Form (`/reminders/appointment/new?kind=`, `/reminders/appointment/[id]/edit`): kind tiles ویزیت حضوری /
   مشاوره تلفنی / مشاوره آنلاین; «با چه کسی؟», «تخصص», topic chips (سونوگرافی, ویزیت دوره‌ای, آزمایش, مشاوره,
   واکسن, سایر), «توضیح کوتاه»; date + time buttons (Lalezar values, locale calendar pickers); «محل» (label becomes
   link/phone hint for online/phone); remind-before chips (۱ ساعت/۳ ساعت/۱ روز/۲ روز قبل); add-to-calendar switch;
   prep textarea (one line = one checklist item). «ذخیره نوبت».
2. Detail (`/reminders/appointment/[id]`): header «جزئیات نوبت» + «N روز دیگر», edit button; hero card (kind chip,
   «هفته N بارداری» chip only in pregnancy mode — computed client-side from the pregnancy entity, date tile, weekday
   · time, topic); rows با چه کسی / در خصوص / محل (+ «مسیریابی» → maps link for in-person) / یادآوری (switch →
   `is_active`); «قبل از نوبت» checklist (PATCH prep); «افزودن به تقویم» (downloads an `.ics` built by
   `shared/lib/ics` — unit-tested), «لغو نوبت» (confirm → cancel → back).
3. Cancelled appointment shows a muted state and hides cancel.

## Acceptance
- Light/dark + fa/en screenshots of both screens; ics generator tests (Tehran TZ, alarm = remind_before);
  build green.
