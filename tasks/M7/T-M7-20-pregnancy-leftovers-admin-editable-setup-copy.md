---
id: T-M7-20
title: Pregnancy leftovers — admin-editable setup copy, source note, alert seed text, badge, 9b
milestone: M7
type: backend
status: done
depends_on: [T-M7-18,T-M2-34]
parallel_group: M7-E
touches: [backend/database/migrations, admin-web/messages, frontend/src/screens/pregnancy-calendar/ui/PregnancyCalendarPage.tsx, backend-go/internal/http/routes_pregnancy_v2.go,backend-go/internal/pregnancy,backend-go/internal/messages,backend-go/internal/admin/messages,backend-go/db/migrations,backend-go/db/queries,backend-go/api/openapi.yaml,backend-go/contract,frontend/src/screens/pregnancy-onboarding,frontend/src/screens/pregnancy,frontend/src/screens/reminder-appointment-form,frontend/src/entities/pregnancy,frontend/src/shared/ui,frontend/scripts/check-dark-mode.mjs,admin-web/src,docs/pregnancy-v2,docs/go-migration]
skills: [verify-all,new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && golangci-lint run && make contract ROUTES=all && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../admin-web && npm run typecheck && npm run test
---

# T-M7-20 — Pregnancy leftovers — admin-editable setup copy, source note, alert seed text, badge, 9b

## Why
Items T-M7-18 and T-M2-34 could not finish inside their touches (see their PROGRESS sections and
docs/pregnancy-v2/design-audit.md A1/E2/F4/D1 rows, docs/go-migration/backend-review-m3-m7.md #10/#11/#13).

## Scope
1. A1: `GET /api/v1/pregnancy/v2/setup-copy` returning the admin-edited `pregnancy_setup/*` texts (welcome, dating,
   history, result) in the request locale; the Setup screens read it (bundled copy only as fallback). OpenAPI,
   contract case (Go-only route), int test.
2. E2: calendar `source_note` caveat becomes an admin-editable message item (registry entry + guarded seed migration),
   Go lang text as fallback.
3. F4: `pregnancy_alert/week_entered` seed text no longer says «از امروز» (guarded data migration: only rows equal to
   the old seed; the card already shows the fact date).
4. D1: add the `--success-fill`/on-fill pair to `frontend/scripts/check-dark-mode.mjs`.
5. 9b: after saving an appointment opened from the pregnancy calendar, return to the calendar (e.g. `returnTo`
   param validated against an allow-list of in-app paths — no free-form redirect).
6. Ultrasound week/day inputs show Persian digits in fa (shared `NumberField` or a locale-aware variant; no
   regression elsewhere).
7. T-M2-34 #10: v2 Today's unread badge counts computed alerts (week_entered, weight_missing_week) without opening
   Alerts — evaluate in the Today handler (idempotent) or a daily job.
8. T-M2-34 #11: `weight_missing_week.from_weekday` editable in admin (registry params schema + admin-web form).
9. Fix the stale `create_profile_empty_body` description in `backend-go/contract/cases/profile.yaml` (D-24).

## Acceptance
- All 9 done with tests; verify green (incl. contract, admin-web); audit rows A1/E2/F4/D1 → Fixed.
