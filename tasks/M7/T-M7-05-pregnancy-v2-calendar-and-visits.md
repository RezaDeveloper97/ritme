---
id: T-M7-05
title: Pregnancy v2 — calendar, care plan and visit stages (on top of M3 appointments)
milestone: M7
type: backend
status: todo
depends_on: [T-M7-02, T-M3-02]
parallel_group: M7-C
touches: [backend-go/internal/pregnancy/v2/calendar, backend-go/internal/care, backend-go/internal/http/routes_pregnancy_v2.go, backend-go/db/queries/pregnancy, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/pregnancy/... ./internal/messages/... && go test ./internal/care/... && make test-int PKG=./internal/pregnancy/...
---

# T-M7-05 — Pregnancy v2 — calendar, care plan and visit stages

## Scope
1. M3 appointment meta gains optional `care_item_key` and `stage` (booked|done|result) + `result_note`
   (validated, backwards compatible).
2. `GET /pregnancy/v2/calendar?month=YYYY-MM` (Jalali month for fa): day markers (visit, week start, today), visits of
   the month, selected-day details payload, care plan items with window (week_from–week_to → dates from due date),
   state (done / booked / to_book) and suggested date.
3. Tests: item state from linked appointments, window dates across LMP vs ultrasound dating, month boundaries.

## Acceptance
- Calendar artboard data fully covered; tests green.

## Note from T-M7-08 (frontend contract)
The frontend parsers are in `frontend/src/entities/pregnancy/api/v2-schema.ts` (primary shape + tolerated
alternatives) — match them unless the spec says otherwise, and report differences. Admin text comes back as plain
localized strings; dates `YYYY-MM-DD` with sibling `*_label`; 409 `pregnancy_not_active` / 404 → frontend treats as
"go to Setup". Admin `illustration_key` enum = `FETUS_ILLUSTRATION_KEYS` in
`frontend/src/shared/ui/illustrations/fetus-keys.ts`. Symptom severities `mild|moderate|severe`; mood 1–5.
