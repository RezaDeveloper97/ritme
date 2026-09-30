---
id: B-N6-10
title: N6 rollout — stage deploy and e2e
milestone: N6
type: release
status: todo
depends_on: [B-N6-09]
parallel_group: N6-J
touches: [docs/qa/bloom,bloom/PROGRESS.md]
skills: [verify-all,deploy-stage]
verify: bash -c 'test -s docs/qa/bloom/n6-stage.md'
---

# B-N6-10 — N6 rollout — stage deploy and e2e

## Why
Ship N6.

## Scope
- Vitals + safety modal, record + share link, lab flow with fake AI, todo; stage only.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- `docs/qa/bloom/n6-stage.md`
- `verify` green
