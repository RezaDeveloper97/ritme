# PWA audit: Ritme web app, checked against the `pwa` skill and Lighthouse

- **Task:** T-M1-06 (investigation only, so no code was changed)
- **Date:** 2026-09-19
- **Source revision:** `stage` @ `393373c`
- **Targets audited:**
  - production `https://web.ritme.app`, which sits behind the ArvanCloud CDN
  - staging `https://stage.ritmeapp.ir`, which is behind the nginx Basic-auth/cookie gate. The credentials were read from the server at run time and are not recorded here.
  - a static read of the source
- **No local build.** The perf-baseline agent owned `frontend/.next` at the time, and both deployed stacks report `version.json` → `1.0.2`, so they were enough to audit against.

Fix-task legend:
- **07** = T-M1-07 (manifest, icons, installability)
- **08** = T-M1-08 (SW lifecycle, caching, headers)
- **09** = T-M1-09 (iOS, standalone, safe-area)

Severity:
- **High:** users see it today, or it breaks updates
- **Med:** a real gap with a limited blast radius
- **Low:** polish or hardening

---

## 1. Headline findings

1. **High · 08: every first-time visitor gets a spurious full reload about 2 s after the page loads.**
   - The SW's `activate` calls `clients.claim()`, which fires `controllerchange` on a page that had no controller.
   - `useAppUpdate` reloads on *any* `controllerchange`.
   - Mid-typing on `/signup`, this wipes the input. It is also why Lighthouse reports a same-URL "redirect" (≈3.5 s wasted) and LCP 3.5 s.
2. **High · 08: SW identity is tied to `package.json` `version` only.**
   - Two different builds can ship as the same version: prod and staging both say `1.0.2` but serve different `offline.html`.
   - When that happens, `sw.js` is byte-identical, so no `updatefound` fires, the precache (offline page, icons, manifest) is never refreshed, and the runtime cache is never cleaned.
   - The two-tier update flow is silently bypassed for every deploy that forgets the bump.
3. **Med · 08: runtime cache is unbounded and uses stale-while-revalidate for content-hashed `/_next/static/*`.**
   - The cache only shrinks on a version change.
   - SWR re-downloads immutable chunks on every use.
4. **Med · 08: prerendered HTML is sent with `Cache-Control: s-maxage=31536000` through a CDN (ArvanCloud).**
   - Arvan currently answers `x-cache: BYPASS` for HTML, but a CDN rule change would pin old HTML for a year.
   - That HTML references deleted chunks, which leads to `ChunkLoadError`. The app has no `ChunkLoadError` handler.
5. **Med · 09: no bottom safe-area on the floating tab bar and the fixed toasts/FABs.**
   - On a notched iPhone in standalone mode, the tab bar (12 px from the bottom) sits on the home indicator.
6. **Med · 07: manifest lacks `screenshots`, so Android Chrome shows the bare mini-infobar instead of the richer install sheet.**
   - It also lacks `display_override`, `shortcuts` and `categories`.
   - `start_url: "/"` costs two redirects (`/` → `/fa` → `/fa/splash`).

Things that are **fine** today:
- Installable: Chrome `Page.getInstallabilityErrors` returns `[]` on both prod and staging.
- The SW registers and controls the page.
- `/api/*`, `version.json` and `sw.js` never reach Cache Storage.
- Going offline gives the branded offline page.
- The two-tier soft/forced update UI matches the skill.
- The install prompt is hidden inside the Android shell and in any installed display mode.
- `sw.js` and `version.json` are `no-store`.
- Hashed assets are `immutable`.
- Maskable icons respect the 80 % safe zone.

---

## 2. Lighthouse (mobile)

Lighthouse **12.8.2** via `npx -y lighthouse@12`, Chrome headless, simulated throttling, `--form-factor=mobile`.

Lighthouse 12 **removed the PWA category**, so installability was checked with Chrome's own CDP calls instead: `Page.getAppManifest` and `Page.getInstallabilityErrors` (§3).

