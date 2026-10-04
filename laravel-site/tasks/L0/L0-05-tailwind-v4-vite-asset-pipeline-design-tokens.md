---
id: L0-05
title: Tailwind v4 + Vite asset pipeline, design tokens, self-hosted fonts
milestone: L0
type: frontend
status: todo
depends_on: [L0-04]
parallel_group: L0-C
touches: [vite.config.js,package.json,resources/css,resources/js,resources/fonts,tailwind.config.js,postcss.config.js]
skills: []
verify: composer verify
---

# L0-05 — Tailwind v4 + Vite asset pipeline, design tokens, self-hosted fonts

## Why
A purged, hashed, minified build is the base for GTmetrix scores and for never serving stale CSS/JS.

## Scope
- Tailwind CSS v4 via `@tailwindcss/vite`; `resources/css/app.css` with `@theme` tokens from `docs/AUDIT.md`
  (colors, fonts, radii, shadows, breakpoints matching 1180/1024/700 → custom `max-*` variants), base layer (body
  font, colors, focus-visible ring, `details` reset), component layer only for genuinely shared bits (`.rt-prose`).
- Fonts: copy `design/html/assets/fonts/*.woff2` to `resources/fonts`; `@font-face` with `font-display: swap` and the
  existing `unicode-range` split; Vite emits hashed font files. Only weights actually used (audit) are kept.
- Vite: entries `resources/css/app.css`, `resources/js/app.js` (+ per-page lazy modules later), `build.manifest`,
  hashed names, `cssMinify: 'lightningcss'`, modern browser targets, no sourcemaps in prod, assets inlining limit 2 KB.
- `resources/js/app.js` is tiny and module-based: imports page modules lazily by `data-module` attribute
  (`import.meta.glob` + dynamic import) so pages pay only for what they use.
- Document in `laravel-site/CLAUDE.md`: how to add tokens, never write raw hex in Blade (use theme tokens), RTL
  logical utilities (`ps-/pe-/ms-/me-/start/end`).

## Out of scope
- Converting pages (L3), critical CSS (L9-01).

## Acceptance
- `npm run build` outputs hashed CSS/JS/fonts + `manifest.json`; a test Blade view using `@vite` renders hashed URLs;
  built CSS for an empty page < 15 KB gzip.
