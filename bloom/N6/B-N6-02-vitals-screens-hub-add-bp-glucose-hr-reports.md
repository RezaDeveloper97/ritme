---
id: B-N6-02
title: Vitals screens — hub, add BP/glucose/HR, reports
milestone: N6
type: frontend
status: todo
depends_on: [B-N6-01]
parallel_group: N6-B
touches: [frontend/src/screens/vitals*,frontend/src/entities/vitals,frontend/src/features/add-vital]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N6-02 — Vitals screens — hub, add BP/glucose/HR, reports

## Why
Section ج.

## Design
- `docs/design/night-bloom/c-health-record/nbl_Vitals_Hub.dc.html` (+ `nbd_Vitals_Hub`)
- `docs/design/night-bloom/c-health-record/nbl_Vitals_AddBP.dc.html` (+ `nbd_Vitals_AddBP`)
- `docs/design/night-bloom/c-health-record/nbl_Vitals_AddGlucose.dc.html` (+ `nbd_Vitals_AddGlucose`)
- `docs/design/night-bloom/c-health-record/nbl_Vitals_AddHR.dc.html` (+ `nbd_Vitals_AddHR`)
- `docs/design/night-bloom/c-health-record/nbl_Vitals_BPReport.dc.html` (+ `nbd_Vitals_BPReport`)
- `docs/design/night-bloom/c-health-record/nbl_Vitals_GlucoseReport.dc.html` (+ `nbd_Vitals_GlucoseReport`)

## Scope
- Six screens per artboards; unit toggle for glucose.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
