---
id: B-N7-05
title: Doctor screens — list, profile & slots, review & book, booked, chat
milestone: N7
type: frontend
status: todo
depends_on: [B-N7-02,B-N7-03,B-N7-04]
parallel_group: N7-E
touches: [frontend/src/screens/doctor*,frontend/src/entities/doctor,frontend/src/features/book-visit]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N7-05 — Doctor screens — list, profile & slots, review & book, booked, chat

## Why
Section د user side.

## Design
- `docs/design/night-bloom/d-doctor-assistant/nbl_v17_Doctors.dc.html` (+ `nbd_v17_Doctors`)
- `docs/design/night-bloom/d-doctor-assistant/nbl_v17_DoctorProfile.dc.html` (+ `nbd_v17_DoctorProfile`)
- `docs/design/night-bloom/d-doctor-assistant/nbl_v17_ReviewBook.dc.html` (+ `nbd_v17_ReviewBook`)
- `docs/design/night-bloom/d-doctor-assistant/nbl_v17_Booked.dc.html` (+ `nbd_v17_Booked`)
- `docs/design/night-bloom/d-doctor-assistant/nbl_v17_Chat.dc.html` (+ `nbd_v17_Chat`)

## Scope
- Five screens per artboards.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
