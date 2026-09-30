---
id: B-N4-10
title: N4 rollout — stage deploy and two-account e2e
milestone: N4
type: release
status: todo
depends_on: [B-N4-07,B-N4-09]
parallel_group: N4-J
touches: [docs/qa/bloom,bloom/PROGRESS.md]
skills: [verify-all,deploy-stage]
verify: bash -c 'test -s docs/qa/bloom/n4-stage.md'
---

# B-N4-10 — N4 rollout — stage deploy and two-account e2e

## Why
Ship N4.

## Scope
- Woman invites man, man signs up with code, access changes reflect live, revoke; stage only.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- `docs/qa/bloom/n4-stage.md`
- `verify` green
