---
id: T-M12-22
title: Push/notice delivery exactly once under concurrent ticks and after a failed send
milestone: M12
type: backend
status: done
depends_on: [T-M12-11]
parallel_group: M12-TEST
touches: [backend-go/internal/care/push_notices_int_test.go, backend-go/internal/care, backend-go/internal/notify]
skills: []
verify: cd backend-go && make test-int PKG="./internal/care/... ./internal/notify/..."
---

# T-M12-22 — Push/notice delivery exactly once under concurrent ticks and after a failed send
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-09: only a single-tick dedupe test exists; two workers or a retry can double-notify (annoying) or drop a dose reminder (harmful).

## Scope
1. Two `TickNotices` in parallel over the same rows → each notice delivered exactly once (fake push transport counts).
2. Transport fails once → retried later and delivered once; permanent failure (410) → subscription removed, no retry storm.
3. Medication reminder + inbox notice for the same event → not duplicated across channels if the design says so.
4. Fix the claim/lock logic if any case fails.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- All cases green with `-race`.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
