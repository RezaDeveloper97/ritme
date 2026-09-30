---
id: B-N4-06
title: «ثبت برای چه کسی؟» in medication and appointment forms
milestone: N4
type: frontend
status: todo
depends_on: [B-N4-05]
parallel_group: N4-F
touches: [frontend/src/screens/reminder-medication-form,frontend/src/screens/reminder-appointment-form,frontend/src/features/manage-medication,frontend/src/features/manage-appointment]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N4-06 — «ثبت برای چه کسی؟» in medication and appointment forms

## Why
Companions with edit access can log for the owner.

## Design
- `docs/design/night-bloom/companion-family/nbl_Hamdam_RecordFor.dc.html` (+ `nbd_Hamdam_RecordFor`)

## Scope
- Target picker (self / owner) shown only when edit is granted.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
