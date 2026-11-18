---
id: T-M12-08
title: Autopilot: sharing, invite, consent and export controls require confirmation
milestone: M12
type: frontend
status: todo
depends_on: []
parallel_group: M12-SEC
touches: [frontend/src/shared/autopilot/config.ts, frontend/src/app/autopilot/catalog.test.ts, docs/autopilot.md]
skills: []
verify: cd frontend && npx vitest run src/app/autopilot src/shared/autopilot
---

# T-M12-08 — Autopilot: sharing, invite, consent and export controls require confirmation
**Model:** sonnet. Evidence: `docs/quality-2026-11/findings.md`.

## Why
S-14: controls that share health data with other people are drivable by a script without the user's «ادامه». Not exploitable today, but the next model-script caller would make it so.

## Scope
1. Add `invite.create`, `invite.consent`, `companion.join-submit`, `teen.share:*`, `report.send`, `privacy.export` to `safety.confirmTargets` (or `blockedTargets` where a script should never reach them).
2. Catalog test: any id matching `/share|invite|consent|send|export|join|delete|pay/` must be in `confirmTargets` or `blockedTargets`, so new controls cannot slip through.

## Out of scope
- —
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Test fails if one of these ids is removed from the lists; `docs/autopilot.md` safety section updated.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
