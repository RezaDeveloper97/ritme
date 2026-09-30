---
id: B-N1-15
title: Restyle reminders (v13) and checkups (v14) screens
milestone: N1
type: frontend
status: done
depends_on: [B-N1-03]
parallel_group: N1-O
touches: [frontend/src/screens/reminders,frontend/src/screens/reminder-medication-form,frontend/src/screens/reminder-appointment-form,frontend/src/screens/reminder-appointment-detail,frontend/src/screens/reminders-add,frontend/src/screens/checkups,frontend/src/screens/checkup-detail,frontend/src/screens/checkup-history,frontend/src/screens/checkup-self-exam,frontend/src/screens/checkup-mark-done,frontend/src/widgets/checkups-card,frontend/src/widgets/today-reminders]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-15 — Restyle reminders (v13) and checkups (v14) screens

## Why
M3/M4 shipped these; re-skin only.

## Design
- `docs/design/night-bloom/c-health-record/nbl_v13_Reminders.dc.html` (+ `nbd_v13_Reminders`)
- `docs/design/night-bloom/c-health-record/nbl_v13_AddChooser.dc.html` (+ `nbd_v13_AddChooser`)
- `docs/design/night-bloom/c-health-record/nbl_v13_AddMedication.dc.html` (+ `nbd_v13_AddMedication`)
- `docs/design/night-bloom/c-health-record/nbl_v13_AddAppointment.dc.html` (+ `nbd_v13_AddAppointment`)
- `docs/design/night-bloom/c-health-record/nbl_v13_AppointmentDetail.dc.html` (+ `nbd_v13_AppointmentDetail`)
- `docs/design/night-bloom/c-health-record/nbl_v14_Checkups.dc.html` (+ `nbd_v14_Checkups`)
- `docs/design/night-bloom/c-health-record/nbl_v14_CheckupDetail.dc.html` (+ `nbd_v14_CheckupDetail`)
- `docs/design/night-bloom/c-health-record/nbl_v14_MarkDone.dc.html` (+ `nbd_v14_MarkDone`)
- `docs/design/night-bloom/c-health-record/nbl_v14_History.dc.html` (+ `nbd_v14_History`)
- `docs/design/night-bloom/c-health-record/nbl_v14_SelfExam.dc.html` (+ `nbd_v14_SelfExam`)
- `docs/design/night-bloom/c-health-record/nbl_v14_Main.dc.html` (+ `nbd_v14_Main`)

## Scope
- Reminders list, add chooser, add medication, add appointment, appointment detail; checkups list, detail, mark done, history, self exam; home cards.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
