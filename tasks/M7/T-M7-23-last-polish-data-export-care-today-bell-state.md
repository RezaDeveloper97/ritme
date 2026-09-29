---
id: T-M7-23
title: Last polish — data export, care/today bell state, 429 copy, admin number inputs, PWA banner layering
milestone: M7
type: frontend
status: done
depends_on: [T-M2-35,T-M7-22]
parallel_group: M7-E
touches: [frontend/src/screens/profile,frontend/src/features/manage-account,frontend/src/entities/care-reminder,frontend/src/widgets/today-reminders,frontend/src/screens/reminders,frontend/src/screens/profile-reminders,frontend/src/screens/reminder-appointment-form,frontend/src/screens/checkup-custom-form,frontend/src/screens/checkup-mark-done,frontend/src/shared/api,frontend/src/features/pwa-install,frontend/src/widgets/pwa-install,frontend/src/app/globals.css,frontend/src/app/message-scopes.ts,frontend/messages,backend-go/internal/care,backend-go/internal/http/write_throttle.go,backend-go/internal/platform/ratelimit,backend-go/resources/translations,backend-go/internal/i18n/testdata,backend-go/api/openapi.yaml,backend-go/contract,admin-web/src,admin-web/messages,docs/qa]
skills: [verify-all,check-colors]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../admin-web && npm run typecheck && npm run lint && npm run test && cd ../backend-go && go vet ./... && go test ./... && golangci-lint run && make contract ROUTES=all
---

# T-M7-23 — Last polish — data export, care/today bell state, 429 copy, admin number inputs, PWA banner layering

## Why
Open items of T-M7-22 and T-M2-35 (tasks/PROGRESS.md).

## Scope
1. Profile: restore «گرفتن خروجی داده‌ها» (the existing `useExportData` hook + its keys; §11 "support data export").
   Check the export works end to end locally (what file/format it produces).
2. `/care/today` `next_appointment` (and any appointment rows it returns) carries `is_active`; the home reminders
   card uses it and drops the per-appointment `GET /care/appointments/{id}` added in T-M7-22. OpenAPI + tests.
3. Write-throttle 429 body localized (fa/en, same envelope shape as other errors) in Go; appointment, custom
   checkup and MarkDone forms show it (reuse the shared helper pattern from T-M7-21/T-M2-35). Contract: if a
   Laravel-parity route's 429 body is pinned in goldens, keep those unchanged (only the new write throttle).
4. Move the duplicated medication schedule rule (profile-reminders `rowSchedule` vs reminders `medicationSchedule`)
   into `entities/care-reminder`, used by both.
5. admin-web number inputs show fa digits (a text+inputmode numeric field like the app's `LocaleNumberField`),
   accept both digit sets.
6. PWA install banner never covers dialogs/sheets (layering) — find the banner component and fix z-index/stacking
   via tokens/classes (no inline styles).

## Acceptance
- All 6 done with tests; verify green.
