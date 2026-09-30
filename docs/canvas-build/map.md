# Map — Neshan wrapper (CB-CORE-06)

Directory (`nbl_Dir_Map`, `W_Dir_Search`) and insurance centres (`nbl_Ins_Centers`) show places on a Neshan map
(roadmap/DECISIONS.md #16). The wrapper lives in `frontend/src/shared/ui/map` and is imported as
`@/shared/ui/map`. It is domain-agnostic: callers own the data, the copy (fa/en) and the cards.

## Contract

```ts
import { isMapEnabled, NeshanMap, type MapPin } from '@/shared/ui/map';

if (!isMapEnabled()) return <PlaceList … />;          // no key → list only (flag)

<NeshanMap
  ariaLabel={t('map.label')}
  pins={pins}                                         // { id, label, lat, lng, count? }
  selectedId={selected}  onSelect={setSelected}       // map tap → null
  renderPin={(pin, selected) => <CategoryIcon … />}   // optional; default = dot / label / count
  top={<SearchField … />}                             // search + chips overlay
  controls={<ListToggle … />}                         // bottom row, beside my-location
  card={selected && <PlaceCard … />}                  // selected-card slot
  searchAreaLabel={t('map.searchArea')} onSearchArea={(bbox) => refetch(bbox)}
  myLocationLabel={t('map.myLocation')} onLocate={…} onLocateError={…}
  onUnavailable={() => setListOnly(true)}             // 'no-key' | 'load-failed'
  className="h-dvh"                                   // the map fills its box
/>
```

- **Fallback.** No key → `isMapEnabled()` is `false` and `NeshanMap` renders nothing (unit-tested in
  `map.test.ts`). An SDK that fails to load (offline, blocked, 15 s timeout) also renders nothing and calls
  `onUnavailable('load-failed')` — callers show the list in both cases. Dev and tests need no key.
- **Lazy.** The view is a `next/dynamic` chunk (`ssr: false`) and the SDK script/stylesheet are injected on the first
  mount only — nothing Neshan-related is in any route bundle or loaded without a key.
- **«جست‌وجو در این محدوده».** After a user drag/zoom (or my-location) a pill appears; tapping it calls
  `onSearchArea({ south, west, north, east })` (6 dp).
- **My location.** Only on tap, via `navigator.geolocation`; the point is handed to `onLocate` and shown as a dot. It is
  never stored, sent or logged by the wrapper (CLAUDE.md §11) — whether it goes to the API is the caller's decision.
- **Light / dark.** Overlays and pins use tokens only (`--surface`, `--line`, `--brand`, `--brand-fill`,
  `--on-brand`, `--data`, `--shadow-*`). The basemap follows `<html data-theme>`: Neshan's `neshanVector` (light) and
  `neshanVectorNight` (dark), switched live with `setMapType`. There is no custom N&B map style (Neshan offers only its
  own four styles); POI and traffic layers are off so our pins read clearly.
- **SDK.** Pinned in `frontend/src/shared/config/map.ts` (`NESHAN_SDK`): mapbox-gl 1.13.2 + neshan-sdk 1.1.5 from
  `static.neshan.org`. Note the SDK itself writes the map key into `localStorage['neshan-map-auth']`.

## Environment

| Var | Where | Notes |
|---|---|---|
| `NEXT_PUBLIC_NESHAN_KEY` | **build-time** env of the frontend image on the server (stage / prod) | A Neshan *web map* key (panel: platform.neshan.org → API keys → type «وب/Map»). Inlined into the client bundle — that is how Neshan web keys work — so restrict it to the app's domains (`web.ritme.app`, `stage.ritmeapp.ir`, …) in the Neshan panel. Never commit it, never put it in `.env*` files in the repo, never log it (DECISIONS #18). Empty/blank = no key. |

Because it is `NEXT_PUBLIC_*`, changing the key needs a frontend **rebuild**, not just a restart. Pass it as a docker
build arg / env in the server's compose file next to `NEXT_PUBLIC_API_BASE_URL`.

## CSP (required before the map works on stage/prod)

`frontend/next.config.ts` enforces a CSP that blocks Neshan today. When the first screen that mounts the map ships
(CB-DIR-07 / CB-INS-06 — not this task: `next.config.ts` is outside CB-CORE-06's `touches`), add, preferably only
when `NEXT_PUBLIC_NESHAN_KEY` is set:

| Directive | Add | Why |
|---|---|---|
| `script-src` | `https://static.neshan.org` | SDK script, RTL-text plugin |
| `style-src` | `https://static.neshan.org` | SDK stylesheet |
| `connect-src` | `https://*.neshan.org` (at least `vts`, `tvts`, `tile1`–`tile8`, `static`, `api`) | vector tiles, styles, glyphs (`/fontstack/…pbf`), sprites, key check |
| `worker-src` | `blob:` | mapbox-gl spawns its tile worker from a blob URL |
| `child-src` | `blob:` | same, for older WebKit which reads `child-src` for workers |
| `img-src` | already `https:` + `data:` + `blob:` | sprites / watermark — nothing to add |

Verify in headless Chrome against a production build (`npm run build && npm start` with a real key in the shell only)
that the console shows no CSP violations while panning, zooming and switching theme.

## Where it is used

| Board | Task | Usage |
|---|---|---|
| `nbl_Dir_Map` | CB-DIR-07 (bbox API: CB-DIR-02) | full-screen map: `top` = search + chips, `controls` = «فهرست N مجموعه», `card` = selected place, clusters via `count` |
| `nbl_Ins_Centers` | CB-INS-06 | small inline map card above the list (`className="h-40 rounded-3xl"`, no `card`), «نقشه» expands it |
| `W_Dir_Search` | CB-DIR-11 | desktop split view |

Visual fidelity against these boards is checked in the DIR / INS tasks, not here.
