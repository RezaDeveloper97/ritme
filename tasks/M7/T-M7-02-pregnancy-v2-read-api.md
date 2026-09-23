---
id: T-M7-02
title: Pregnancy v2 — dating preview, today, week and week-state API
milestone: M7
type: backend
status: todo
depends_on: [T-M7-01]
parallel_group: M7-B
touches: [backend-go/internal/pregnancy/v2, backend-go/internal/http/routes_pregnancy_v2.go, backend-go/db/queries/pregnancy, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/pregnancy/... ./internal/messages/... && make test-int PKG=./internal/pregnancy/...
---

# T-M7-02 — Pregnancy v2 — dating preview, today, week and week-state API

## Scope
1. `routes_pregnancy_v2.go` + package `internal/pregnancy/v2` reusing `pregnancy/calc`.
2. `POST /pregnancy/v2/dating-preview` (no write): weeks+days, due date, usual birth range (±uncertainty), confidence,
   localized basis sentence from `pregnancy_setup` templates.
3. `GET /pregnancy/v2/today`: prev/current/next week summaries (trimester·week label, title, size line), due card
   (date, days left, range), progress % and trimester start dates, next visit (from M3 appointments if present, else
   next care item window), week tip (via the message engine layer from T-M7-04 — until it lands, read
   `pregnancy_week_tip` directly behind an interface), this week's tasks + done state, unread alert count. ≤ 5 queries.
4. `GET /pregnancy/v2/weeks/{n}` (1–42, 404 otherwise) and `PUT /pregnancy/v2/weeks/{n}/state`.
5. Jalali/Gregorian labels by locale; OpenAPI; integration tests (LMP vs ultrasound dating, week 40+ overdue, no
   pregnancy → 409 `pregnancy_not_active`).

## Acceptance
- Shapes documented in OpenAPI and matching docs/pregnancy-v2/README.md; tests green.

## Note from T-M7-08 (frontend contract)
The frontend parsers are in `frontend/src/entities/pregnancy/api/v2-schema.ts` (primary shape + tolerated
alternatives) — match them unless the spec says otherwise, and report differences. Admin text comes back as plain
localized strings; dates `YYYY-MM-DD` with sibling `*_label`; 409 `pregnancy_not_active` / 404 → frontend treats as
"go to Setup". Admin `illustration_key` enum = `FETUS_ILLUSTRATION_KEYS` in
`frontend/src/shared/ui/illustrations/fetus-keys.ts`. Symptom severities `mild|moderate|severe`; mood 1–5.
