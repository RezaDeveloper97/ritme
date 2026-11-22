---
id: T-M12-23
title: Frontend logic tests: AI gate states and many requests hitting a 401 at once
milestone: M12
type: frontend
status: done
depends_on: [T-M12-12]
parallel_group: M12-TEST
touches: [frontend/src/features/ai-gate, frontend/src/shared/api/apiClient.test.ts, frontend/src/shared/session]
skills: []
verify: cd frontend && npx vitest run src/features/ai-gate src/shared/api src/shared/session
---

# T-M12-23 — Frontend logic tests: AI gate states and many requests hitting a 401 at once
**Model:** sonnet. Evidence: `docs/quality-2026-11/findings.md`.

## Why
T-06/T-11: `features/ai-gate` has no tests though it decides whether a paid action may start; concurrent 401s are untested (double logout, lost redirects).

## Scope
1. AI gate model: no subscription, out of credits, feature off, enough credits, cost changes between render and press → the right sheet/state each time; cost always from the API, never a literal.
2. 401: five parallel requests return 401 → exactly one session teardown and one redirect; a 401 with a non-session `error_code` does not log out (per backend 401 rules).

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Tests green; pure model tests in the node environment (no DOM needed).
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
