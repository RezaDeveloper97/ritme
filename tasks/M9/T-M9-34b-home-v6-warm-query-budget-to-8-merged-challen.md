---
id: T-M9-34b
title: Home v6 warm query budget to 8: merged challenge-completion and reminder queries
milestone: M9
type: backend
status: done
depends_on: [T-M9-34]
parallel_group: M9-C
touches: [backend-go/db/queries/home,backend-go/internal/home,backend-go/internal/querybudget/budgets.go]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/home/... && make test-int PKG="./internal/home/... ./internal/querybudget/..." && make contract ROUTES=home && make contract-v6
---

# T-M9-34b — Home v6 warm query budget to 8: merged challenge-completion and reminder queries
**Model:** opus.

## Why
T-M9-34 acceptance asked for warm `/home` ≤ 8 queries under v6; it measures 9 (querybudget harness; v1 is 11). The
remaining split reads are challenge completions (14-day window + today = 2 queries) and the doctor + medication
reminders (2 queries).

## Scope
1. One sqlc query for challenge completions (window incl. today) in `db/queries/home`.
2. One UNION (or single) query for the doctor and medication reminders.
3. Use them on the v6 path only (v1 byte-identical); lower the `home v6` row in `internal/querybudget/budgets.go` to warm 8.

## Out of scope
- v1 path changes; other routes.

## Acceptance
- Warm `/home` v6 ≤ 8 queries in `TestQueryBudgets`; contract `home` and `contract-v6` unchanged.
