---
id: T-M3-09
title: Care reminders rollout — route /api/v1/care to Go, verify, stage
milestone: M3
type: release
status: todo
depends_on: [T-M3-03, T-M3-06, T-M3-07, T-M3-08, T-M2-09]
parallel_group: M3-D
touches: [deploy, tasks/PROGRESS.md, docs/care-reminders/README.md]
skills: [verify-all, deploy-stage]
verify: bash -c 'curl -fsS -u "$STAGE_AUTH" https://stage.ritmeapp.ir/api/v1/care/enums -H "Accept: application/json" -o /dev/null -w "%{http_code}" | grep -qE "^(200|401)$"'
---

# T-M3-09 — Care reminders rollout — route /api/v1/care to Go, verify, stage

## Why
The feature is Go-only; clients reach it only once nginx sends `/api/v1/care/` to the Go container. Depends on the
strangler infrastructure (T-M2-09, currently blocked).

## Scope
1. Add `/api/v1/care/` to the Go route group in `deploy/go-routes.inc` (stage first); confirm the Laravel twin
   migration ran so `reminder_intakes` exists.
2. `verify-all` green; end-to-end on stage: add medication → appears on both homes → tick dose → add appointment →
   detail → cancel; light and dark.
3. Production only when the user asks (same route line on prod).

## Acceptance
- Stage serves `/api/v1/care/*` from Go; e2e checklist passed with screenshots; PROGRESS updated.
