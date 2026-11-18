---
id: T-M12-29
title: Trim the home i18n payload (sub-key scopes or lazy namespaces)
milestone: M12
type: frontend
status: todo
depends_on: [T-M12-12, T-M12-28]
parallel_group: M12-PERF
touches: [frontend/src/app/message-scopes.ts, frontend/src/app/message-scopes.test.ts, frontend/src/shared/i18n, frontend/src/screens/home]
skills: []
verify: cd frontend && npm test && npm run build && wc -c .next/server/app/fa/home.html
---

# T-M12-29 — Trim the home i18n payload (sub-key scopes or lazy namespaces)
**Model:** sonnet. Evidence: `docs/quality-2026-11/findings.md`.

## Why
P-04: home serialises 146 KB of messages into HTML and RSC on every navigation.

## Scope
1. Support sub-key scopes (e.g. `care.today.*`) in `RouteMessages`, or let mode-specific widgets load their namespace on mount.
2. Keep the message-scopes test as the safety net: every key a home component uses must be in its scope.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Home messages payload at least halved (before/after in perf.md); no raw keys on home in any mode.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
