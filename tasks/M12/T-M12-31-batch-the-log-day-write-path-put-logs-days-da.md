---
id: T-M12-31
title: Batch the log-day write path (PUT /logs/days/:date) and measure it
milestone: M12
type: backend
status: todo
depends_on: [T-M12-30]
parallel_group: M12-PERF
touches: [backend-go/internal/healthlog, backend-go/internal/http/handlers_logs.go, backend-go/db/queries/healthlog, backend-go/internal/querybudget, backend-go/loadtest/log_burst.js]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && make test-int PKG="./internal/healthlog/... ./internal/http/... ./internal/querybudget/..."
---

# T-M12-31 — Batch the log-day write path (PUT /logs/days/:date) and measure it
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
P-06: a 10-field save takes ~25-40 round trips inside a FOR UPDATE lock, and the load test measures a different (legacy) route.

## Scope
1. One `DELETE … WHERE (category,param) IN (…)` and one multi-row INSERT per save; pass the pre-read day into `SaveDay` instead of reading twice.
2. Same results: existing healthlog tests + a property test (random edits → same final rows as the old path).
3. Query-budget row for the write; `log_burst.js` switched to `PUT /logs/days/:date`; zero stale reads after write (cycle revision bumps once).

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Queries per save ≤ ~6 for a 10-field change; budget and k6 numbers in perf.md.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
