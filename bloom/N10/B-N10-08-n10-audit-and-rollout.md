---
id: B-N10-08
title: N10 audit and rollout
milestone: N10
type: release
status: todo
depends_on: [B-N10-02,B-N10-03,B-N10-04,B-N10-05,B-N10-06,B-N10-07]
parallel_group: N10-H
touches: [docs/night-bloom/audit-n10.md,docs/qa/bloom,bloom/PROGRESS.md]
skills: [verify-all,deploy-stage]
verify: bash -c 'test -s docs/qa/bloom/n10-stage.md'
---

# B-N10-08 — N10 audit and rollout

## Why
Finish the queue.

## Scope
- Fidelity audit + fixes, stage deploy, e2e, final summary of the whole Night & Bloom programme; production cutover is a separate decision for the user.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- `docs/qa/bloom/n10-stage.md`
- `verify` green
