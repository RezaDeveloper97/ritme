---
id: T-M7-13
title: Frontend — pregnancy «تقویم و ویزیت‌ها» (Calendar) v2 + doctor PDF
milestone: M7
type: frontend
status: todo
depends_on: [T-M7-08, T-M7-05, T-M3-08]
parallel_group: M7-D
touches: [frontend/src/screens/pregnancy-calendar, frontend/src/app/[locale]/pregnancy/calendar, frontend/src/shared/lib/pdf]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M7-13 — Frontend — pregnancy Calendar v2

## Scope
`/pregnancy/calendar` from `Calendar.dc.html`: month header with pregnancy-week range, month grid (visit dots,
week-start markers, today), legend, selected-day panel (empty text + add); next-visit card with 3-stage stepper
(tap advances stage via M3 appointment PUT; «نتیجه ثبت شد» asks for a result note), prep text, reminder, «مسیریابی»;
care plan list (done / booked / «رزرو» → M3 AddAppointment prefilled with care_item_key, title, suggested date);
source note; «گزارش علائم برای پزشک (PDF)» from `/pregnancy/v2/report` via `shared/lib/pdf` (reuse M4's generator if
present); «ویزیت جدید» → M3 AddAppointment.

## Acceptance
- Light/dark × fa/en screenshots; PDF renders Persian correctly; build green.
