---
id: T-M12-27b
title: Lazy sheet registry: global CSS 216 KB toward 114 KB
milestone: M12
type: frontend
status: todo
depends_on: [T-M12-27]
parallel_group: M12-perf
touches: [frontend/src/app/sheets,frontend/src/app/layout.tsx,frontend/src/app/ModuleRouteGate.tsx,frontend/CLAUDE.md,.claude/skills/verify-all/SKILL.md,frontend/scripts/check-bundle-size.mjs,docs/quality-2026-11/perf.md]
skills: []
verify: cd frontend && npm run build && npm run check:bundle && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M12-27b — Lazy sheet registry: global CSS 216 KB toward 114 KB

## Why
T-M12-27 cut `globals.css` to 114 KB, but the root layout statically imports `src/app/sheets/registry.tsx` and a
few gates, which pull ~100 KB of sheet CSS (lab, plus, companion, record, onboarding, rmd/cfm) into every route:
216 KB global. Sheet CSS reached from the layout also loads *before* globals.css, so it loses cascade ties (the
calendar pill regression T-M12-27 had to fold back: `cal-`, `osheet-`, `pad-`, `fld-`, `jcal-`).

## Scope
1. Load sheets in the registry with `next/dynamic` (per sheet, `ssr:false` where already client-only) and the global
   gates' heavy children the same way; keep open-latency acceptable (prefetch on idle if needed).
2. Re-measure; lower the `check-bundle-size.mjs` global budget to measured × 1.1.
3. Try moving `cal-`/`osheet-`/`pad-`/`fld-`/`jcal-` out again once they are no longer layout-reachable; screenshot
   calendar + log light/dark, pixel-diff vs before.
4. `frontend/CLAUDE.md` §10.1: classes live in `globals.css` (tokens, shell, primitives) **or** in a slice's
   `ui/<prefix>.css` / `shared/ui/styles/<prefix>.css`; CSS reachable from the root layout must win by specificity.
   Add `npm run check:bundle` to the verify-all skill.
5. Optional: a small vitest for `check-bundle-size.mjs` (over-budget fixture exits 1).

## Out of scope
- 

## Acceptance
- CSS every route downloads ≤ 150 KB minified; bundle gate budget lowered; no pixel diffs on the screens touched.
- CLAUDE.md + verify-all updated; all gates green.