```
npx -y lighthouse@12 https://web.ritme.app/fa/signup --form-factor=mobile \
  --only-categories=performance,accessibility,best-practices,seo --output=json --output=html
# staging: same, plus --extra-headers=<file with Basic auth, deleted after the run>
```

| Metric | Prod `/fa/signup` | Staging `/fa/signup` |
| --- | --- | --- |
| Performance | **90** | **91** |
| Accessibility | **88** | **88** |
| Best practices | **100** | **100** |
| SEO | **100** | **100** |
| FCP | 1.1 s | 1.0 s |
| LCP | 3.6 s | 3.5 s |
| TBT | 10 ms | 10 ms |
| CLS | 0 | 0 |
| Speed Index | 1.1 s | 1.0 s |
| TTI | 3.6 s | n/a |

Failing or flagged audits, identical on both:

- `redirects`: "Avoid multiple page redirects, est. savings 3,570 ms". Both entries are `https://web.ritme.app/fa/signup` (same URL). This is the SW-claim reload in finding **F-SW-1**, not an HTTP redirect.
- `largest-contentful-paint-element`: 3,570 ms, for the same reason. LCP should fall to about FCP once F-SW-1 is fixed.
- `meta-viewport`: `user-scalable=no` / `maximum-scale=1`. This is a deliberate product decision (`NoZoom`) and costs accessibility points. It is recorded, not assigned.
- `color-contrast`: a `<span>` on signup. This is the known brand-palette AA debt (CLAUDE.md §10.2 `lint:dark` report). It is not a PWA issue.
- `csp-xss` (informational, "High"): "No CSP found in enforcement mode". See H-4.
- `unused-javascript` (89 KiB), `legacy-javascript` (11 KiB), `render-blocking-insight`: these belong to the perf tasks T-M1-10/11.

Raw reports are in the session scratchpad (`lh-prod-signup.report.{json,html}`, `lh-stage-signup.report.{json,html}`). They were not committed.

The logged-in screens were **not** Lighthouse-run. That needs a test-OTP account, and creating one would be a write on a server, which this investigation was not allowed to do.

---

## 3. Evidence commands

**CDP probe.** Headless Chrome, a fresh profile, a 390×844 @3x mobile viewport, driven by the scratchpad script `pwa-probe.mjs`. It performs these steps:

1. Navigate to `/fa/signup`.
2. Call `Page.getAppManifest` and `Page.getInstallabilityErrors`.
3. Wait for `serviceWorker.ready`, then reload.
4. Record `navigationPreload.getState()`, dump Cache Storage and read `storage.persisted()`.
5. Put **both the page and the SW target** offline and navigate to `/fa/signup`, `/fa/home` and `/`.

Key output (prod; staging is identical apart from the host):

```
installabilityErrors: []
manifest.errors: []   display: kFullscreen   id/startUrl/scope: https://web.ritme.app/
swReady: { scope: "https://web.ritme.app/", active: "activated", controlled: true }
navPreload: { enabled: false, headerValue: "true" }
caches:
  ritme-precache-v1.0.2: 8  (/offline.html, /manifest.webmanifest, /logo.webp, 5 icons)
  ritme-runtime-v1.0.2: 30  (6 woff2, 2 css, 20 /_next/static/chunks/*.js, /icon.png, /icons/icon-192.png)
  -> no /api/*, no /version.json, no /sw.js, no HTML documents
storage.persisted(): false
offline /fa/signup -> title "ریتمی — آفلاین", h1 "اتصال اینترنت برقرار نیست"
offline /fa/home   -> same offline page
offline /          -> same offline page
```

**First-visit reload probe.** Scratchpad script `reload-probe.mjs`, fresh profile, top-frame document requests only:

```
doc request https://web.ritme.app/fa/signup @9ms
frameNavigated https://web.ritme.app/fa/signup @1422ms
load @1746ms
doc request https://web.ritme.app/fa/signup @2064ms     <- second load, nobody asked for it
frameNavigated https://web.ritme.app/fa/signup @2267ms
load @2292ms
```

**Headers** (`curl -sS -o /dev/null -D - https://web.ritme.app<path>`):

