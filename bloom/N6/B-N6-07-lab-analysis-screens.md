---
id: B-N6-07
title: Lab analysis screens
milestone: N6
type: frontend
status: in_progress
depends_on: [B-N6-06,B-N6-06b]
parallel_group: N6-G
touches: [frontend/src/screens/lab-*,frontend/src/entities/lab,frontend/src/features/upload-lab]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N6-07 — Lab analysis screens

## Why
Seven lab screens.

## Design
- `docs/design/night-bloom/c-health-record/nbl_Lab_Intro.dc.html` (+ `nbd_Lab_Intro`)
- `docs/design/night-bloom/c-health-record/nbl_Lab_Consent.dc.html` (+ `nbd_Lab_Consent`)
- `docs/design/night-bloom/c-health-record/nbl_Lab_Upload.dc.html` (+ `nbd_Lab_Upload`)
- `docs/design/night-bloom/c-health-record/nbl_Lab_Processing.dc.html` (+ `nbd_Lab_Processing`)
- `docs/design/night-bloom/c-health-record/nbl_Lab_Verify.dc.html` (+ `nbd_Lab_Verify`)
- `docs/design/night-bloom/c-health-record/nbl_Lab_Result.dc.html` (+ `nbd_Lab_Result`)
- `docs/design/night-bloom/c-health-record/nbl_Lab_Marker.dc.html` (+ `nbd_Lab_Marker`)

## Scope
- Intro (history), Consent, Upload (camera/gallery/PDF, pages, type, date, fasting), Processing (progress, leave & notify), Verify, Result (flags, summary, doctor questions, add to record, ask assistant → B-N7-07), Marker (trend, explanation).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
