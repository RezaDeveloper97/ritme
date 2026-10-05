---
id: L9-01
title: Critical CSS, font loading and asset budget
milestone: L9
type: quality
status: done
depends_on: [L3-11,L4-03,L5-03,L6-03]
parallel_group: L9-A
touches: [vite.config.js,tools/critical.mjs,resources/views/layouts,resources/css,docs/PERFORMANCE.md]
skills: []
verify: composer verify
---

# L9-01 — Critical CSS, font loading and asset budget

## Scope
- Measure per-route CSS/JS/HTML/font bytes (gzip) → `docs/PERFORMANCE.md` budget table (targets: HTML < 40 KB gz,
  CSS < 25 KB gz, JS < 15 KB gz per page, fonts ≤ 2 preloaded).
- Decide by measurement: inline the whole CSS when < 14 KB gz, else build-time critical CSS per template
  (`tools/critical.mjs` via headless Chrome coverage, no external services) inlined with the rest loaded
  non-blocking (`media=print onload` pattern without inline JS — use a module or `<link rel=preload as=style>`;
  CSP-safe).
- Fonts: preload only the hero weights, verify `unicode-range` split avoids latin file on Persian-only pages, check
  `size-adjust` fallback metrics to reduce CLS on swap.
- Ensure no unused JS is loaded on pages without modules; `modulepreload` for page modules.

## Acceptance
- Budget met on every template or exceptions justified in `docs/PERFORMANCE.md`.
