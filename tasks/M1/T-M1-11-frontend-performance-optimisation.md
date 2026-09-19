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
Only the ranked items from `docs/investigations/perf-baseline.md` §3, in this order:
1. **(#4) Fonts**: stop preloading all 6 Vazirmatn weights (128 KB on every page, 31 % of prod page weight). Use
   one variable woff2, or drop 500 (2 uses → 400/600) and 900 (10 uses → 800) and preload only the body weight.
2. **(#5) Home CLS**: reserve space (skeleton/min-height) for the home `.sec` blocks that shift when data arrives.
   Baseline CLS is 0.063 simulated and 0.201 devtools-throttled.
3. **(#3, frontend half) Slim month**: switch `useCycleMonth` to the calendar view **only if T-M1-12 has shipped it**.
   Otherwise skip it and note it as a follow-up.
4. **(#8) Fewer cold-load requests**: home fires 11 GETs + 11 CORS preflights. Merge or defer the small boot reads
   where it is cheap. The same-origin API proxy is infra and out of scope here (baseline §5 open item 2).
5. **(#9) Per-route client weight**: pass only the needed namespaces to `NextIntlClientProvider` (all 23, 50 KB, go
   into every page today). Replace axios with a thin `fetch` client in `shared/api` (−15 KB gz). Optionally look at
   zod 4 / `zod/mini`.
6. **(#10, frontend half) Dead code/deps**: remove `react-hook-form` and `@hookform/resolvers`,
   `shared/lib/cookie-state`, the duplicate `screens/home/ui/SectionHead.tsx`, and `widgets/day-tasks` (verify first).
   **Do not delete `features/manage-account`**: it is unwired, not dead (CLAUDE.md §11 export/delete), and is a
   product decision.

Measured as **not** worth doing: RSC conversion of screens, dayjs/jalaliday, images, TanStack dedupe (no duplicate
fetches found), and more `next/dynamic` (sheets are already dynamic). Keep FSD layering and fa/en i18n intact.
`@next/bundle-analyzer` stays ad-hoc; don't add it to `package.json`.

## Out of scope
Visual redesigns; anything not in the baseline's list.

## Acceptance
- Re-measure with the same method as the baseline (§6.4: route table, analyzer JSON, Lighthouse ×3 with SW/cache
  cleared on `/fa/home`, `/fa/calendar`, `/fa/log`, CDP request count). Append a before/after table to
  `docs/investigations/perf-baseline.md` (append is fine even though another task created it). No regressions.
- Full frontend gate green including `npm run build`.
