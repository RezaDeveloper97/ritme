---
id: T-M12-14
title: Account export covers every user table (schema-driven test)
milestone: M12
type: backend
status: done
depends_on: [T-M12-13]
parallel_group: M12-TEST
touches: [backend-go/internal/accountexport, backend-go/internal/http/account_export_int_test.go, docs/quality-2026-11/user-tables.md]
skills: []
verify: cd backend-go && make test-int PKG="./internal/accountexport/... ./internal/http/..."
---

# T-M12-14 — Account export covers every user table (schema-driven test)
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-04: nothing checks the export against the list of user tables, so new features silently drop out of "download my data".

## Scope
1. Reuse the T-M12-13 table list: each user table is either in the export or on an export-exclusion list with a reason (e.g. tokens, derived caches).
2. Behaviour test: seed every table, export, assert each included table's row appears, and the second user's rows never appear.
3. Add the missing sections to the export.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Test green; the exclusion list is documented in `user-tables.md`.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
