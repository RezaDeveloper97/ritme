---
id: T-M1-09
title: PWA — iOS standalone and safe-area polish
milestone: M1
type: frontend
status: todo
depends_on: [T-M1-06]
parallel_group: M1-D
touches: [frontend/src/app/[locale]/layout.tsx, frontend/public/splash]
skills: [pwa]
verify: cd frontend && npm run typecheck && npm run lint && npm run lint:styles && npm run lint:dark && npm run build
---

# T-M1-09 — PWA — iOS standalone and safe-area polish

## Scope
Fix the iOS gaps from `docs/investigations/pwa-audit.md`: `apple-mobile-web-app-capable` / title /
status-bar-style (light + dark), startup images, `viewport-fit=cover` with `env(safe-area-inset-*)` usage where
content hits the notch/home indicator, theme-color per scheme. The theme-color hex in `layout.tsx` is a baselined
exception; keep colours tokenised elsewhere (CLAUDE.md §10).

## Acceptance
- Screenshot in an iPhone-sized viewport (memory `ritme-headless-ui-verification`) in light + dark: no content
  under the notch or home indicator.
- lint:styles and lint:dark green.
