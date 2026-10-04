# L8-03 — PWA verification (offline, update tiers, installability)

One command, repeatable, against the **production build** (`npm run build` = `vite build && node tools/build-sw.mjs`):

```bash
node tools/pwa-check.mjs            # uses the existing production build
node tools/pwa-check.mjs --build    # runs `npm run build` first
```

The tool starts its own `php -S` on a free port with Laravel's router (what `php artisan serve` runs), drives headless
system Chrome over CDP (Node built-ins only, same plumbing as `tools/shot.mjs`), stops only its own server, and writes
`pwa-check.json` + 390-wide screenshots here (PNGs are git-ignored). Exit 1 on any failed check.

Result on 2026-10-04, build `20261004154441-bba02217`: **23 passed, 0 failed, 0 warnings.**

## How each scenario is produced

| Scenario | How |
|---|---|
| Offline | `Network.emulateNetworkConditions {offline: true}` on the page **and** the service-worker target (auto-attached before it runs). The "unvisited → /offline" result shows the worker's own `fetch()` really failed. |
| Soft update, worker channel | The tool's router answers `/sw.js` byte-changed (built file + a comment) — a redeploy with no build and no repo change — then `registration.update()`. |
| Soft update, polling channel | The running bundle's `__BUILD_ID__` is rewritten on the wire (CDP `Fetch`) to `20000101000000-pwacheck`. The real `/pwa/version.json` reports the deployed build, which is newer. |
| Forced update | `PwaSettings.min_build_id` is set to the deployed build through `UpdateSettings` (`artisan tinker`). The cache is invalidated by the observer, as it is from the admin. The client is an "old tab": its worker is already installed and `Network.setBypassServiceWorker` serves it the rewritten old bundle. `min_build_id` is put back to its previous value (`null`) in a `finally`, then read back to confirm. |
| External / CSP | `Network.requestWillBeSent` on every page and worker target, plus a `securitypolicyviolation` probe injected before page scripts and `Log`/console CSP reports. CSP is **not** bypassed. |

## Checks

| Area | Check | Result | Evidence |
|---|---|---|---|
| build | Production build present, no `public/hot`, `public/sw.js` stamped with `build-id.json` | ✔ | both `20261004154441-bba02217`; 33 precache URLs |
| http | `/manifest.webmanifest` 200 `application/manifest+json` | ✔ | |
| http | `/pwa/version.json` no-store, no cookie, deployed id | ✔ | `max-age=0, no-store, private`; `{"build_id":"20261004154441-bba02217","min_build_id":null}` |
| http | `/sw.js` JavaScript with the build id | ✔ | 4.7 kB |
| http | `/offline` 200, `noindex`, retry button | ✔ | |
| manifest | `Page.getAppManifest`: no errors. Has name, short_name, id, start_url, standalone, colours, 192/512 + maskable icons | ✔ | «ریتمی», `start_url /?source=pwa`, 5 icons |
| manifest | `Page.getInstallabilityErrors` empty | ✔ | `[]` (needs a non-incognito profile; an incognito context reports `in-incognito`) |
| sw | Registers (scope `/`, `updateViaCache: none`), activates, controls the page | ✔ | |
| precache | `ritme-shell-<build>` holds every URL `build-sw.mjs` derives from the Vite manifest, incl. `/offline` + icons, and no admin entries | ✔ | 33/33 present |
| sw | Visited pages stored in `ritme-pages-<build>` | ✔ | `["/","/cycle"]` |
| never | `/admin/login`, `/shop/cart` and `/pwa/version.json` online: not answered by the worker and in no `ritme-*` cache | ✔ | `fromServiceWorker=false` |
| offline | Visited `/cycle` comes from the page cache | ✔ | `fromServiceWorker=true`, same title as online — `offline-visited-390.png` |
| offline | Unvisited `/about` shows the precached `/offline` page | ✔ | title «اتصال برقرار نیست», retry button visible — `offline-unvisited-390.png` |
| offline | `/admin/login` and `/shop/cart` are not served by the worker | ✔ | `net::ERR_INTERNET_DISCONNECTED`: no cached copy and no `/offline` |
| install | `beforeinstallprompt` → own banner on a later page view | ✔ | `install-banner-390.png` |
| soft | Worker channel: a redeployed sw.js installs and waits, and the toast appears | ✔ | `soft-sw-channel-390.png` |
| soft | «به‌روزرسانی» → SKIP_WAITING → exactly one reload | ✔ | 0 extra reloads, no waiting worker afterwards |
| soft | Polling channel: newer deployed build → toast «نسخه جدید آماده است» | ✔ | `soft-toast-390.png` |
| soft | Toast can be dismissed («بعداً») and never makes the page inert | ✔ | |
| forced | `min_build_id` raised → blocking screen | ✔ | `forced-390.png` |
| forced | Escape and Back cannot leave the screen; the rest of the page is inert; focus stays inside | ✔ | |
| forced | The button reloads. Because the bundle is still old, the screen comes back | ✔ | |
| forced | `min_build_id` restored | ✔ | `null` |
| external | No request to a non-local origin from any page or worker | ✔ | 0 |
| csp | No CSP violation on any page | ✔ | 0 |

## Observations (not failures)

- **The forced screen auto-applies on a first visit.** If a page shows the forced screen while its first worker is
  still installing, `pwa.js` (`watchWorker` → `applyForced`) reloads it as soon as the worker is installed. This
  happened in an early run where the old-bundle tab had no worker yet. It is the intended "never stay on an old build"
  behaviour. It is also why the forced check uses a tab whose worker is already installed.
- **Query-string mismatch on icons.** The layout asks for icons with `?v=` (e.g. `/icons/icon-192.png?v=b2a80f2f`).
  The precache stores them without the query, so cache-first misses the precached copy. The icon is fetched and cached
  a second time, and before that first online fetch it is not available offline. This only costs a little cache space.
- **Admin CSS in the public cache.** After `/admin/login` is opened, the admin theme CSS
  (`/build/assets/theme-*.css`) ends up in `ritme-shell-<build>`. The admin navigation itself bypasses the worker
  (✔ above), but the admin page is still a client inside scope `/`, so its `/build/*` subresources take the static
  cache-first route. The file is hashed and immutable, so this is harmless. It is noted for L8-02 in case admin assets
  should stay out of the visitor cache entirely.
- **No `Cache-Control` on `/sw.js` here.** Under `php -S` the header is absent; on the real host it comes from
  `public/.htaccess`. Chrome ignores the HTTP cache for worker scripts anyway (`updateViaCache: none`).
- **Builds by other agents.** Concurrent `npm run build`s change the build id. The tool compares `build-id.json` at
  the start and end of the run and fails with "rerun" if it changed.
