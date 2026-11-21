---
id: T-M12-27
title: Split the 1 MB globals.css into per-screen CSS and add a CSS byte budget
milestone: M12
type: frontend
status: done
depends_on: [T-M12-12]
parallel_group: M12-PERF
touches: [frontend/src/app/globals.css, frontend/src/app, frontend/src/entities, frontend/src/shared/ui/styles, frontend/src/shared, docs/quality-2026-11/perf.md, frontend/src/screens, frontend/src/widgets, frontend/src/features, frontend/scripts/check-bundle-size.mjs, frontend/package.json, frontend/scripts/check-styles.mjs, frontend/scripts/check-dark-mode.mjs]
skills: []
verify: cd frontend && npm run build && node scripts/check-bundle-size.mjs && npm run lint:styles && npm run lint:dark
---

# T-M12-27 — Split the 1 MB globals.css into per-screen CSS and add a CSS byte budget
**Model:** sonnet (opus reviews). Evidence: `docs/quality-2026-11/findings.md`.

## Why
P-01: every route downloads and parses ~1 MB of render-blocking CSS, mostly for screens the user is not on.

## Scope
1. Record the "before" prod build numbers (`.next/static/css` bytes, Lighthouse mobile on /fa/home, /fa/calendar, /fa/log) in `docs/quality-2026-11/perf.md`.
2. Move each screen-prefixed block (`.nb-`, `.lab-`, `.pgn-`, `.plus-`, `.lrn-`, `.ttc-`, `.ins-`, `.shp-`, …) to a CSS file imported by that screen so Next emits it per route; tokens, shell and shared primitives stay global.
3. Keep the style gate and dark-mode gate green (update their globs to the new files).
4. `check-bundle-size.mjs`: fails when the global layout CSS or the first-load JS of /fa/home exceeds a budget (measured × 1.1).
5. Screenshot every moved screen in light and dark (headless UI verification) to catch missing styles.

## Out of scope
- Visual changes.
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Global CSS under ~150 KB built; "after" numbers in perf.md; no visual diffs.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
