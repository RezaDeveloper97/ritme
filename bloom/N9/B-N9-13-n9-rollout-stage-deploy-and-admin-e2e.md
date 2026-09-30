---
id: B-N9-13
title: N9 rollout — stage deploy and admin e2e
milestone: N9
type: release
status: todo
depends_on: [B-N9-12]
parallel_group: N9-M
touches: [docs/qa/bloom,bloom/PROGRESS.md]
skills: [verify-all,deploy-stage]
verify: bash -c 'test -s docs/qa/bloom/n9-stage.md'
---

# B-N9-13 — N9 rollout — stage deploy and admin e2e

## Why
Ship N9.

## Scope
- Each role logs in and sees only its modules; message A/B + approval; simulator; config approval; stage only.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- `docs/qa/bloom/n9-stage.md`
- `verify` green
