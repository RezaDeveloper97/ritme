---
id: T-M7-13b
title: Frontend — appointment form prefill (title, date, care_item_key) and stage fields
milestone: M7
type: frontend
status: done
depends_on: [T-M7-13,T-M4-08]
parallel_group: M7-D
touches: [frontend/src/screens/reminder-appointment-form,frontend/src/features/manage-appointment,frontend/src/app/[locale]/reminders/appointment/new/page.tsx,frontend/src/screens/pregnancy-calendar]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# T-M7-13b — Frontend — appointment form prefill (title, date, care_item_key) and stage fields

## Why
Several screens link to `/reminders/appointment/new?kind=&title=&date=&care_item_key=` (pregnancy calendar «رزرو», checkups card/detail «ثبت نوبت») but the form reads only `kind`, and `features/manage-appointment` drops `care_item_key`/`stage`/`result_note` — so care-plan items never get linked (T-M7-05 state stays `to_book`).

## Scope
- Route + `AppointmentFormPage` read `title`, `date` (YYYY-MM-DD, not in the past), `care_item_key` from the query and prefill.
- `toAppointmentBody` / appointment schema carry `care_item_key`, `stage`, `result_note`; edit keeps them.
- Pregnancy calendar stage stepper switches from direct `apiClient.put` to `useUpdateAppointment`.
- Import remind-before values from `entities/care-reminder` in the calendar screen.

## Out of scope
- Backend changes (fields already exist, T-M7-05).

## Acceptance
- Booking from the pregnancy care plan creates an appointment with `care_item_key`; calendar shows the item as `booked`.
- Unit tests for query-prefill parsing and body mapping; build green. 
