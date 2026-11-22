---
id: T-M12-32
title: Indexes and limits: user_notifications listing, next appointment, assistant history
milestone: M12
type: backend
status: done
depends_on: []
parallel_group: M12-PERF
touches: [backend/database/migrations, backend-go/internal/platform/config/config.go, backend-go/db/migrations, backend-go/db/queries/home, backend-go/db/queries/care, backend-go/db/queries/assistant, backend-go/internal/home, backend-go/internal/care, backend-go/internal/assistant]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && make test-int PKG="./internal/home/... ./internal/care/... ./internal/assistant/..."
---

# T-M12-32 — Indexes and limits: user_notifications listing, next appointment, assistant history
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
P-07/P-08: notifications grow with Web Push and are listed with a filesort + OFFSET; the next-appointment read and assistant history are unbounded.

## Scope
1. goose migration: index `(user_id, created_at, id)` on `user_notifications`; keep the response shape (contract) — add keyset internally only if it does not change the API.
2. `ListAppointmentsFrom`: filter cancelled in SQL + `LIMIT`.
3. Assistant: load only the last N messages that fit the context window (N from config).
4. Int test with 5k notifications: `EXPLAIN` uses the index (no filesort).

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Tests green; EXPLAIN evidence in perf.md.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
