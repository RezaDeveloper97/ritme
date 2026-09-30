---
id: B-N2-11
title: N2 rollout — stage deploy and e2e (signup branches, purchase, trial)
milestone: N2
type: release
status: todo
depends_on: [B-N2-09,B-N2-10]
parallel_group: N2-K
touches: [docs/qa/bloom,bloom/PROGRESS.md]
skills: [verify-all,deploy-stage]
verify: bash -c 'test -s docs/qa/bloom/n2-stage.md'
---

# B-N2-11 — N2 rollout — stage deploy and e2e (signup branches, purchase, trial)

## Why
Ship N2 to staging.

## Scope
- Signup every goal branch, mode switch, fake-gateway purchase, trial start/expiry via test clock, admin plans; stage only.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- `docs/qa/bloom/n2-stage.md`
- `verify` green
