---
id: T-M12-19b
title: Lock the user row in challenge rewards and admin period edits (deadlock and overlap)
milestone: M12
type: backend
status: todo
depends_on: [T-M12-19]
parallel_group: M12-TEST
touches: [backend-go/internal/challenges,backend-go/internal/admin,backend-go/db/queries/challenges]
skills: []
verify: cd backend-go && make test-int PKG="./internal/challenges/... ./internal/plus/... ./internal/admin/..."
---

# T-M12-19b — Lock the user row in challenge rewards and admin period edits (deadlock and overlap)

**Model:** opus. Found in T-M12-19 (2026-11-18) after the plus user-row lock landed.

## Why
1. `challenges/service.go` `finish`: `InsertRewardGrant` (FK → S lock on `users`) then `GrantDays` (X lock on the user)
   → two concurrent `CompleteTask` of the same user with Plus-day rewards can deadlock (MariaDB 1213, no retry).
2. Admin period extend/move locks only the subscription row, not the user → can overlap a concurrent purchase.

## Scope
1. `CompleteTask`: take the user-row lock right after `LockUserEnrollment`, before any insert (order: row → user, as
   in plus). Concurrency test: two challenges finishing at once for one user → both rewards granted, no deadlock.
2. Admin period edits: lock the user row first; test admin extend racing a settle → no overlap.

## Acceptance
- Both concurrency tests green 3× natively and under `make test-race-docker`; red without the lock.
