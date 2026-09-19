---
id: T-M1-07
title: PWA — manifest, icons and installability fixes
milestone: M1
type: frontend
status: done
depends_on: [T-M1-06]
parallel_group: M1-D
touches: [frontend/src/app/manifest.ts, frontend/public/icons, frontend/src/app/icon.png, frontend/src/app/apple-icon.png, frontend/src/app/favicon.ico, frontend/public/screenshots, frontend/public/logo.png, frontend/src/shared/pwa/InstallPrompt.tsx]
skills: [pwa]
verify: cd frontend && npm run typecheck && npm run lint && npm run lint:styles && npm run build
---

# T-M1-07 — PWA — manifest, icons and installability fixes

## Scope
Fix the manifest/icon/install gaps listed in `docs/investigations/pwa-audit.md` (typical: stable `id`, locale-aware
`start_url`, `scope`, `display_override`, RTL `dir`/`lang`, maskable icons with safe zone, screenshots, shortcuts).
Hex literals in `manifest.ts` are baselined style-gate exceptions — don't add new ones elsewhere.

Gaps found by T-M1-06 (ids refer to the audit table):
- M-11 (Med) `screenshots` narrow + wide — needed for Chrome's richer install UI.
- M-3 `start_url` `/` costs two redirects (`/`→`/fa`→`/fa/splash`); point it at the locale route.
- M-5 `display: fullscreen` with no `display_override` — decide (product call) and add the fallback chain.
- M-12 `shortcuts`, M-13 `categories`.
- M-15 `/favicon.ico` 404 → add `src/app/favicon.ico`.
- M-10 / I-1 512px icons are upscaled from the 320px `logo.png`; two different apple-touch icons exist
  (`src/app/apple-icon.png` linked, `public/icons/apple-touch-icon.png` precached) — regenerate all from one
  ≥1024px/vector master (open item: needs the master from design).
- U-9 `InstallPrompt`: don't show the iOS guide inside in-app browsers; wrap `localStorage` in try/catch.
- M-7/M-8 (fa-only `dir`/`lang`, single light theme colour) — document as accepted, no change required.

## Acceptance
- Lighthouse installability passes; Chrome DevTools → Application → Manifest shows no warnings.
- Every gap in the audit tagged T-M1-07 is fixed or explicitly deferred with a reason.
