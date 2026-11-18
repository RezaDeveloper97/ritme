---
id: T-M12-33
title: Home: fetch the ring months in parallel instead of after /cycle/today
milestone: M12
type: frontend
status: todo
depends_on: [T-M12-28]
parallel_group: M12-PERF
touches: [frontend/src/screens/home/ui/HomePage.tsx, frontend/src/entities/cycle]
skills: [local-dev]
verify: cd frontend && npm run typecheck && npm test
---

# T-M12-33 — Home: fetch the ring months in parallel instead of after /cycle/today
**Model:** sonnet. Evidence: `docs/quality-2026-11/findings.md`.

## Why
P-09: the previous-month request waits for `/cycle/today`, adding one round trip to every cold home on a slow network.

## Scope
1. Prefetch previous, current and next month in parallel at mount (`useQueries`), deduped with the existing keys; drop the derived late fetch.
2. Measure: CDP network waterfall on cold /fa/home before/after (requests count and critical-path depth) in perf.md.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Critical path one step shorter; month requests ≤ 3; ring renders identically.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
