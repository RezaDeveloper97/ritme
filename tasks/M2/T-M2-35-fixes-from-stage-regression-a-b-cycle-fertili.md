---
id: T-M2-35
title: Fixes from stage regression A+B (cycle, fertility, care, checkups)
milestone: M2
type: frontend
status: in_progress
depends_on: [T-M5-13,T-M7-21]
parallel_group: M2-J
touches: [frontend/src/screens/calendar,frontend/src/screens/home,frontend/src/screens/fertility-log,frontend/src/screens/fertility-insights,frontend/src/screens/profile-reminders,frontend/src/screens/checkup-detail,frontend/src/screens/reminder-medication-form,frontend/src/entities/cycle,frontend/src/entities/health-log,frontend/src/entities/fertility,frontend/src/widgets/fertility-tiles,frontend/src/app/globals.css,frontend/messages,backend-go/internal/cycle,backend-go/internal/messages/manager,backend-go/internal/reminder,backend-go/internal/care,backend-go/resources/translations,backend-go/internal/i18n/testdata,backend-go/api/openapi.yaml,backend-go/contract/allowlist,docs/go-migration/deviations.md,docs/qa]
skills: [verify-all,check-colors]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../backend-go && go vet ./... && go test ./... && golangci-lint run && make contract ROUTES=all
---

# T-M2-35 — Fixes from stage regression A+B (cycle, fertility, care, checkups)

## Why
Final stage regression (docs/qa/stage-regression-2026-09-29-a.md and -b.md) found 10 + 4 bugs (details, file:line, repro there).

## Scope
Part A:
- B-1 calendar day sheet chance uses the same level/labels as Log (`fertility_level` v1.1), not a local 3-level map.
- B-2 post-ovulation note: «کم» must not say fertility is higher — `daily_card` note for `post_ovulation` (Go; deviation D-nn + allowlist, Laravel copied the bug).
- B-3 `/messages/daily` phase/is_fertile_window from the v1.1 resolver (§19) so avoiding users don't get "peak fertility" tips on O+1 (Go; deviation + allowlist).
- B-4 one PMS length everywhere per task.md §25.2 (calendar, home timeline, insights strips) — shared constant in entities/cycle.
- B-5 no console error for a day without a health log (treat 404 as empty in the query).
- B-6 chance card copy follows the ring's days-to-ovulation, B-7 Log title for past dates («شانس بارداری در این روز»), B-8 insights calendar caption names both months when the first row spans two, B-9 lavender page background on Log/BBT/Insights (light), B-10 ovulation row turquoise in the home timeline.
Part B:
- B-1 legacy reminders sheet: a cancelled appointment can't be toggled on (API 422 + UI disabled), B-2 legacy sheet separators «، » and hours without leading zero in fa, B-3 alternate-day medication label in the legacy sheet, B-4 checkup detail «ثبت نوبت» uses the same prefill handoff as the home card.
- Note: pregnant users' medication default duration «تا پایان بارداری» per design, if the form supports an end date.
- 429 bodies: Go returns a localized message (fa/en) for the write throttle, and forms show it.

## Acceptance
- All items fixed with tests; verify green; the QA docs get a "Resolution" line per bug.