```
/                     307 -> /fa
/fa                   307 -> /fa/splash
/fa/signup            200  cache-control: s-maxage=31536000   server: ArvanCloud  x-cache: BYPASS
/sw.js                200  cache-control: no-store, max-age=0   service-worker-allowed: /
/version.json         200  cache-control: no-store, max-age=0
/manifest.webmanifest 200  content-type: application/manifest+json  cache-control: no-cache
/offline.html         200  cache-control: public, max-age=0
/icons/icon-192.png   200  cache-control: public, max-age=0
/icon.png             200  cache-control: public, immutable, no-transform, max-age=31536000  (linked with ?hash)
/apple-icon.png       200  same as /icon.png
/favicon.ico          404  (text/html)
/_next/static/chunks/webpack-570beb4549820601.js
                      200  cache-control: public, max-age=31536000, immutable  content-encoding: gzip  x-cache: MISS -> HIT
/_next/static/chunks/nonexistent-deadbeef.js
                      404  text/plain  no-store
no Strict-Transport-Security, no Content-Security-Policy, no X-Content-Type-Options on any response
```

On staging, every nginx-exempt path gets a **second** `Cache-Control` header on top of Next's own:

```
/sw.js               cache-control: no-store, max-age=0  +  cache-control: no-store
/manifest.webmanifest cache-control: no-cache            +  cache-control: no-store
/icons/icon-192.png  cache-control: public, max-age=0    +  cache-control: public, max-age=3600   (contradictory)
```

**Deployed vs. source.** Both hosts report `1.0.2`:

```
diff <(sed 's/__APP_VERSION__/1.0.2/g' frontend/scripts/sw.template.js) prod-sw.js    -> identical
diff frontend/public/offline.html prod-offline.html
  -> prod has the old prefers-color-scheme dark block; source/staging use data-theme + ritme_theme script
diff -q frontend/public/offline.html stage-offline.html  -> identical
```

So prod and staging run **different** code under the **same** version and the **same** `sw.js` bytes.

**Head tags** (staging, current source):

```
<meta name="viewport" content="width=device-width, initial-scale=1, minimum-scale=1, maximum-scale=1, viewport-fit=cover, user-scalable=no"/>
<meta name="theme-color" content="#F2ECFF"/>
<link rel="manifest" href="/manifest.webmanifest"/>
<meta name="mobile-web-app-capable" content="yes"/>          <- Next 15.5 emits this, NOT apple-mobile-web-app-capable
<meta name="apple-mobile-web-app-title" content="ریتمی"/>
<meta name="apple-mobile-web-app-status-bar-style" content="default"/>
<link rel="icon" href="/icon.png?56c1f72568bf0662" sizes="256x256"/>
<link rel="apple-touch-icon" href="/apple-icon.png?2c4865cbee337d17" sizes="180x180"/>
(no apple-touch-startup-image links)
```

At runtime, the CDP probe found **two** `theme-color` metas on staging: `#f2ecff` and `#F2ECFF`.

**Icons** (`sips`, ImageMagick):

```
public/icons/icon-192.png       192x192 alpha   public/icons/icon-512.png  512x512 alpha (transparent bg)
public/icons/maskable-192.png   192x192         public/icons/maskable-512.png 512x512 (white bg, logo inside 80% circle ✔)
public/icons/apple-touch-icon.png 180x180 (white bg)  <- precached by SW, never linked
src/app/apple-icon.png          180x180, no alpha     <- the one actually linked; visibly different/older palette
src/app/icon.png                256x256                public/logo.png 320x320  <- source of the 512 icons (upscaled)
```

---

## 4. Rule-by-rule table

