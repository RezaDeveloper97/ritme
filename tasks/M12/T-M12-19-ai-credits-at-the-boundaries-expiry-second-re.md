---
id: T-M12-19
title: AI credits at the boundaries: expiry second, refund after expiry, stacked purchase, spend+refund race
milestone: M12
type: backend
status: done
depends_on: [T-M12-11]
parallel_group: M12-TEST
touches: [backend-go/internal/ai/credits, backend-go/internal/plus, backend-go/db/queries/plus, docs/ai-credits.md]
skills: []
verify: cd backend-go && go test -race -count=1 ./internal/ai/credits/... && make test-int PKG="./internal/ai/credits/... ./internal/plus/..."
---

# T-M12-19 — AI credits at the boundaries: expiry second, refund after expiry, stacked purchase, spend+refund race
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-06: the ledger is the money; its edges are untested.

## Scope
1. Spend at exactly `period_end - 1s`, `period_end`, `period_end + 1s` (test clock) — document which side the boundary falls on and assert it.
2. Refund of a spend whose period has since expired: what happens is defined in docs/ai-credits.md (ask the user if not) and asserted.
3. Buying a plan while one is running: period credits granted per the documented rule, never twice.
4. Concurrent spend + refund on the same grant (goroutines, `-race`): balance never negative, total conserved.
5. Fix any bug found in the same task.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- All cases green with `-race`; any rule decided with the user is written into docs/ai-credits.md.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
