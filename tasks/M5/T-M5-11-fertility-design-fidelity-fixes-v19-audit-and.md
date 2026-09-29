---
id: T-M5-11
title: Fertility — design fidelity fixes (v19 audit) and TTC home
milestone: M5
type: frontend
status: todo
depends_on: [T-M5-10]
parallel_group: M5-D
touches: [frontend/src/screens/home,frontend/src/screens/fertility-log,frontend/src/screens/fertility-bbt,frontend/src/screens/fertility-insights,frontend/src/widgets,frontend/src/features,frontend/src/entities/fertility,frontend/src/entities/cycle,frontend/src/app/globals.css,frontend/src/app/message-scopes.ts,frontend/messages,backend-go/resources/translations,backend-go/internal/i18n/testdata,docs/fertility-ttc]
skills: [verify-all,check-colors]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../backend-go && go test ./... && make contract ROUTES=all
---

# T-M5-11 — Fertility — design fidelity fixes (v19 audit) and TTC home

## Why
Design fidelity audit (docs/fertility-ttc/design-audit.md) compared the implementation with `docs/design/ttc-v19/` and listed deviations by
severity with file:line and a suggested fix.

## Scope
- Fix EVERY high and med item in docs/fertility-ttc/design-audit.md, and the low items that are cheap (same file you're already in). For any
  item you decide not to fix, write the reason in the audit table (column "Resolution").
- fertility_level: the home must use the top-level v1.1 `fertility_level` (`infoView.fertilityLevel`) with the `fertility.chance.levels.*` labels, same as the fertility screens (frontend only; don't change `daily_card` before T-M2-26).
- Build the TTC-specific home parts the design shows for `trying` users (ring centre «تخمک‌گذاری تا N روز», phase pills, «شانس بارداری امروز» card, LH tip card) using the existing `fertility.home.*` keys; non-TTC homes unchanged.
- Fix the tiles late render (they wait on GET /profile, `HomePage.tsx` ~885): render from cached/known intention or show a stable skeleton so the layout doesn't jump.
- Palette rules (frontend/CLAUDE.md §10.2) win over design colours; layout/content/copy follow the design.
- New/changed copy in fa + en; sync the backend translation seed and i18n goldens (tests enforce it).

## Out of scope
- Clinical content; other features' screens.

## Acceptance
- Audit table updated with a Resolution per row; verify green; light + dark full-page screenshots re-taken locally
  next to the design ones in the audit folder (`*-impl.png`, same names) and visually checked.
