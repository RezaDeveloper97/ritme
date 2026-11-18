---
id: T-M12-16
title: Routine flow test: buy plan → AI spend → balance → period end → spend refused → credit pack
milestone: M12
type: backend
status: todo
depends_on: [T-M12-11]
parallel_group: M12-TEST
touches: [backend-go/internal/http/routine_credits_int_test.go]
skills: []
verify: cd backend-go && make test-int PKG="./internal/http/..." RUN=TestRoutineCredits
---

# T-M12-16 — Routine flow test: buy plan → AI spend → balance → period end → spend refused → credit pack
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-05/T-06: credits rules (docs/ai-credits.md) are tested per function; the whole money path is not.

## Scope
1. Fake gateway (testing env only): checkout → verify → subscription active with period credits.
2. Spend through a real AI-gated route with the fake AI provider → balance drops by the admin-configured cost (read from DB, never hard-coded).
3. Buy an extra pack → period credits are spent first, extras after.
4. Move the test clock past the period end → period credits gone, extras kept but locked → gated route answers the managed "no subscription" error, provider never called.
5. Renew → extras usable again. Replaying the bank callback changes nothing.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Flow test green; asserts ledger rows, not only HTTP codes.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
