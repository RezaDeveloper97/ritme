---
id: T-M4-10
title: Checkups rollout — route checkups and admin catalog to Go, content review, stage
milestone: M4
type: release
status: in_progress
depends_on: [T-M4-02, T-M4-04, T-M4-06, T-M4-07, T-M4-09, T-M2-09]
parallel_group: M4-D
touches: [deploy, tasks/PROGRESS.md, docs/checkups/README.md]
skills: [verify-all, deploy-stage]
verify: bash -c 'curl -fsS -u "$STAGE_AUTH" https://stage.ritmeapp.ir/api/v1/checkups -H "Accept: application/json" -o /dev/null -w "%{http_code}" | grep -qE "^(200|401)$"'
---

# T-M4-10 — Checkups rollout — route checkups and admin catalog to Go, content review, stage

## Why
Go-only feature: reachable only once nginx sends `/api/v1/checkups` (and the admin catalog, with admin-web) to Go.

## Scope
1. Add `/api/v1/checkups` to the Go route group on stage; admin catalog is served by admin-web + Go admin API.
2. The user (or a clinician they name) signs off the seeded catalog copy in admin before production.
3. `verify-all` green; stage e2e: home card → list → detail → mark done (with local attachment) → history → PDF →
   self-exam; admin edit of a type visible in the app; light and dark.
4. Production only when the user asks.

## Acceptance
- Stage e2e checklist with screenshots in PROGRESS; content sign-off recorded.

## Note (frontend foundation)
Copy the new frontend namespace JSON (`frontend/messages/{fa,en}/<ns>.json`) into the backend translation seed
(`backend-go/resources/translations/<code>/`, CLAUDE.md §6.4) if that is still how admin-editable translations are
seeded.
