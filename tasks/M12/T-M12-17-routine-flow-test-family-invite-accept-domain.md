---
id: T-M12-17
title: Routine flow test: family invite → accept → domain-limited reads → revoke → 403
milestone: M12
type: backend
status: done
depends_on: [T-M12-11]
parallel_group: M12-TEST
touches: [backend-go/internal/http/routine_family_int_test.go]
skills: []
verify: cd backend-go && make test-int PKG="./internal/http/..." RUN=TestRoutineFamily
---

# T-M12-17 — Routine flow test: family invite → accept → domain-limited reads → revoke → 403
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-05: the authorizer is well tested in isolation; the path through invites, projections and revocation as a member actually experiences it is not.

## Scope
1. Owner invites with one domain at view level → member accepts → member reads the granted domain (200) and every other domain route (403/404 per contract), and cannot write.
2. Owner raises the level → write allowed; owner revokes → the very next member request is 403 (no stale cache); member's own data untouched.
3. Mode switch (e.g. pregnancy) changes the epoch → member view follows the decided rule.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Flow test green across every domain in a table.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
