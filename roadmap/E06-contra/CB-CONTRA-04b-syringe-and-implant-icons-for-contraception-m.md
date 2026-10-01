---
id: CB-CONTRA-04b
title: Syringe and implant icons for contraception methods
epic: CONTRA
type: frontend
status: todo
depends_on: [CB-CONTRA-04]
parallel_group: CONTRA-E
touches: [frontend/src/shared/ui/Icon.tsx,frontend/src/entities/contraception/model/look.ts,docs/qa/canvas/contra.md,docs/qa/canvas/contra]
skills: [new-fsd-slice]
boards: [nbl_Contra_Setup.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run lint:styles && npm run lint:dark && npm run test
---

# CB-CONTRA-04b — Syringe and implant icons for contraception methods

## Why
CB-CONTRA-04 QA: the setup tiles use `calendar` for the injection and `hand` for the implant; the board shows a syringe and a rod. The shared icon set has neither glyph.

## Boards
- `nbl_Contra_Setup.dc.html`

## Scope
1. Add `syringe` and `implant` glyphs to `shared/ui/Icon.tsx` (same stroke style as the set).
2. Use them in `entities/contraception/model/look.ts` for injection / implant (setup tiles + other-methods cards).

## Out of scope
- Anything else in the icon set; Android/native (never).

## Acceptance
- Setup + other screens show the new glyphs in light and dark; row in `docs/qa/canvas/contra.md` updated to ✔.
- `verify` green.
