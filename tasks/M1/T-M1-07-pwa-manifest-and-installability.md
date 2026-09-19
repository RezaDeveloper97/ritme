---
id: T-M1-07
title: PWA — manifest, icons and installability fixes
milestone: M1
type: frontend
status: todo
depends_on: [T-M1-06]
parallel_group: M1-D
touches: [frontend/src/app/manifest.ts, frontend/public/icons, frontend/src/app/icon.png, frontend/src/app/apple-icon.png, frontend/src/shared/pwa/InstallPrompt.tsx]
skills: [pwa]
verify: cd frontend && npm run typecheck && npm run lint && npm run lint:styles && npm run build
---

# T-M1-07 — PWA — manifest, icons and installability fixes

## Scope
Fix the manifest/icon/install gaps listed in `docs/investigations/pwa-audit.md` (typical: stable `id`, locale-aware
`start_url`, `scope`, `display_override`, RTL `dir`/`lang`, maskable icons with safe zone, screenshots, shortcuts).
Hex literals in `manifest.ts` are baselined style-gate exceptions — don't add new ones elsewhere.

## Acceptance
- Lighthouse installability passes; Chrome DevTools → Application → Manifest shows no warnings.
- Every gap in the audit tagged T-M1-07 is fixed or explicitly deferred with a reason.
