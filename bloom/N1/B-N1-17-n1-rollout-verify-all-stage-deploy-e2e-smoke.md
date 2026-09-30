---
id: B-N1-17
title: N1 rollout — verify-all, stage deploy, e2e smoke
milestone: N1
type: release
status: todo
depends_on: [B-N1-16]
parallel_group: N1-Q
touches: [docs/qa/bloom,bloom/PROGRESS.md]
skills: [verify-all,deploy-stage]
verify: bash -c 'test -s docs/qa/bloom/n1-stage.md'
---

# B-N1-17 — N1 rollout — verify-all, stage deploy, e2e smoke

## Why
Ship the milestone to staging and prove it end to end.

## Scope
- Run verify-all; commit; push the `stage` branch only if the user agrees; `/deploy-stage`; smoke on https://stage.ritmeapp.ir in light + dark (signup, home 3 states, calendar, log, me, TTC, pregnancy, reminders, checkups); write `docs/qa/bloom/n1-stage.md`. Never production.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Stage smoke doc with screenshots and a pass/fail table
- `verify` green
