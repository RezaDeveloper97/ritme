---
id: B-N7-10
title: N7 rollout — stage deploy and e2e (book, chat, assistant)
milestone: N7
type: release
status: todo
depends_on: [B-N7-08,B-N7-09]
parallel_group: N7-J
touches: [docs/qa/bloom,bloom/PROGRESS.md]
skills: [verify-all,deploy-stage]
verify: bash -c 'test -s docs/qa/bloom/n7-stage.md'
---

# B-N7-10 — N7 rollout — stage deploy and e2e (book, chat, assistant)

## Why
Ship N7.

## Scope
- Book with fake payment, doctor replies from admin, assistant chat → summary → handoff; stage only.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- `docs/qa/bloom/n7-stage.md`
- `verify` green
