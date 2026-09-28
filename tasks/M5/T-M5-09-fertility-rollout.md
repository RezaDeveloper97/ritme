---
id: T-M5-09
title: Fertility rollout — route /api/v1/fertility to Go, verify, stage
milestone: M5
type: release
status: in_progress
depends_on: [T-M5-03, T-M5-05, T-M5-06, T-M5-07, T-M5-08, T-M2-09]
parallel_group: M5-D
touches: [deploy, tasks/PROGRESS.md, docs/fertility-ttc/README.md]
skills: [verify-all, deploy-stage]
verify: bash -c 'curl -fsS -u "$STAGE_AUTH" https://stage.ritmeapp.ir/api/v1/fertility/today -H "Accept: application/json" -o /dev/null -w "%{http_code}" | grep -qE "^(200|401)$"'
---

# T-M5-09 — Fertility rollout — route /api/v1/fertility to Go, verify, stage

## Scope
1. Add `/api/v1/fertility` to the Go route group on stage (needs T-M2-09).
2. `verify-all` green; stage e2e as a `trying` user: tiles → log LH/BBT/intercourse → tiles update → BBT chart over
   several seeded days → insights; light and dark.
3. Production only when the user asks.

## Acceptance
- E2E checklist with screenshots in PROGRESS.

## Blocked
Staging e2e not run: the agent has no access to the server (89.251.8.115 / stage.ritmeapp.ir) — denied by the
user's permission policy (2026-09-26, again 2026-09-27). Local e2e done instead (2026-09-27): verify-all green, API +
UI light/dark e2e passed, see docs/fertility-ttc/README.md § Local e2e. Remaining: stage e2e by the user (or grant
server access), T-M5-10 UI fixes, production when asked.
