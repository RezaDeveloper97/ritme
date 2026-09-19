---
id: T-M1-09
title: PWA — iOS standalone and safe-area polish
milestone: M1
type: frontend
status: done
depends_on: [T-M1-06]
parallel_group: M1-D
touches: [frontend/src/app/[locale]/layout.tsx, frontend/src/app/globals.css, frontend/public/splash]
skills: [pwa]
verify: cd frontend && npm run typecheck && npm run lint && npm run lint:styles && npm run lint:dark && npm run build
---

# T-M1-09 — PWA — iOS standalone and safe-area polish

## Scope
Fix the iOS gaps from `docs/investigations/pwa-audit.md`: `apple-mobile-web-app-capable` / title /
status-bar-style (light + dark), startup images, `viewport-fit=cover` with `env(safe-area-inset-*)` usage where
content hits the notch/home indicator, theme-color per scheme. The theme-color hex in `layout.tsx` is a baselined
exception; keep colours tokenised elsewhere (CLAUDE.md §10).

Gaps found by T-M1-06 (ids refer to the audit table):
- I-5 / U-10 (Med) no bottom safe-area on `.tabbar` (12px from the bottom of the full-height shell), `.pwa-toast` /
  `.pwa-install` and the `bottom: 88px` fixed bars (`.cal-fab-row`, globals.css ~1417) — they sit on the home
  indicator in iOS standalone. (`globals.css` touches overlap with other CSS tasks — rebase carefully.)
- I-3 (Med) `statusBarStyle: 'default'` keeps a white status bar in dark mode → `black-translucent` (the shell
  already pads `env(safe-area-inset-top)`).
- I-2 Next 15.5 emits only `mobile-web-app-capable`; add `apple-mobile-web-app-capable` via `metadata.other`.
- I-4 no `apple-touch-startup-image` set (light + dark).
- I-6 two `theme-color` metas at runtime on staging — find the second injector.

## Acceptance
- Screenshot in an iPhone-sized viewport (memory `ritme-headless-ui-verification`) in light + dark: no content
  under the notch or home indicator.
- lint:styles and lint:dark green.
- Headless Chrome reports zero safe-area insets: verify on a real iPhone, or via CDP
  `Emulation.setSafeAreaInsetsOverride` if supported.
