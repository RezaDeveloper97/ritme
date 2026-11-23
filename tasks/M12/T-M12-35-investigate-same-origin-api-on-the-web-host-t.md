---
id: T-M12-35
title: Investigate same-origin API on the web host to remove CORS preflights
milestone: M12
type: investigate
status: todo
depends_on: [T-M12-02]
parallel_group: M12-PERF
touches: [docs/quality-2026-11/same-origin.md]
skills: []
verify: test -f docs/quality-2026-11/same-origin.md
---

# T-M12-35 — Investigate same-origin API on the web host to remove CORS preflights
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
P-11: about 11 OPTIONS round trips per cold home because the API is cross-origin and the preflight cache is per URL.

## Scope
1. Measure preflights on stage (one origin already) vs beta; write the nginx path-routing design for beta, cookie/CSP/CORS effects, the PWA and android-shell base-URL impact (note only — no Android work), and rollback.
2. Ask the user whether to implement it on beta; if yes, add a follow-up task.

## Out of scope
- Any production change.
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Doc with numbers and a yes/no from the user.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