| # | Rule (skill / task scope) | Status | Sev | Evidence | Fix |
| --- | --- | --- | --- | --- | --- |
| **Manifest** | | | | | |
| M-1 | `name`, `short_name`, `description` | ok | – | `Page.getAppManifest` errors `[]` | – |
| M-2 | stable `id` | ok | – | `id: "/"` → `https://web.ritme.app/`. Keep it forever, because changing it orphans installs. | – |
| M-3 | `start_url` locale-aware, no redirect chain | gap | Low | `"/"` → 307 `/fa` → 307 `/fa/splash`. That is two redirects on every launch. Offline, the launch lands on `offline.html` either way. Use `/fa/splash` (or `/fa`) and add `?source=pwa` if analytics want it. | 07 |
| M-4 | `scope` | ok | – | `"/"` | – |
| M-5 | `display` + `display_override` | gap | Low | `display: "fullscreen"` with no `display_override`. Fullscreen hides the Android status bar and has no fallback chain. Suggest `display: "standalone"` + `display_override: ["fullscreen","standalone"]`, or keep fullscreen deliberately and document why. `InstallPrompt` already treats all modes as installed. | 07 |
| M-6 | `orientation` | ok | – | `portrait` | – |
| M-7 | RTL `dir`/`lang` | ok (partial) | Low | `dir:"rtl", lang:"fa"` is hard-coded while locales are DB rows (CLAUDE.md §6). An `en` user still gets an RTL/fa install name. Acceptable for M1 (fa is the default). Could be revisited with a per-locale manifest. | 07 (defer ok) |
| M-8 | `theme_color`/`background_color` light **and** dark | ok (limited) | Low | `#F2ECFF` both. The manifest spec has no per-scheme colours (only the experimental `user_preferences`). The app starts light by design, so the splash stays light. Nothing to fix beyond documenting it. | 07 (doc only) |
| M-9 | icons 192/512 `any` + `maskable` | ok | – | 4 icons; maskable safe zone checked visually (logo inside the 80 % circle). | – |
| M-10 | icon source quality | gap | Low | 512 px icons are upscaled from the 320 px `public/logo.png`, so the edges are soft. A ≥1024 px or vector master is needed. | 07 |
| M-11 | `screenshots` (narrow + wide) | gap | Med | Absent. Without them, Chrome Android shows the mini-infobar/basic dialog, not the richer install UI. | 07 |
| M-12 | `shortcuts` | gap | Low | Absent. Candidates: log today (`/fa/log`), calendar (`/fa/calendar`). Locale-prefixed, so fa only. | 07 |
| M-13 | `categories` | gap | Low | Absent. `["health","medical","lifestyle"]` | 07 |
| M-14 | manifest served correctly | ok | – | `application/manifest+json`, `no-cache` (prod). The staging gate exempts it (`credentials: omit`). | – |
| M-15 | favicon | gap | Low | `/favicon.ico` → 404 (HTML). Only `/icon.png` 256 is linked. Add `src/app/favicon.ico`. | 07 |
| **Service worker** | | | | | |
| S-1 | registration + scope | ok | – | `serviceWorker.ready` → scope `/`, `activated`, `controlled: true`. `Service-Worker-Allowed: /`. | – |
| S-2 | install lifecycle / no auto `skipWaiting` | ok | – | `sw.template.js` install only precaches. Activation is user-driven via `SKIP_WAITING`. | – |
| **S-3 (F-SW-1)** | first install must not reload the page | **gap** | **High** | `activate` → `clients.claim()` → `controllerchange` on an uncontrolled page → `useAppUpdate.onControllerChange` → `location.reload()`. Reload probe: a second document load at 2064 ms on a fresh profile. Lighthouse "redirects" 3.5 s / LCP 3.6 s. Fix: reload only if the page *was* controlled when it loaded, or only after the user pressed update (`apply()` sets a flag). | 08 |
| **S-4 (F-SW-2)** | every deploy ships a byte-different SW | **gap** | **High** | The SW is stamped only with `package.json` `version`. Prod and staging are both `1.0.2` with the same `sw.js` but a different `offline.html` (diff above). Unbumped deploys therefore never fire `updatefound`, never refresh the precache (users keep an old offline page and icons forever) and never clean caches. Fix: stamp a build id (git SHA or content hash of the precache list + build id) into `sw.js`. Keep `version` for the human-facing update toast, and add `buildId` to `version.json`. Optionally make `deploy.sh` refuse a deploy whose `version` equals the live one. | 08 |
| S-5 | old-cache cleanup on activate | ok (conditional) | – | Deletes every cache ≠ current PRECACHE/RUNTIME. It only runs when the version changes (see S-4). | 08 (via S-4) |
| S-6 | runtime cache bounded | gap | Med | No entry cap. Probe: 30 entries after one page, and it grows across every unbumped deploy (S-4). Add an entry cap (e.g. 150, LRU by insertion) or per-build cache names. | 08 |
| S-7 | hashed `/_next/static/*` cache-first | gap | Med | Uses SWR, so every cached immutable chunk is re-fetched in the background on every request, wasting mobile data. It should be cache-first for `/_next/static/`, with SWR kept for `/icons`, images and fonts. | 08 |
| S-8 | navigation network-first + offline fallback | ok | – | Offline probe (with the SW target also offline) → `offline.html` for `/fa/signup`, `/fa/home` and `/`. | – |
| S-9 | navigation preload | gap | Low | `navigationPreload.getState()` → `enabled: false`. Every navigation waits for the SW to boot before the network request starts. Enable it in `activate` and use `event.preloadResponse`. | 08 |
| S-10 | navigation timeout on slow networks | gap | Low | `fetch(request)` has no timeout. On a stalled connection (common with VPN/filtering, which the offline page itself mentions), the user sees a blank page, not the offline page. Consider a ~8–10 s race to the offline page. | 08 |
| S-11 | never cache `/api/*`, `version.json`, `sw.js` | ok | – | Bypass in `fetch`. Cache Storage dump contains none of them. Prod API is cross-origin (`api.ritme.app`), so it is skipped by the origin check as well. | – |
| S-12 | same-origin images on staging | note | Low | On staging `/storage/*` (admin-uploaded banners/articles) is same-origin and matches the image regex, so it is runtime-cached. It holds no health data (only admin controllers store files). This is fine once S-6 caps the cache. | 08 (via S-6) |
| S-13 | `ChunkLoadError` after deploy | gap | Med | `grep -rni chunk frontend/src` finds no handler. A tab open across a deploy loads old chunk names, and the server 404s them (`text/plain`, confirmed). Arvan and the SW runtime cache hide this only if they happen to hold the old chunk. Add a one-shot `window` `error`/`unhandledrejection` handler for `ChunkLoadError`/"Loading chunk" that reloads once (sessionStorage guard). | 08 |
| S-14 | `SKIP_WAITING` message handler | ok | – | Present, plus `CLEAR_RUNTIME_CACHE` for the forced path. | – |
| **Headers** | | | | | |
| H-1 | `sw.js`, `version.json` no-store; `Service-Worker-Allowed` | ok | – | See curl above (`next.config.ts` `headers()`). | – |
| H-2 | hashed assets immutable | ok | – | `/_next/static/*` → `public, max-age=31536000, immutable`, Arvan `HIT`. | – |
| H-3 | HTML must not be long-cached by shared caches | gap | Med | Prerendered pages send `s-maxage=31536000` through ArvanCloud. The CDN currently bypasses HTML (`x-cache: BYPASS` ×2), but that depends on a panel rule, not on our headers. Set `Cache-Control: private, no-cache` (or `s-maxage=0, must-revalidate`) for document routes in `next.config.ts`. | 08 |
| H-4 | CSP compatible with SW; transport/security headers | gap | Low | No `Content-Security-Policy`, `Strict-Transport-Security` or `X-Content-Type-Options` on any response (Lighthouse `csp-xss`: "No CSP found in enforcement mode"). Any CSP added must include `worker-src 'self'`, `manifest-src 'self'` and `connect-src` for `api.ritme.app`. Start with Report-Only. HSTS belongs in `deploy/proxy-ssl.conf`. | 08 |
| H-5 | no duplicate/contradictory `Cache-Control` on staging | gap | Low | The nginx `add_header Cache-Control` in `vhost-stage.inc` stacks on Next's header (e.g. icons `max-age=0` + `max-age=3600`). Drop the nginx ones for paths where Next already sets a value, or use `proxy_hide_header`. | 08 |
| H-6 | compression | note | Low | gzip only (no brotli) at Arvan/Next. This is outside PWA scope and is passed to T-M1-11. | (T-M1-11) |
| **iOS** | | | | | |
| I-1 | `apple-touch-icon` 180 | ok (inconsistent) | Low | `/apple-icon.png` (from `src/app/apple-icon.png`) is linked. The SW precaches a *different* `/icons/apple-touch-icon.png` that nothing links, and the two images differ visually. Keep one source (regenerated from the same master as M-10). | 07 |
| I-2 | `apple-mobile-web-app-capable` | gap | Low | Next 15.5 renders only `mobile-web-app-capable`. iOS ≥16.4 honours the manifest `display`, but older iOS needs the `apple-` meta for standalone. Add it via `metadata.other`. | 09 |
| I-3 | status-bar style, light + dark | gap | Med | `statusBarStyle: 'default'`, so the status bar is white in standalone even when the user turned dark mode on (the in-app `theme-color` rewrite does not affect the iOS standalone status bar). Use `black-translucent`. The shell already pads `env(safe-area-inset-top)` (`.app-shell`), and the status bar then shows the app's own background in both themes. Verify text contrast in light. | 09 |
| I-4 | startup images | gap | Low | No `apple-touch-startup-image` links and no `public/splash/`. iOS shows a white screen on cold launch, which clashes with the lavender/dark canvas. Needs a light + dark set for the common iPhone sizes (media queries on device-width/height/DPR + `prefers-color-scheme`). | 09 |
| I-5 | `viewport-fit=cover` + safe-area insets | gap | Med | `viewport-fit=cover` is set, but `env(safe-area-inset-*)` appears only 3× in `globals.css`: `.app-shell` top, `.osheet-body` bottom, `.osheet-foot` bottom. **Missing bottom inset:** `.tabbar` (`margin: 0 12px 12px`, the last child of the full-height shell), `.pwa-toast`/`.pwa-install` (`position: fixed; bottom: 88px`), `.cal-fab-row` and the fixed bar at `globals.css:1417` (`bottom: 88px`). On an iPhone with a home indicator (34 px) the tab bar overlaps it in standalone. | 09 |
| I-6 | theme-color per scheme | ok (minor) | Low | One `theme-color` meta in the layout (by design, app starts light) that `ThemeApplier` rewrites. The runtime probe shows **two** metas on staging (`#f2ecff`, `#F2ECFF`). Harmless because the store rewrites all of them, but T-M1-09 should find the second injector while it is in `layout.tsx`. | 09 |
| I-7 | storage eviction | note | Low | `navigator.storage.persisted()` → `false`, and `persist()` is never requested. The session token lives in `localStorage` (see T-M1-03). A Safari *tab* is subject to the 7-day script-storage cap, but a home-screen app is not. Consider `navigator.storage.persist()` after login on Chromium. Cross-reference to T-M1-03, not a PWA-task fix. | (T-M1-03) |
| **Install / update UX** | | | | | |
| U-1 | `version.json` source of truth, never cached | ok | – | Stamped by `generate-version.mjs`, `no-store`, SW bypass. | – |
| U-2 | build-time version baked in | ok | – | `NEXT_PUBLIC_APP_VERSION` from `package.json` (`next.config.ts`). | – |
| U-3 | SW channel + polling (start, visible, 15 min) + `registration.update()` on visible | ok | – | `useAppUpdate.ts`. | – |
| U-4 | numeric semver compare | ok | – | `semver.ts`. | – |
| U-5 | soft = dismissible toast; forced = blocking, focus-trapped overlay | ok | – | `UpdateGate.tsx`: `alertdialog`, Escape/Tab trapped, `z-index: 999`, `body.pwa-locked`. | – |
| U-6 | `controllerchange` reloads at most once | ok, **but** see S-3 | High | The module-level guard exists, but it also fires on the *first* claim (S-3). | 08 |
| U-7 | forced path retries if still old after reload | ok (implicit) | – | `apply()` → `reg.update()` → `reload()`. Navigation is network-first and HTML is never SW-cached, so the reload gets the new bundle. It is not an explicit retry loop, which is acceptable. | – |
| U-8 | install prompt: Chromium `beforeinstallprompt`, iOS instructions, hidden in shell/installed | ok | – | `InstallPrompt.tsx`: `isShellUserAgent`, all three installed display modes, and a dismiss that is remembered. | – |
| U-9 | install prompt in iOS in-app browsers / unguarded storage | gap | Low | The iOS guide also shows inside in-app WebViews (Instagram/Telegram) where Add-to-Home-Screen is impossible. `localStorage.getItem/setItem` is not wrapped in try/catch (throws in some private modes). | 07 |
| U-10 | toast/prompt above bottom nav + safe area | gap | Med | `bottom: 88px` fixed with no inset. Covered by I-5. | 09 |
| **Offline** | | | | | |
| O-1 | offline fallback page | ok | – | Branded, RTL, retry button, VPN hint. It is theme-aware in source (reads `ritme_theme`), but prod still serves the older `prefers-color-scheme` copy, which S-4 would have refreshed. | 08 (via S-4) |
| O-2 | which screens work offline | by design | – | **None.** Every navigation offline goes to `offline.html`: HTML is never cached and API data must never rest in Cache Storage (CLAUDE.md §11). Client-side route changes offline fail their RSC fetch → hard nav → offline page. An offline-capable shell (cached HTML shell + in-memory/IndexedDB-encrypted data) would be a product decision for a later milestone, not an M1 gap. | – |
| O-3 | offline page fonts | note | Low | `offline.html` asks for `Vazirmatn`, but the hashed `next/font` files have other names, so the page falls back to the system font unless the runtime cache happens to hold them. The screenshot renders legibly. Optional: precache one woff2 under a stable path. | 08 (optional) |

