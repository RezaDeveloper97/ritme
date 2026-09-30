---
id: B-N3-14
title: N3 rollout — stage deploy and e2e
milestone: N3
type: release
status: todo
depends_on: [B-N3-13]
parallel_group: N3-N
touches: [docs/qa/bloom,bloom/PROGRESS.md]
skills: [verify-all,deploy-stage]
verify: bash -c 'test -s docs/qa/bloom/n3-stage.md'
---

# B-N3-14 — N3 rollout — stage deploy and e2e

## Why
Ship N3.

## Scope
- Log a week of data per mode, voice log (fake provider), analysis screens; stage only.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- `docs/qa/bloom/n3-stage.md`
- `verify` green
