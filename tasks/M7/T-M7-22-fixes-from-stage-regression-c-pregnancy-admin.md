---
id: T-M7-22
title: Fixes from stage regression C (pregnancy, admin-web)
milestone: M7
type: frontend
status: in_progress
depends_on: [T-M7-21]
parallel_group: M7-E
touches: [admin-web/src,admin-web/messages,backend-go/internal/admin,backend-go/internal/pregnancy,backend-go/db/queries/pregnancy,backend-go/internal/messages/pregnancyalerts,frontend/src/screens/pregnancy-calendar,frontend/src/screens/pregnancy-log,frontend/src/screens/onboarding-pregnancy-basis,frontend/src/screens/profile,frontend/src/features/manage-account,frontend/src/widgets/today-reminders,backend-go/api/openapi.yaml,docs/qa]
skills: [verify-all,check-colors]
verify: cd admin-web && npm run typecheck && npm run lint && npm run test && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run test && cd ../backend-go && go vet ./... && go test ./... && golangci-lint run && make contract ROUTES=all
---

# T-M7-22 — Fixes from stage regression C (pregnancy, admin-web)

## Why
Final stage regression part C (docs/qa/stage-regression-2026-09-29-c.md) found 2 medium + 8 low bugs.

## Scope
- M1 admin-web: list screens scroll sideways at 390 px — `.table-wrap { position: relative }` (and check every list).
- M2 account deletion reachable again in the web app (Profile → delete account with the existing `DeleteAccountConfirm`,
  §11 "support data export/delete"); verify the flow ends the session and clears local data.
- L1 Today badge counts only alerts the Alerts screen shows (exclude legacy v1 `symptom_based` or show it too — be consistent).
- L2 onboarding manual week field uses `LocaleNumberField`.
- L3 admin-web fa digits everywhere (pagination, ranges, counts, alert preview) via the admin number formatter.
- L4 bell-off appointment: calendar and cycle-home reminders card don't show «یادآور: … قبل».
- L5 re-dating in Setup refreshes the `week_entered` alert (text + fact date) for the current week.
- L6 Log spotting box colour matches the Alerts level colour.
- L7 admin week grid reflects structured week details (1–42) and `/pregnancy-weeks/{n}` opens existing details.
- L8 Go admin writes JSON without `\uXXXX` escaping (SetEscapeHTML(false)/no ASCII escaping) for translatable fields; test that Persian is stored as UTF-8.

## Acceptance
- All fixed with tests; verify green; QA doc gets a Resolution line per bug.
