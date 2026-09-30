---
id: B-N8-10
title: N8 rollout — stage deploy incl. instructor.ritmeapp.ir, e2e
milestone: N8
type: release
status: todo
depends_on: [B-N8-08,B-N8-09]
parallel_group: N8-J
touches: [docs/qa/bloom,bloom/PROGRESS.md]
skills: [verify-all,deploy-stage]
verify: bash -c 'test -s docs/qa/bloom/n8-stage.md'
---

# B-N8-10 — N8 rollout — stage deploy incl. instructor.ritmeapp.ir, e2e

## Why
Ship N8.

## Scope
- Instructor uploads, grants a not-yet-registered phone, that phone signs up and sees the course, offline download; stage only (DNS for instructor.ritmeapp.ir must exist — ask the user).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- `docs/qa/bloom/n8-stage.md`
- `verify` green
