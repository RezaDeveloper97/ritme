---
id: T-M7-18
title: Pregnancy v2 — design fidelity fixes (v2 audit)
milestone: M7
type: frontend
status: done
depends_on: [T-M7-17,T-M2-34]
parallel_group: M7-E
touches: [frontend/src/screens/pregnancy,frontend/src/screens/pregnancy-onboarding,frontend/src/screens/pregnancy-week,frontend/src/screens/pregnancy-log,frontend/src/screens/pregnancy-calendar,frontend/src/screens/pregnancy-alerts,frontend/src/widgets,frontend/src/features,frontend/src/entities/pregnancy,frontend/src/shared/lib/date,frontend/src/app/globals.css,frontend/src/app/message-scopes.ts,frontend/messages,backend-go/internal/pregnancy,backend-go/internal/messages/pregnancyalerts,backend-go/api/openapi.yaml,backend-go/contract,backend-go/resources/translations,backend-go/internal/i18n/testdata,docs/pregnancy-v2]
skills: [verify-all,check-colors]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../backend-go && go vet ./... && go test ./... && make contract ROUTES=all
---

# T-M7-18 — Pregnancy v2 — design fidelity fixes (v2 audit)

## Why
Design fidelity audit (docs/pregnancy-v2/design-audit.md) compared the implementation with `docs/design/pregnancy-v2/`:
37 deviations (4 high, 12 med, 21 low).

## Scope
- Fix EVERY high and med item, and the cheap low items. For any item not fixed, write the reason in the audit
  table (column "Resolution"). Includes the backend parts (carousel title duplicating the eyebrow, `source_note`,
  `week_entered` «از امروز», week numbering decision — follow the design's completed-weeks convention only if it
  matches the clinical convention used elsewhere in the app; otherwise document why not).
- Setup welcome must read the admin-editable `pregnancy_setup/*` copy (so admin edits take effect).
- Dark-mode «ذخیره شد» button contrast (use a theme-stable fill token); alerts actions styled per level (no gradient
  on every card, §10.2); calendar care rows show «حدود …» from `suggestedDate`; move direct `Intl.DateTimeFormat`
  use into `shared/lib/date` (§7).
- Known items 9b (after saving an appointment from the calendar, return to the calendar), 9d (alert ordering),
  9i (PDF separator readability) if not already fixed.
- Palette rules win over design colours; copy in fa + en; seed + i18n goldens synced.

- From T-M3-10 (reminders audit L-6): place the «یادآورهای امروز» card where the reminders design shows it on this home and make sure the same visit isn't shown twice on the home.

## Out of scope
- Clinical content sign-off (human); content register rewrite.

## Acceptance
- Audit table has a Resolution per row; verify green; light + dark full-page `*-impl` screenshots re-taken in the
  audit folder and visually checked.
