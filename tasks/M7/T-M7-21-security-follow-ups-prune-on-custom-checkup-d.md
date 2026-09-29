---
id: T-M7-21
title: Security follow-ups — prune on custom-checkup delete, show cap messages
milestone: M7
type: frontend
status: in_progress
depends_on: [T-M7-19]
parallel_group: M7-E
touches: [frontend/src/features/manage-custom-checkup,frontend/src/features/manage-medication,frontend/src/features/manage-appointment,frontend/src/screens/reminder-medication-form,frontend/src/screens/reminder-appointment-form,frontend/src/screens/checkup-custom-form,frontend/src/shared/api,docs/security]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run test
---

# T-M7-21 — Security follow-ups — prune on custom-checkup delete, show cap messages

## Why
Open items 2 and 3 of T-M7-19 (docs/security/audit-m3-m7.md, tasks/PROGRESS.md § T-M7-19).

## Scope
1. After a custom checkup is deleted (`useDeleteCustomCheckup` onSuccess), prune its records' on-device attachments
   right away (`pruneCheckupAttachments`/a "soon" variant from `entities/checkup`).
2. When the API answers 422 with `error_code: "limit_reached"` (medications, appointments, custom checkups), the
   form shows the server's localized `message` (not the generic error). Reuse the shared API error type; no new
   i18n keys needed. Tests.

## Acceptance
- Both done with unit tests; verify green.
