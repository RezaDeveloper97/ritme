---
id: T-M1-08
title: PWA — service worker lifecycle, caching and headers
milestone: M1
type: frontend
status: done
depends_on: [T-M1-06]
parallel_group: M1-D
touches: [frontend/scripts/sw.template.js, frontend/scripts/generate-version.mjs, frontend/public/offline.html, frontend/next.config.ts, frontend/src/shared/pwa/useAppUpdate.ts, frontend/src/shared/pwa/UpdateGate.tsx, frontend/public/version.json, deploy/vhost-web.inc, deploy/vhost-stage.inc, deploy/proxy-ssl.conf]
skills: [pwa]
verify: cd frontend && npm run typecheck && npm run lint && npm run test && npm run build
---

# T-M1-08 — PWA — service worker lifecycle, caching and headers

## Scope
Fix the SW/caching/header gaps from `docs/investigations/pwa-audit.md`. Edit `scripts/sw.template.js`, never
`public/sw.js`. Hard rules:
- `/api/*`, `version.json`, `sw.js` are never cached (health-data privacy, CLAUDE.md §11).
- Hashed `/_next/static/*` → cache-first + immutable; HTML → network-first with offline fallback; bounded runtime
  cache (entry cap) with cleanup of every old cache on activate.
- A deploy must never leave a user on old HTML that references deleted chunks — handle `ChunkLoadError` by a
  single reload.
- The two-tier update flow keeps working (soft toast / forced screen).

Gaps found by T-M1-06 (ids refer to the audit table), in priority order:
- **S-3 (High)** first-ever visit reloads the page ~2s after load: `clients.claim()` fires `controllerchange` on an
  uncontrolled page and `useAppUpdate` reloads on any controllerchange. Only reload if the page was controlled at
  load or the user pressed update. (Also the cause of Lighthouse's same-URL "redirect" and LCP 3.5s.)
- **S-4 (High)** SW is stamped only with `package.json` version — prod and staging are both `1.0.2` with identical
  `sw.js` but different `offline.html`. Stamp a build id / content hash into `sw.js` (+ `buildId` in
  `version.json`) so every deploy ships a byte-different worker and refreshes the precache.
- S-6 runtime cache has no entry cap; S-7 `/_next/static/*` uses SWR instead of cache-first.
- S-13 no `ChunkLoadError` handler (one-shot reload with sessionStorage guard).
- H-3 prerendered HTML is sent `s-maxage=31536000` through the ArvanCloud CDN — send `private, no-cache` for documents.
- S-9 navigation preload disabled; S-10 no navigation timeout → offline page on stalled networks.
- H-4 no CSP / HSTS / nosniff (CSP must allow `worker-src 'self'`, `manifest-src 'self'`; start Report-Only;
  HSTS in `deploy/proxy-ssl.conf`). H-5 staging nginx stacks a second, sometimes contradictory `Cache-Control`.

## Acceptance
- Manual: build v(N) → install → build v(N+1) → soft update appears and applies; offline → offline page;
  no `/api` entry in Cache Storage.
- A fresh browser profile loading `/fa/signup` produces exactly **one** document load (no claim reload).
- Two builds with the same `package.json` version still produce different `sw.js` bytes.
- `npm run build` green; headers checked with `curl -I` on staging after deploy.
