---
id: T-M10-19b
title: Vitals follow-ups: alert copy placeholders in message_contents, add-form preview, targets contract, goldens
milestone: M10
type: fullstack
status: done
depends_on: [T-M10-19,T-M10-21]
parallel_group: M10-C
touches: [backend-go/db/migrations,backend/database/migrations,backend-go/contract/fixtures/dump.sql,backend-go/contract/cases/vitals.yaml,backend-go/contract/golden/vitals,backend-go/contract/allowlist/vitals.yaml,backend-go/api/openapi.yaml,backend-go/internal/vitals,frontend/src/features/add-vital,docs/engine-v6/drafts/T-M10-19b-vitals-alert-placeholders.sql]
skills: []
verify: cd backend-go && make schema-diff && go vet ./... && go test ./... && make test-int PKG='./internal/vitals/... ./db/migrations/...' && make contract ROUTES=vitals && cd ../frontend && npm run typecheck && npm run lint && npm run test
---

# T-M10-19b — Vitals follow-ups: alert copy placeholders in message_contents, add-form preview, targets contract, goldens
**Model:** opus. D-170.

## Why
These T-M10-19 items lived outside its touches.

## Needs a decision first
The `vitals_alert` copy is safety content (D-90/D-160). Ask the user whether changing the seeded wording to placeholders needs the content approval flow.

## Scope
1. A goose data migration plus Laravel twin turns the seeded `message_contents` `vitals_alert` rows (goose 00035: «۱۸۰/۱۲۰», «۵۴») into `{systolic}/{diastolic}/{mgdl}` placeholders.
   - Only rows that still hold the seed text are changed; Down restores them.
   - Draft: `docs/engine-v6/drafts/T-M10-19b-vitals-alert-placeholders.sql`.
   - Extend the copy test to check the DB rows, and update `dump.sql`.
2. `frontend/src/features/add-vital/ui/AddVitalForm.tsx`: the glucose preview passes `method` and the user's diabetic targets, so the preview matches the server class.
3. Contract cases for `GET/PUT/DELETE /vitals/targets` in `contract/cases/vitals.yaml`, recorded from Go (Go-only route).
4. Re-record the vitals goldens that now carry `status`, `readings_needed` and hub `reports`. Then make those fields `required` in `openapi.yaml` (they were left optional, marked `T-M10-19b`).

## Acceptance
- No seeded vitals alert text holds a hard-coded threshold.
- The add-form preview equals the saved reading's class for diabetic and spec=1 users.
- verify green.

## Decision
User (2026-10-23): move the vitals alert copy into `message_contents` placeholders as-is, without the D-90/D-160 safety-content approval flow; any later wording change goes through that flow.
