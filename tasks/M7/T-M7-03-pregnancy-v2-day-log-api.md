---
id: T-M7-03
title: Pregnancy v2 — day log API (mood, symptoms, water, weight, visit note)
milestone: M7
type: backend
status: done
depends_on: [T-M7-01]
parallel_group: M7-B
touches: [backend-go/internal/pregnancy/v2/daylog, backend-go/internal/http/routes_pregnancy_v2.go, backend-go/db/queries/pregnancy, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/pregnancy/... ./internal/messages/... && make test-int PKG=./internal/pregnancy/...
---

# T-M7-03 — Pregnancy v2 — day log API

## Scope
1. `GET/PUT /pregnancy/v2/days/{date}`: symptoms map `{nausea|vomiting|fatigue|headache|back_pain|breast_pain|
   heartburn|constipation|spotting: mild|moderate|severe}` — the first seven + spotting write the existing
   `pregnancy_symptom_logs` columns through the v1 service (same alert side effects), heartburn/constipation/mood/
   water/visit_note go to `pregnancy_daily_extras`, weight goes to the `pregnancy_weekly_logs` row of that week.
2. Response: merged day + last weight (value, date) + alerts raised by this save.
3. Explicit null clears; not in the future; 422 fa/en. Idempotent PUT (offline outbox may resend).
4. `GET /pregnancy/v2/report?from=&to=` — timeline data for the doctor PDF.
5. Integration tests incl. spotting → v1 critical alert still fires.

## Acceptance
- v1 `GET /pregnancy/symptoms/{date}` shows what v2 wrote; tests green.

## Note from T-M7-08 (frontend contract)
The frontend parsers are in `frontend/src/entities/pregnancy/api/v2-schema.ts` (primary shape + tolerated
alternatives) — match them unless the spec says otherwise, and report differences. Admin text comes back as plain
localized strings; dates `YYYY-MM-DD` with sibling `*_label`; 409 `pregnancy_not_active` / 404 → frontend treats as
"go to Setup". Admin `illustration_key` enum = `FETUS_ILLUSTRATION_KEYS` in
`frontend/src/shared/ui/illustrations/fetus-keys.ts`. Symptom severities `mild|moderate|severe`; mood 1–5.
