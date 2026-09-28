---
id: T-M5-09
title: Fertility rollout — route /api/v1/fertility to Go, verify, stage
milestone: M5
type: release
status: done
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

## Resolved
2026-09-28: the user authorized staging access; stage e2e passed on `stage` @ ea8a1eb (see
docs/fertility-ttc/README.md § Staging e2e). Production only when the user asks.
