---
id: T-M12-14b
title: Account export loader on sqlc with explicit column lists
milestone: M12
type: backend
status: done
depends_on: [T-M12-14,T-M12-19c]
parallel_group: M12-TEST
touches: [backend-go/internal/accountexport,backend-go/db/queries/profile]
skills: []
verify: cd backend-go && CGO_ENABLED=0 make sqlc && make test-int PKG="./internal/accountexport/... ./internal/http/..." RUN=TestAccountExport
---

# T-M12-14b — Account export loader on sqlc with explicit column lists

**Model:** sonnet (opus reviews). Follow-up of T-M12-14.

## Why
T-M12-14 added 96 missing tables to the export through a generic loader in `accountexport/tables.go` that builds
`SELECT * FROM <const table> WHERE <const owner>` strings — constants only, user id bound, but it breaks the
backend rule "sqlc only" and `SELECT *` would export any future sensitive column automatically (hidden columns are a
deny-list by name).

## Scope
1. One sqlc query per exported table in `db/queries/profile/export_*.sql` with an explicit column list (allow-list);
   the loader keeps its section/part layout and decryption.
2. Export output byte-identical for the seeded fixture (compare before/after JSON in the test).
3. Coverage test unchanged and green.

## Acceptance
- No string-built SQL left in `internal/accountexport`; export JSON identical; tests green.
