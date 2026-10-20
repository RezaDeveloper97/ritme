---
id: T-M9-50
title: Beta rollout with backup and user OK
milestone: M9
type: investigate
status: todo
depends_on: [T-M9-48, T-M9-49]
parallel_group: M9-R
touches: [docs/engine-v6/rollout.md]
skills: [deploy-beta]
verify: grep -q 'Beta: v6' docs/engine-v6/rollout.md
---

# T-M9-50 — Beta rollout with backup and user OK
**Model:** main session only. Beta has real users.

## Why
Beta is the first place real users see v6 (X-03). It must be reversible and approved.

## Scope
1. Ask the user (Persian) for the go-ahead, showing the stage results (diff classes, screenshots, perf table) and the
   visible changes (I-01 onboarding range, E-04 lateness, E-13 pill, E-15 skip).
2. Back up the beta database (the `deploy-beta` skill procedure; keep the dump outside the volumes); never wipe volumes.
3. Deploy beta with `CYCLE_ENGINE=shadow` for ≥ 48 h; report diff classes on real data to the user; then switch to
   `v6` only after a second OK.
4. Watch for 48 h: 5xx rate, `engine_cache_verify` (sampled), support reports; rollback plan = set `CYCLE_ENGINE=v1`
   (data migrations are compatible with v1).
5. Record "Beta: v6 since <date>" in `rollout.md`.

## Out of scope
- Production (never — T-M2-26/27 stay blocked).

## Acceptance
- Two user approvals recorded; backup path recorded; beta on v6 with no open incident.
