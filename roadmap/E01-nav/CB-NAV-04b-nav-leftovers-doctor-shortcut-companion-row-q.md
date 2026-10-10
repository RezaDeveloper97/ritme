---
id: CB-NAV-04b
title: NAV leftovers: doctor shortcut, companion row, quick-access server storage
epic: NAV
type: frontend
status: done
depends_on: [CB-NAV-04]
parallel_group: NAV-D
touches: [frontend/src/widgets/quick-access,frontend/src/widgets/for-today]
skills: [new-fsd-slice]
boards: [nbd_Nav_Today.dc.html]
verify: cd frontend && npm run typecheck && npm run test
---

# CB-NAV-04b — NAV leftovers: doctor shortcut, companion row, quick-access server storage

## Why
Leftovers from the NAV QA (docs/qa/canvas/nav.md `## CB-NAV-04`) that need a decision or another epic's screen.

## Boards
- `nbd_Nav_Today.dc.html`

## Scope
1. «پزشک» quick-access shortcut: set its `href` once the doctor directory route (CB-DIR / B-N7) exists.
2. Companion suggestion row in «برای امروز» once an owner-facing companion-suggestion API exists.
3. Quick-access server storage (README §8 Q3) — only after the user decides; today it is device-local.
4. Mount «برای امروز» + «دسترسی سریع» on the teen home (widgets already support `teen`).

## Out of scope
- Android/native work.

## Acceptance
- Each item shipped or explicitly dropped; QA table updated; `verify` green.
