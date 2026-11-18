---
id: T-M12-22b
title: Durable push retry: release the claim in the DB after a passing failure
milestone: M12
type: backend
status: todo
depends_on: [T-M12-22,T-M12-19c]
parallel_group: M12-TEST
touches: [backend-go/db/queries/care,backend-go/internal/care,docs/care-reminders/README.md]
skills: []
verify: cd backend-go && make test-int PKG=./internal/care/...
---

# T-M12-22b — Durable push retry: release the claim in the DB after a passing failure

**Model:** opus. Follow-up of T-M12-22.

## Why
T-M12-22 fixed lost reminders after a passing push failure (timeout/429/5xx) with an in-memory retry queue in the
process that holds the claim. A restart/deploy inside the retry window still loses the dose or notice.

## Scope
1. sqlc queries in `db/queries/care/push.sql` to release a claim (`DELETE` the `reminder_pushes` row / set
   `pushed_at = NULL`) with an attempt counter, so any process retries it on a later tick; keep the 3-attempt cap and
   the lookback windows.
2. Test: claim → passing failure → new Pusher instance (restart) delivers once; permanent failure never retried.
3. Document the retry rules in `docs/care-reminders/README.md` (Web Push section).

## Acceptance
- Restart test green, red on T-M12-22 code; exactly-once tests still green with -race.
