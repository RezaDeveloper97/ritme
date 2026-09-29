---
id: T-M3-10
title: Care reminders — design fidelity fixes (v13 audit)
milestone: M3
type: frontend
status: todo
depends_on: [T-M3-08]
parallel_group: M3-D
touches: [frontend/src/screens/reminder-appointment-form,frontend/src/screens/reminder-medication-form,frontend/src/screens/reminder-appointment-detail,frontend/src/screens/reminders,frontend/src/widgets,frontend/src/features,frontend/src/entities/care-reminder,frontend/src/app/globals.css,frontend/messages,backend-go/resources/translations,backend-go/internal/i18n/testdata,docs/care-reminders]
skills: [verify-all,check-colors]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../backend-go && go test ./... && make contract ROUTES=all
---

# T-M3-10 — Care reminders — design fidelity fixes (v13 audit)

## Why
Design fidelity audit (docs/care-reminders/design-audit.md) compared the implementation with `docs/design/reminders-v13/` and listed deviations by
severity with file:line and a suggested fix.

## Scope
- Fix EVERY high and med item in docs/care-reminders/design-audit.md, and the low items that are cheap (same file you're already in). For any
  item you decide not to fix, write the reason in the audit table (column "Resolution").
- Palette rules (frontend/CLAUDE.md §10.2) win over design colours; layout/content/copy follow the design.
- New/changed copy in fa + en; sync the backend translation seed and i18n goldens (tests enforce it).

## Out of scope
- Clinical content; other features' screens.

## Acceptance
- Audit table updated with a Resolution per row; verify green; light + dark full-page screenshots re-taken locally
  next to the design ones in the audit folder (`*-impl.png`, same names) and visually checked.
