---
id: T-M1-06
title: PWA audit against the pwa skill and Lighthouse
milestone: M1
type: investigate
status: done
depends_on: []
parallel_group: M1-A
touches: [docs/investigations/pwa-audit.md]
skills: [pwa]
verify: test -s docs/investigations/pwa-audit.md
---

# T-M1-06 — PWA audit against the pwa skill and Lighthouse

## Why
The PWA exists (memory `ritme-pwa`) but was never audited end-to-end. Find every gap before fixing any.

## Scope
Audit the running production build (`npm run build && npm run start`, or staging) against the `pwa` skill's rules
and Lighthouse PWA/Best-practices checks:
- **Manifest** (`frontend/src/app/manifest.ts`): `id`, `start_url` (with locale), `scope`, `display` +
  `display_override`, `orientation`, `dir`/`lang` for RTL, `theme_color`/`background_color` for light **and**
  dark, `icons` (any + maskable, sizes), `screenshots` (narrow + wide), `shortcuts`, `categories`.
- **Service worker** (`frontend/scripts/sw.template.js` → stamped `public/sw.js`; never edit `public/sw.js`):
  install/activate lifecycle, old-cache cleanup, navigation preload, offline fallback, what is cached vs. never
  cached, cache size limits, behaviour across deploys (no mixed old-HTML/new-chunks → `ChunkLoadError`).
- **Headers** (`frontend/next.config.ts`, `deploy/vhost-web.inc`): `sw.js`/`version.json`/manifest no-store,
  immutable hashed assets, `Service-Worker-Allowed`, CSP compatible with SW.
- **iOS**: apple-touch-icon, `apple-mobile-web-app-*` meta, startup images, status-bar style, safe-area insets,
  standalone viewport, storage eviction.
- **Install/update UX**: `frontend/src/shared/pwa/*` two-tier update, install prompt per platform, no prompt inside
  the Android shell.
- **Offline**: which screens work offline, what the user sees when not.

## Acceptance
- `docs/investigations/pwa-audit.md` with a table: rule · status (ok/gap) · evidence · fix task (T-M1-07/08/09).
- Lighthouse report numbers recorded (mobile).
- Adjust the scope of T-M1-07/08/09 to match the gaps found.
