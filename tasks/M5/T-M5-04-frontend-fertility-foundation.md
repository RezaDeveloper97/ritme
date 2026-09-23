---
id: T-M5-04
title: Frontend — fertility entity, day-log feature, tokens, icons and i18n
milestone: M5
type: frontend
status: done
depends_on: []
parallel_group: M5-A
touches: [frontend/src/entities/fertility, frontend/src/features/log-fertility-day, frontend/src/app/globals.css, frontend/src/shared/ui/Icon.tsx, frontend/messages/fa, frontend/messages/en, frontend/src/shared/i18n, frontend/src/app/message-scopes.ts]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M5-04 — Frontend — fertility entity, day-log feature, tokens, icons and i18n

## Why
Foundation for the M5 screens; code against docs/fertility-ttc/README.md before the backend lands.

## Scope
1. `entities/fertility`: types, zod schemas (unknown enum → null), queries + `fertilityKeys` (today, day, bbt,
   insights); BBT formatting helpers (fa digits, «٫», 2 decimals) with tests.
2. `features/log-fertility-day`: PUT mutation; invalidates `fertilityKeys.all`, the cycle/home keys and the health-log
   keys (same columns).
3. Tokens for the README color table in both `:root` and `[data-theme="dark"]` (reuse M3/M4 tokens if present).
4. Icons: flask-LH, heart, target, moon-reminder (thermo exists) — stroke paths from the artboards.
5. i18n namespace `fertility` (fa + en) with all copy of the four artboards.

## Acceptance
- Parser/formatter tests; lint gates green; no hex outside globals.css.
