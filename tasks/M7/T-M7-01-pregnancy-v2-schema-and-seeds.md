---
id: T-M7-01
title: Pregnancy v2 — new tables and seeds (week details, care plan, daily extras, week state, messages)
milestone: M7
type: backend
status: done
depends_on: []
parallel_group: M7-A
touches: [backend-go/db/migrations, backend-go/db/queries/pregnancy, backend-go/sqlc.yaml, backend/database/migrations, docs/go-migration/deviations.md, docs/pregnancy-v2/README.md]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/pregnancy/... ./internal/messages/... && make schema-diff
---

# T-M7-01 — Pregnancy v2 — new tables and seeds

## Why
Everything in the v2 module that the admin must define needs structured storage (docs/pregnancy-v2/README.md § Data, § What the admin
defines). v1 tables stay untouched so the Laravel contract of `/pregnancy/*` does not change.

## Scope
1. Goose migration + schema-only Laravel twin for `pregnancy_week_details`, `pregnancy_care_items`,
   `pregnancy_daily_extras`, `pregnancy_week_user_state` (next free goose number).
2. Seeds in **both** migrations (schema-diff compares row counts): weeks 1–42 details (fill from existing
   `pregnancy_weekly_content` where possible + artboard copy for week 8, rest marked `[needs review]`), the 5
   care items of the Calendar artboard, `message_contents` rows for groups `pregnancy_week_tip` (1–42, fa+en),
   `pregnancy_alert` (one per rule in T-M7-04 with default params + texts) and `pregnancy_setup`.
3. sqlc queries for the new tables.
4. `deviations.md`: `/api/v1/pregnancy/v2/*` is a Go-only addition (M7).

## Acceptance
- `make schema-diff` exit 0; sqlc generates; PROGRESS lists the seed copy as needing clinical review.
