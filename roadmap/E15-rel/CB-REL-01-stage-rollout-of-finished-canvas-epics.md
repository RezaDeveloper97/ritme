---
id: CB-REL-01
title: Stage rollout of finished canvas epics
epic: REL
type: release
status: todo
depends_on: [CB-NAV-04]
parallel_group: REL-A
touches: [docs/qa/canvas/stage.md]
skills: [deploy-stage, verify-all]
boards: []
verify: test -s docs/qa/canvas/stage.md
---

# CB-REL-01 — Stage rollout of finished canvas epics

## Why
Ship finished epics to stage. **Only when the user asks in this session** (skill §7).

## Scope
1. verify-all → deploy-stage → per finished epic a stage smoke → docs/qa/canvas/stage.md. Server env the user sets first: NEXT_PUBLIC_NESHAN_KEY, file-storage key/path, AI provider config (bloom adapter) — never committed.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Stage smoke table all ✔.
- `verify` green.
