---
id: B-N5-11
title: N5 rollout — stage deploy and e2e (pregnancy → postpartum → child)
milestone: N5
type: release
status: todo
depends_on: [B-N5-09,B-N5-10]
parallel_group: N5-K
touches: [docs/qa/bloom,bloom/PROGRESS.md]
skills: [verify-all,deploy-stage]
verify: bash -c 'test -s docs/qa/bloom/n5-stage.md'
---

# B-N5-11 — N5 rollout — stage deploy and e2e (pregnancy → postpartum → child)

## Why
Ship N5.

## Scope
- Pregnancy → birth → postpartum home, add child, growth/vaccine, spouse sees shared child; stage only.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- `docs/qa/bloom/n5-stage.md`
- `verify` green
