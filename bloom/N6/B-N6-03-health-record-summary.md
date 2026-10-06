---
id: B-N6-03
title: Health record summary
milestone: N6
type: fullstack
status: todo
depends_on: [B-N6-01]
parallel_group: N6-C
touches: [backend-go/internal/healthrecord,backend-go/db,backend-go/api,frontend/src/screens/health-record,frontend/src/entities/health-record]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N6-03 — Health record summary

## Why
«پرونده سلامت من».

## Design
- `docs/design/night-bloom/c-health-record/nbl_Record_Summary.dc.html` (+ `nbd_Record_Summary`)

## Scope
- Basic info (height, weight, BMI, blood type), conditions, meds (from care), allergies, pregnancies & births, cycle summary, vitals 30d, checkups & labs — each section editable where the data is user-owned.

- From B-N6-01: analyses (`/analysis/summary` vitals card, pregnancy analysis BP/glucose) still read only taxonomy v2 measurements — make them read `vital_readings` merged with log values (same merge rule as `internal/vitals/merge.go`).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
