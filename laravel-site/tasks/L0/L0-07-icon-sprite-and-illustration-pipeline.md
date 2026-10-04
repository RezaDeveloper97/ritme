---
id: L0-07
title: Icon sprite and illustration pipeline
milestone: L0
type: frontend
status: todo
depends_on: [L0-04]
parallel_group: L0-B
touches: [resources/svg,resources/views/components/icon.blade.php,tools/build-sprite.mjs,app/View/Components/Icon.php,vite.config.js]
skills: []
verify: composer verify
---

# L0-07 — Icon sprite and illustration pipeline

## Why
The pages repeat the same inline SVGs dozens of times (up to 69 per page). A cached sprite cuts HTML weight; big
illustrations need optimisation.

## Scope
- Extract unique icons from `design/html/*.html` (dedupe by normalized path data; names from `docs/AUDIT.md`) into
  `resources/svg/icons/*.svg`; illustrations into `resources/svg/illustrations/`.
- SVGO (dev dep) at build: optimise all SVGs; generate `sprite.svg` with `<symbol id>` per icon as a hashed Vite
  asset (`currentColor` strokes so color comes from CSS).
- `<x-icon name="drop" class="size-5" />` → `<svg aria-hidden="true" focusable="false"><use href="{hashed sprite}#drop"/></svg>`;
  optional `label` prop → `role="img"` + `<title>`. Unknown icon name throws in non-production.
- `<x-illustration name="…" />` inlines optimised SVG from disk (cached via CacheAside `media` ns) with width/height
  attributes to prevent CLS.
- Hashed sprite is preloaded only on pages that use icons above the fold (`<link rel="preload" as="image">` not
  needed for `<use>` — measure; document the choice).

## Out of scope
- Raster images (L2).

## Acceptance
- Sprite builds; component unit test; one sample view renders icons correctly in Chrome and Safari (same-origin
  `<use>`); icon count and sprite size noted in `docs/AUDIT.md`.
