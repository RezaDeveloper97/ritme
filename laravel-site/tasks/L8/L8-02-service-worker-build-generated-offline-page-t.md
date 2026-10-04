---
id: L8-02
title: Service worker (build-generated), offline page, two-tier update, install prompt
milestone: L8
type: frontend
status: todo
depends_on: [L8-01,L1-07]
parallel_group: L8-B
touches: [resources/js/sw,resources/js/modules/pwa.js,vite.config.js,tools/build-sw.mjs,resources/views/pages/offline.blade.php,app/Http/Controllers/Pwa,app/Filament/Pages/Settings/PwaSettings.php,tests/Feature/Pwa,tests/js]
skills: [pwa]
verify: composer verify && node --test tests/js
---

# L8-02 — Service worker (build-generated), offline page, two-tier update, install prompt

## Why
PWA rules were requested; the `pwa` skill (load it) mandates a two-tier update system. A stale SW is the classic
"cached forever" bug — the SW must be generated from the build manifest, versioned, and never hand-edited.

## Scope
- `tools/build-sw.mjs` runs after `vite build`: reads `public/build/manifest.json`, writes `public/sw.js` with
  `BUILD_ID` (git sha + time) and the precache list (CSS, JS, fonts, sprite, offline page, icons). `public/sw.js` is
  in `.gitignore` and shipped in the deploy package.
- Strategies: precache shell; navigations network-first (3 s timeout) → cached page → `/offline`; `/build/*` and
  fonts cache-first (immutable); `/media/*` stale-while-revalidate with an entry cap (60) and LRU trimming; **never**
  cache `/admin*`, `/shop/cart`, `/shop/checkout`, `/shop/order/*`, `/search`, non-GET, or responses with
  `Cache-Control: no-store`.
- Two-tier update: `/pwa/version.json` (`build_id`, `min_build_id`, `message`) from admin PWA settings; soft tier →
  non-blocking «نسخه جدید آماده است» toast with reload (skipWaiting on click); forced tier (`build_id < min`) →
  blocking full-screen update view. Version checked on load + on focus (throttled).
- Install prompt: Android/Chromium `beforeinstallprompt` deferred banner (dismiss remembered 30 days), iOS Safari
  hint sheet (Share → Add to Home Screen) — both in Persian; never shown on first page view.
- `/offline` page (noindex) in the site design; registered SW only in production builds (`import.meta.env.PROD`).
- JS unit tests for routing rules and version comparison.

## Acceptance
- Offline navigation shows cached pages / offline page; a new build triggers the soft toast; raising
  `min_build_id` forces the blocking screen; admin never cached; tests green.
