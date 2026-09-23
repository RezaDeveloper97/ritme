---
id: T-M4-01
title: Checkups — schema, default catalog seed and status engine (Go)
milestone: M4
type: backend
status: done
depends_on: []
parallel_group: M4-A
touches: [backend-go/db/migrations, backend-go/db/queries/checkups, backend-go/sqlc.yaml, backend-go/internal/checkups, backend/database/migrations, docs/go-migration/deviations.md, docs/checkups/README.md]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/checkups/... && make test-int PKG=./internal/checkups/... && make schema-diff
---

# T-M4-01 — Checkups — schema, default catalog seed and status engine (Go)

## Why
The periodic-checkups feature (docs/checkups/README.md, artboards `docs/design/checkups-v14/`) needs an
admin-editable catalog, per-user records and a deterministic status engine. Go-only, like M3.

## Scope
1. Goose migration + schema-only Laravel twin for `checkup_types`, `checkup_records`, `user_checkup_settings`
   exactly as in the README. If M3's migration exists, take the next goose number.
2. Seed the six default types (bilingual title/subtitle/why/prep/guide/findings from the artboards' copy) with
   `INSERT IGNORE` on `key` in **both** migrations so `make schema-diff` row counts match.
3. `internal/checkups/engine`: pure functions over (types, records, settings, birthday, cycle predictions, clock) →
   items with status/section/next_due + summary; cycle-timed next-due uses the cycle engine's predicted period
   starts (read-only reuse of `internal/cycle`), falling back to a monthly date when no cycle data exists.
4. Table-driven unit tests: never recorded, overdue by months, due inside the lead window, age `not_yet` (start year
   label), range interval (mammography), user `next_due_on` override, disabled setting, pregnancy-hidden type.
5. `deviations.md`: `/api/v1/checkups*` and `/api/admin/v1/checkup-types*` are Go-only additions (M4).

## Out of scope
HTTP handlers (T-M4-02, T-M4-03), push notifications.

## Acceptance
- `make schema-diff` exit 0; engine coverage ≥ 90 %; tests green.
- PROGRESS lists the seed copy as **needing medical review** before production (open item for the user).
