---
id: T-M7-15
title: Pregnancy v2 rollout — clinical content sign-off, route /pregnancy/v2 to Go, stage e2e
milestone: M7
type: release
status: todo
depends_on: [T-M7-07, T-M7-09, T-M7-10, T-M7-11, T-M7-12, T-M7-13, T-M7-14, T-M2-09]
parallel_group: M7-E
touches: [deploy, tasks/PROGRESS.md, docs/pregnancy-v2/README.md]
skills: [verify-all, deploy-stage]
verify: bash -c 'curl -fsS -u "$STAGE_AUTH" https://stage.ritmeapp.ir/api/v1/pregnancy/v2/today -H "Accept: application/json" -o /dev/null -w "%{http_code}" | grep -qE "^(200|401|409)$"'
---

# T-M7-15 — Pregnancy v2 rollout

## Scope
1. The user (or a named clinician) reviews week details, care plan windows, alert thresholds/texts and tips in admin;
   the reviewer line is filled.
2. Route `/api/v1/pregnancy/v2/` (and the new admin endpoints) to Go on stage.
3. `verify-all`; stage e2e: signup as pregnant → setup → today → week → log (offline too) → alert fires → calendar
   with a booked visit → PDF → profile switch back to cycle; light and dark.
4. Production only when the user asks.

## Acceptance
- Sign-off recorded; e2e checklist with screenshots in PROGRESS.
