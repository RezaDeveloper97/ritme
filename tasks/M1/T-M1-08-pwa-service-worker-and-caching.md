---
id: T-M1-08
title: PWA — service worker lifecycle, caching and headers
milestone: M1
type: frontend
status: todo
depends_on: [T-M1-06]
parallel_group: M1-D
touches: [frontend/scripts/sw.template.js, frontend/scripts/generate-version.mjs, frontend/public/offline.html, frontend/next.config.ts, frontend/src/shared/pwa/useAppUpdate.ts, frontend/src/shared/pwa/UpdateGate.tsx, deploy/vhost-web.inc]
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

## Acceptance
- Manual: build v(N) → install → build v(N+1) → soft update appears and applies; offline → offline page;
  no `/api` entry in Cache Storage.
- `npm run build` green; headers checked with `curl -I` on staging after deploy.
