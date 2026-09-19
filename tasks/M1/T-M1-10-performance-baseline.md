---
id: T-M1-10
title: Performance baseline (frontend + backend), measured
milestone: M1
type: investigate
status: done
depends_on: []
parallel_group: M1-A
touches: [docs/investigations/perf-baseline.md]
skills: []
verify: test -s docs/investigations/perf-baseline.md
---

# T-M1-10 — Performance baseline (frontend + backend), measured

## Why
"Optimise the code" needs numbers first, otherwise we optimise the wrong thing and can't prove a win.

## Scope
Measure, don't change:
- **Frontend**: `next build` route table (first-load JS per route), bundle composition (`@next/bundle-analyzer` run
  ad-hoc — don't commit the dependency unless T-M1-11 needs it), Lighthouse mobile on `/fa/home`, `/fa/calendar`,
  `/fa/log` (LCP, INP/TBT, CLS), number of API requests on cold home load, TanStack Query duplicate fetches,
  heavy client components that could be server components, dayjs/jalaliday + locale weight, fonts (6 Vazirmatn
  weights — are all used?), images.
- **Backend**: p50/p95 of the hot endpoints (home, cycle view, calendar, daily messages) on a realistic seeded
  user; query count per request (enable query log / `DB::listen`), N+1s, missing indexes (`EXPLAIN`), cacheable
  per-user computations (cycle engine), opcache/config/route cache in the prod image.
- **Dead code**: unused files/exports/deps (`npx knip` or `ts-prune`; `cycle_calculations` is known dead storage —
  memory `ritme-cycle-engine`).

## Acceptance
- `docs/investigations/perf-baseline.md`: numbers table + top 10 opportunities ranked by (impact ÷ effort),
  each mapped to T-M1-11 / T-M1-12 / T-M1-13. Edit those task files to match.
