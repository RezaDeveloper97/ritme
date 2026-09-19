---
id: T-M1-11
title: Frontend performance optimisation
milestone: M1
type: frontend
status: todo
depends_on: [T-M1-10, T-M1-03, T-M1-07, T-M1-08, T-M1-09]
parallel_group: M1-E
touches: [frontend/src, frontend/next.config.ts, frontend/package.json, frontend/package-lock.json]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# T-M1-11 — Frontend performance optimisation

## Why
Implement the frontend items from `docs/investigations/perf-baseline.md`. Depends on the session and PWA tasks
because it touches `frontend/src` broadly — running it in parallel with them would conflict.

## Scope
Only the ranked items from the baseline, typically: code-split heavy sheets/charts with `next/dynamic`, move
static parts to server components, dedupe/prefetch queries and tune `staleTime`, trim unused font weights and
dayjs locales, image sizing, remove dead code/deps found by the baseline. Keep FSD layering and fa/en i18n
intact.

## Out of scope
Visual redesigns; anything not in the baseline's list.

## Acceptance
- Re-measure with the same method as the baseline; append a before/after table to
  `docs/investigations/perf-baseline.md` (append is fine even though another task created it). No regressions.
- Full frontend gate green including `npm run build`.