---

## 5. Scope changes applied to the fix tasks

- **T-M1-07.**
  - Added: M-3 start_url, M-5 display_override, M-11 screenshots, M-12 shortcuts, M-13 categories, M-15 favicon, M-10/I-1 icon master and single apple icon, U-9.
  - Added touches: `frontend/public/screenshots`, `frontend/src/app/favicon.ico`, `frontend/public/logo.png`.
- **T-M1-08.**
  - Named its High/Med items explicitly: S-3 first-visit reload, S-4 build-id stamping, S-6/S-7 cache policy, S-9 nav preload, S-13 ChunkLoadError, H-3 HTML cache-control, H-4 security headers, H-5 staging duplicate headers.
  - Added touches: `deploy/vhost-stage.inc`, `deploy/proxy-ssl.conf`, `frontend/public/version.json`.
  - Added a regression check: a fresh profile must produce **one** document load.
- **T-M1-09.**
  - Added: the bottom safe-area for tab bar/toasts/FABs (I-5, U-10), black-translucent status bar (I-3), `apple-mobile-web-app-capable` (I-2), light + dark startup images (I-4), duplicate theme-color (I-6).
  - Added touch: `frontend/src/app/globals.css`. This overlaps with T-M1-08 only through `useAppUpdate`/UpdateGate CSS, so run them serially or rebase.

## 6. Open items

- **Logged-in screens were not audited with Lighthouse** and not screenshotted: that needs a test-OTP account on a server (a write). Do it in T-M1-09's verification step, preferably on staging.
- **Headless Chrome cannot emulate iOS safe-area insets**, so I-5 rests on static evidence. T-M1-09 must verify on a real iPhone, or with CDP `Emulation.setSafeAreaInsetsOverride` if the installed Chrome supports it.
- **A higher-resolution logo master (≥1024 px or SVG)** is needed from design before T-M1-07 can fix M-10.
- **Decide whether `display: fullscreen` stays** (it hides the Android status bar). This is a product call for T-M1-07.
- **ArvanCloud CDN** now fronts `web.ritme.app`. `ritme-server-deployment` memory does not mention it, and its HTML-bypass rule should be documented.
