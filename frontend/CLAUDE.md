# Ritme (ریتمی) — Project Guide for Claude Code

> This file is loaded into your context on every session. Treat it as the
> source of truth for how code is written in this repository. When a request
> conflicts with these rules, follow the rules and flag the conflict.

---

## 1. What this project is

**Ritme** (Persian: **ریتمی**) is a women's health application focused on two
domains:

- **Cycle mode** — menstrual cycle tracking, period & ovulation predictions,
  symptom and mood logging, insights.
- **Pregnancy mode** — week-by-week pregnancy timeline, due-date tracking,
  stage-appropriate content and reminders.

The app switches between these two modes; `mode` is a first-class domain
concept, not a feature flag bolted on later.

**Primary audience:** Persian-speaking (Iranian) users. Persian (`fa`) is the
default locale and the default text direction is **RTL**. English (`en`) ships
alongside it — but the app is **multi-language by design**: the set of locales
lives in the backend's `languages` table and admins add more (with their own
RTL/LTR direction) without a code change. See §6; nothing may hardcode a
locale list.

**Product character:** This is intimate, personal health software. Code, copy,
and UX must be **private by default, respectful, and medically careful**. See
§11 (Sensitive-domain rules) — these are not optional.

---

## 2. Tech stack

| Concern            | Choice                                                        |
| ------------------ | ------------------------------------------------------------- |
| Framework          | **Next.js (App Router + React Server Components)**            |
| Language           | **TypeScript (strict mode)** — no `any`, no implicit `any`    |
| UI                 | **React** function components + hooks only                    |
| Styling            | **Tailwind CSS** with **logical** utilities (`ms-`, `pe-`, …) |
| i18n               | **next-intl** (ICU MessageFormat, per-slice namespaces)       |
| Server state       | **TanStack Query** (`@tanstack/react-query`)                  |
| Client state       | **Zustand** (UI/ephemeral state only)                         |
| Forms + validation | controlled state + **zod** (no form library)                 |
| Dates / calendar   | **Locale-aware** (Jalali for `fa`, Gregorian for `en`) via a centralized date layer (§7) |
| HTTP               | single shared thin `fetch` client in `shared/api` (no axios)  |

> **TODO for the team:** pin exact versions in `package.json` and fill in the
> DB/ORM, auth provider, and analytics choices below before relying on them.
>
> - DB / ORM: `<fill in>`
> - Auth: `<fill in>`
> - Analytics / logging: `<fill in — see §11 for what must NEVER be logged>`

---

## 3. Architecture: Feature-Sliced Design (FSD)

This project uses **Feature-Sliced Design**. It is enforced, not aspirational.
The whole point is to keep coupling under control as the codebase grows, so the
import rules below matter more than any individual file's contents.

### 3.1 Layers (top imports from bottom — never the reverse)

```
src/
├── app/        # init: providers, routing, global styles, i18n + query setup
├── screens/    # route-level composition (one full screen per slice)
│                # NB: this FSD layer is named `screens`, NOT `pages`, so that
│                # Next.js does not mistake it for its legacy Pages Router.
├── widgets/    # large self-contained blocks (CycleCalendar, AppHeader)
├── features/   # user actions that deliver value (log-period, switch-mode)
├── entities/   # domain nouns (cycle, symptom, pregnancy, user, article)
└── shared/     # domain-agnostic foundation (ui kit, lib, config, api client)
```

A layer may **only import from layers strictly below it**:

- `screens` → may use `widgets`, `features`, `entities`, `shared`
- `widgets` → may use `features`, `entities`, `shared`
- `features` → may use `entities`, `shared`
- `entities` → may use `shared`
- `shared` → may use **nothing else in `src`** (fully independent)

### 3.2 Slices & segments

Inside `entities`, `features`, and `widgets`, code is grouped into **slices**
(one folder per domain concept), and inside each slice into **segments**:

```
entities/cycle/
├── ui/         # presentational components (CycleBadge, CycleDayCell)
├── model/      # types, zustand stores, pure domain logic
├── api/        # requests for this entity + query hooks
├── lib/        # helpers specific to this slice
└── index.ts    # PUBLIC API — the only legal entry point to this slice
```

`shared` and `app` are segment-organized but have no slices.

### 3.3 The three rules that must never be broken

1. **Import downward only.** A higher layer never reaches into a lower one's
   internals beyond its public API, and a lower layer never imports a higher
   one. If you feel the urge to import upward, the code is in the wrong layer.
2. **No cross-imports between sibling slices.** `entities/cycle` must NOT import
   from `entities/symptom` directly. If two entities need to interact, compose
   them one layer up (in a `feature` or `widget`), or model the relationship
   explicitly. This is the rule people break first — don't.
3. **Import only from a slice's `index.ts`.** Never
   `import { foo } from '@/entities/cycle/model/foo'`. Always
   `import { foo } from '@/entities/cycle'`. The public API is the contract;
   everything not exported there is private and may change freely.

### 3.4 Enforcement

These rules are checked by tooling, not goodwill. Run the boundary linter before
considering work done (see §9). If you add a new slice, make sure its `index.ts`
exposes exactly what callers need and nothing more.

---

## 4. Where does new code go? (quick decision guide)

Before writing a component or module, decide its layer:

- Is it a **domain noun** with its own data/shape (a cycle, a symptom, a
  pregnancy, an article)? → `entities/<noun>`
- Is it a **user action that produces a result** (log a period, switch mode,
  set a reminder, change language)? → `features/<verb-noun>`
- Is it a **large composite block** assembled from features/entities and
  meaningful on its own (the calendar, the insights panel, the header)? →
  `widgets/<block>`
- Is it a **full screen** tied to a route? → `screens/<screen>` (then mounted by
  an App Router route in `app/`)
- Is it **domain-agnostic and reusable anywhere** (Button, Modal, the date
  helper, the http client)? → `shared/<segment>`

When in doubt, push it **down**, not up. Code is cheaper to promote later than
to untangle.

### 4.1 Routes vs sheets — where a screen appears

Night & Bloom rule (B-N1-04, route map: `docs/night-bloom/routes.md`): a
screen with **a back-button header and its own scroll is a route**; a **short
decision, picker or quick form is a sheet** — a panel that rises from the
bottom over whatever the user was already looking at. Routes live directly
under `src/app/[locale]/` (there is no `(app)` group): the tab roots (`/home`,
`/calendar`, `/services`, `/profile`, `/pregnancy`, `/pregnancy/weeks`), the
sign-up + onboarding flow (`/splash`, `/welcome`, `/signup`, `/otp`,
`/onboarding/*`) and the sub-screens with a back header (`/cycle`, `/log`,
`/checkups/*`, `/reminders/*`, `/fertility/*`, `/pregnancy/*`, and every NEW
route in routes.md). Don't add a route for something that is a short decision.

**Bottom nav** (`widgets/bottom-nav`, spec `docs/night-bloom/nav.md`):
امروز · mode tab · FAB · خدمات · من, where the mode tab follows the life-stage
(تقویم / باروری / بارداری / کودک / علائم) and the FAB opens `?sheet=log`. It
renders only on the paths in `shared/config/app-nav.ts` (tab roots and
first-level hubs) — screens may keep mounting `<BottomNav />`, it stays out on
back-header screens by itself. A new hub adds its path there. It floats over
the screen, so the screen's `.scroll` gets a clearing tail automatically.

**Sheets:**

- **Open one:** `openSheet('<id>', arg?)` from `@/shared/sheet`. The id comes
  from `src/app/sheets/registry.tsx`; `arg` is one short, non-sensitive string
  (an info topic, an article slug) — it rides in the query string, so never a
  cycle phase or anything else covered by §11.
- **Add one:** write the screen in `screens/<name>` exporting a `*Sheet`
  component that renders **content only** — no `.view`, no `.hdr`/`NavBack`, no
  `<BottomNav />`. The sheet chrome (grip, title, close, scroll) belongs to
  `AppSheet`. Then add an entry to `SHEET_REGISTRY` with its `size` and `Title`.
- **The registry is the one place allowed to import `screens` from `app`.**
  `shared/sheet` only ever deals in string ids, so the FSD direction holds.

Sheets are addressed by `?sheet=<id>` and stack, so the system Back dismisses
the top one and deep links work (§4.2). Navigating to another route closes
them.

**The two sizes are different layouts, not two numbers** (see `AppSheet`):

| Size | Height | Scrolling |
| ---- | ------ | --------- |
| `half` | `auto`, from a 42% floor — **the content sets it** | Never. More content makes the panel *taller*. Only if it would outgrow the screen does the body become scrollable, as a last resort. |
| `full` | fixed, 92% of the shell | The body scrolls inside the panel. |

Pick `half` for one decision or a short form (a picker, a log category), `full`
for long copy or a list. A `half` sheet with `max-height` on its own content is
a bug: it re-introduces the inner scrollbar the size exists to avoid.

`AppSheet` is also the primitive for **inline** sheets that aren't routed at all
(`QuickEditSheet`, the calendar month picker, the log `CategorySheet`) — pass
`open`/`onClose` yourself. There is no second sheet implementation; don't add
one.

### 4.2 Back goes back one layer

The system Back (gesture or three-button bar — browser, installed PWA and the
TWA alike) steps back **exactly one layer**:

1. the soft keyboard, if it is up — Android's IME takes that press itself;
2. otherwise the top sheet / modal — only that, the screen stays;
3. otherwise the previous screen — and out of the app, the normal Android way,
   when there is no previous screen in the app.

Reference: home → `/cycle` → log sheet. Back closes the sheet (still on
`/cycle`); Back again → home; Back on home with nothing behind it → Android.

In a TWA there is no reliable JavaScript "back button" event, so every layer is
a **history entry** (`shared/back-stack`, doc header of `stack.ts`):

- **Inline sheets** (`AppSheet` with your own `open`/`onClose`) push one entry
  while open — automatically, via `useBackLayer`. Back pops it and calls
  `onClose`; ×, swipe, backdrop, Escape and a save flip `open` off and the hook
  pops the entry itself. The handler of a Back-close **never** calls `back()`
  again — that is what used to take the user off the screen too.
- **URL sheets** (`?sheet=`) keep their own entry (`shared/sheet/controller`);
  `SheetHost` passes `backEntry={false}` so `AppSheet` doesn't add a second one.
- A new kind of overlay (a custom dialog) calls `useBackLayer(open, close)`.
  Never listen for `popstate` or a hardware-key bridge yourself.

Consequences for new code:

- **The on-screen back arrow** is `router.back('/parent')` from `@/shared/i18n`,
  never `router.push('/parent')`: it walks history when the parent is right
  behind (so the system Back afterwards doesn't bounce here) and replaces the
  screen with the parent otherwise — a deep link from a notification still has
  a visible way home, and the system Back still leaves the app from there.
  Next's own `router.back()` is not exposed: its destination isn't known.
- **Unsaved input must survive a Back.** Back closes a sheet like × does, so a
  form either keeps its draft outside the component (the log sheet:
  `features/log-day/model/drafts.ts`, memory only, wiped at session end) or
  asks before discarding.
- **Navigating from inside an inline sheet**: use `router.push` / `<Link>` from
  `@/shared/i18n`; they mark the navigation so the closing sheet doesn't pop
  its entry under the router's feet (it is skipped on the way back instead).
- **Sign-out is a full document replace** (`window.location.replace`), and
  `SessionGuard` re-checks a screen restored by Back or the bfcache, so no
  signed-in screen survives a session.
- **Test on the installed TWA**, with gesture *and* three-button navigation —
  not just a Chrome tab. Checklist: `docs/back-navigation.md`.

---

## 5. Domain model (current entities)

These are the canonical slices. Reuse them; don't reinvent parallel versions.

| Slice                  | Responsibility                                            |
| ---------------------- | --------------------------------------------------------- |
| `entities/user`        | profile, preferences, current `mode` (cycle/pregnancy)    |
| `entities/cycle`       | menstrual cycle records, period days, phase calculations  |
| `entities/symptom`     | logged symptoms/mood/flow, their types and display        |
| `entities/pregnancy`   | pregnancy record, week/trimester, due-date math           |
| `entities/article`     | educational health content (localized, stage-aware)       |

Representative higher-layer slices:

- **features:** `log-period`, `log-symptom`, `switch-mode`, `switch-locale`,
  `predict-cycle`, `set-reminder`
- **widgets:** `cycle-calendar`, `symptom-tracker`, `pregnancy-timeline`,
  `insights-panel`, `app-header`
- **screens:** `onboarding`, `home`, `calendar`, `insights`, `learn`, `profile`

---

## 6. Internationalization (i18n)

i18n is part of the architecture, not a translation afterthought.

### 6.1 The rule: locales are data, never code

**The set of languages the app ships lives in the backend's `languages` table
and is edited from the admin panel (`/admin/languages`). No code anywhere may
enumerate locales.** Persian (`fa`) is the seeded default and renders RTL;
English (`en`) is the second seeded row. Neither is special-cased — they are
rows, and an admin can add, deactivate, reorder or re-default any of them.

Concretely, this is forbidden:

```ts
// ❌ never — breaks the moment an admin adds a language
const LOCALES = ['fa', 'en'];
const isRtl = locale === 'fa';
const label = locale === 'fa' ? 'فارسی' : 'English';
```

and this is how each of those is written instead:

```ts
// ✅ the live list, resolved at runtime
const { codes, defaultLocale } = await getLocaleRegistry();   // server / middleware
const { data: languages } = useLanguages();                    // client

// ✅ direction is a property of the language row, not of "is it Persian"
const isRtl = useDirection() === 'rtl';

// ✅ the endonym comes from the registry
const label = languages.find((l) => l.code === locale)?.name;
```

The one deliberate exception is **Persian digits** (`۱۲۳`) and the **Jalali
calendar**, which really are Persian-specific: `locale === 'fa'` is correct
there, and `shared/lib/date` documents the fallback every other locale takes
(Gregorian, Latin digits, English month names) until data for it is added.

### 6.2 How a new language reaches the app

1. An admin creates it in `/admin/languages` with a code, endonym, English name
   and **direction (RTL/LTR)**.
2. The backend generates that locale's UI-string files under
   `storage/app/translations/<code>/`, copied from the default language so the
   app is fully usable immediately, plus its `lang/<code>/` PHP files and its
   own copy of the editable smart-message rows (unapproved, pending review).
3. The admin translates strings namespace by namespace in
   `/admin/languages/<id>/translations`.
4. The frontend reads `GET /languages` and `GET /languages/<code>/messages` at
   runtime. **No frontend rebuild is needed** — the new locale gets its URL
   prefix, `<html dir>`, message bundle and an entry in the language picker.

Every content form in the admin panel grows an input for the new locale
automatically, because they all render `<x-admin.translatable>` and validate
through `App\Support\Translatable::rules()`.

### 6.3 Fallback, everywhere

A key or field nobody has translated yet renders **the default language's
text**, never a blank. This holds at every layer — `TranslationStore`,
`Translatable::pick()`, the frontend's message merge — so a language is safe to
add before it is finished, and a key added to the app tomorrow doesn't break the
languages added yesterday.

Consequently **only the default language is `required`** on an admin content
form; every other locale's input is optional.

### 6.4 Mechanics

- **Library:** `next-intl`, integrated with the App Router and RSC so strings
  render on the server and don't bloat the client bundle. Its *routing*
  middleware is deliberately NOT used — it needs the locale list fixed at build
  time. `src/middleware.ts` and `shared/i18n/navigation` do the prefixing
  instead, off the runtime registry.
- **Locale lives in the URL path** (`/fa/...`, `/en/...`, `/ar/...`) for SEO and
  shareable links. An unprefixed URL is redirected using the `NEXT_LOCALE`
  cookie, else the default language.
- **Bundled floor.** `fa` and `en` messages are compiled in
  (`shared/i18n/bundled`) so the app renders during a backend outage and at
  build time. `frontend/messages/**` is the source of truth for them — after
  editing keys there, copy them into the backend seed so languages created
  later inherit the new keys: `cp frontend/messages/<code>/*.json
  backend-go/resources/translations/<code>/` for each bundled locale (embedded
  in the Go binary), plus `php artisan translations:import` in `backend/`
  while production still runs Laravel.
- **Namespaces per slice.** Each feature/widget owns its message namespace and
  loads it lazily. Do NOT dump every string into one giant `common.json`.
- **ICU MessageFormat** for plurals, gender, and number/date formatting. Persian
  has different plural rules than English — write messages with ICU, never
  string-concatenate translated fragments.
- **Never hardcode user-facing text** in components — every visible string goes
  through the translator.

```tsx
// ✅ correct
const t = useTranslations('cycle.calendar');
return <h2>{t('nextPeriodIn', { days: count })}</h2>;

// ❌ never
return <h2>دوره بعدی تا {count} روز دیگر</h2>; // hardcoded, untranslatable
```

### 6.5 Direction

Prefer CSS **logical** properties (`ms-`, `pe-`, `text-start`) so layout follows
`<html dir>` with no JavaScript. Reach for `useDirection()` only where the
direction has to enter JS — swipe/transform math, a chevron that must point
"forward". Never branch on the locale code for this.

---

## 7. Dates & calendar (locale-aware — critical)

Iranian users think in the **Jalali (Shamsi)** calendar; English-speaking users
think in the **Gregorian** one. Getting this wrong is a correctness bug in a
product whose entire job is tracking dates.

- **The calendar follows the locale.** `fa` → Jalali, every other locale →
  Gregorian — everywhere, with no exceptions: the home mini-calendar, the
  calendar screen, birthday and last-period wheels, month labels, weekday
  column order (Saturday-first in Jalali, Sunday-first in Gregorian) and digits
  (Persian in `fa`, Latin elsewhere). Raw ISO/Gregorian strings may exist
  internally and at the API boundary, but are never shown raw to the user.
- **Calendar data is hand-written, so it lags the language list (§6).** Month
  and weekday names exist for `fa` and `en`; a language an admin adds reads its
  dates through the Gregorian/English tables until names for it are added to
  `shared/lib/date`. That fallback is explicit in `calendarLocale()` — never
  index a name table by a raw `Locale`, or a new language renders `undefined`.
- **Calendar *parts* are meaningless without their locale.** `toParts`,
  `todayParts`, `monthMatrix`, `daysInCalendarMonth` and `partsToDate` all take
  a `Locale`; anything that *persists* parts (e.g. the onboarding store) must
  persist the locale beside them and re-express them with `convertParts` when
  the user switches language.
- **One centralized date layer:** `shared/lib/date`. Wrap the date library
  (e.g. `dayjs` + a Jalali plugin) there and expose helpers like
  `formatLongDate()`, `toParts()`, `addDays()`, `diffInDays()`. Components and
  features import **only** from `shared/lib/date` — never call the underlying
  library or `new Date().toLocale...` directly anywhere else.
- **Persian digits** are a formatting concern handled in that layer (and/or via
  ICU number formatting), not sprinkled around components.
- Cycle/pregnancy math (cycle length, predicted period, gestational week, due
  date) lives in the relevant **entity `model/`** as pure, unit-tested
  functions — independent of React, locale, and the date library's surface.

---

## 8. State management

Keep the two kinds of state strictly separate — most state bugs come from
blurring them.

- **Server state** (anything fetched from the API: cycles, symptoms, articles)
  → **TanStack Query**. Do not copy server data into Zustand or React state.
  - Co-locate query hooks in the slice's `api/` segment.
  - Use a **query-key factory** per entity to keep cache keys consistent:
    ```ts
    export const cycleKeys = {
      all: ['cycle'] as const,
      list: (f: CycleFilters) => [...cycleKeys.all, 'list', f] as const,
      detail: (id: string) => [...cycleKeys.all, 'detail', id] as const,
    };
    ```
  - After a mutation, invalidate via the factory — never hand-write key arrays.
- **Client state** (current `mode`, open sheets/modals, multi-step form
  progress, theme) → **Zustand** (or local `useState` when it's truly local).
  Stores live in `shared` (cross-cutting) or in a slice's `model/` (slice-local).

### 8.1 Backend API

The backend is a separate service: **`backend-go/`** (Go + Fiber v3) is the
backend for all new work and serves staging; production still runs the frozen
Laravel `backend/` until the cutover (T-M2-27). Both answer the same `/api/v1`
contract byte for byte, so the frontend never branches on which one it talks
to. New or changed endpoints are added in backend-go (`new-endpoint` skill),
not in `backend/`. **The API is documented as an OpenAPI spec** you should
treat as the source of truth for endpoints, request/response shapes, enums,
and auth:

- **Spec:** `backend-go/api/openapi.yaml` in the repo (OpenAPI 3.1,
  hand-maintained, checked against every route by a Go test), served at
  `/docs/openapi.yaml` behind Basic auth. Production's Laravel copy is still at
  `https://api.ritme.app/docs/api-docs.json` until cutover. The host serves
  `https` (TLS terminates at the nginx proxy; `http` 301-redirects). **Never
  point the app at an `http://` API** — the PWA is served over https, so an
  http API call is mixed content and the browser blocks it (local dev on
  `http://127.0.0.1:8020` is the one exception).
- **Local API:** backend-go on `http://127.0.0.1:8020/api/v1` — set
  `NEXT_PUBLIC_API_BASE_URL` in `.env.local` (never in a prod build). How to
  run it, seed it and log in without SMS: the `local-dev` skill.
- **Base URL:** `https://api.ritme.app/api/v1/`. The frontend is served from a
  **different origin** (`web.ritme.app`), so every browser call is cross-origin
  and depends on the API's CORS allow-list (`CORS_ALLOWED_ORIGINS` in
  backend-go; `backend/config/cors.php` on production until cutover). Adding a
  new frontend origin means adding it to both. The admin panel lives on a
  third hostname, `adpanell.ritme.app`.
- **Auth:** JWT **bearer** token in the `Authorization` header
  (`Authorization: Bearer <token>`). Login is **OTP-based**: `POST
  /auth/send-otp` → `POST /auth/verify-otp` returns the access token. The shared
  `fetch` client in `shared/api` attaches the token; never scatter auth
  handling across slices.

Endpoint groups (map these onto the FSD entity/feature slices — don't invent
parallel ones):

| Group          | Purpose                                                    |
| -------------- | ---------------------------------------------------------- |
| `auth`         | `send-otp`, `verify-otp`, `logout`, current `user`         |
| `profile`      | get / create-update user profile                           |
| `cycle`        | today / by-date / by-month calculations, status, recalc, enums, matrix messages |
| `health-logs`  | daily health log CRUD + form enums                         |
| `messages`     | daily personalized messages, current mode, enums           |
| `pregnancy`    | activate/confirm/deactivate mode, onboarding, profile, weekly content & logs, symptoms, fetal-movement, alerts, enums |

Conventions to mirror from the spec: many resources are **keyed by date**
(`/health-logs/{date}`, `/pregnancy/symptoms/{date}`) or **by week**
(`/pregnancy/weekly/{week}`, `/pregnancy/content/{week}`), and several groups
expose an **`/enums`** endpoint that drives form options — fetch those via
TanStack Query and derive types from them rather than hardcoding option lists.
Dates cross this boundary in the API's format; convert to the display calendar only in
`shared/lib/date` for display (§7). Remember §11: never log health payloads.

---

## 9. Commands

> Adjust to match the real `package.json` scripts.

```bash
npm run dev          # start dev server
npm run build        # production build
npm run start        # run production build
npm run lint         # ESLint
npm run typecheck    # tsc --noEmit
npm run test         # unit tests (domain logic, date layer)
npm run fsd:lint     # FSD boundary check (e.g. steiger ./src)  ← run before done
npm run lint:styles  # style gate: no static style props / hex / unknown vars (§10.1)
npm run lint:styles:accept   # re-baseline after you REDUCE violations
npm run lint:dark    # dark-mode gate: token parity + contrast in both themes (§10.3)
npm run autopilot:catalog   # regenerate docs/autopilot-catalog.md from app/autopilot/catalog.ts (§11.6)
```

**Definition of done for any change:** `typecheck`, `lint`, `fsd:lint`,
`lint:styles` and `lint:dark` all pass, and new domain logic has tests. For any
change that touches UI/colours, additionally run the **`check-colors` skill**
(§10.2). A change that adds a control, a route or a sheet also registers it in the
autopilot catalog and regenerates its doc (`npm run autopilot:catalog`, §11.6).

`lint:styles` is a **ratchet**: `scripts/styles-baseline.json` records the
violations each file still carries, and the gate fails only when a file goes
*above* its baseline. So legacy screens don't block you, but a clean file (like
`screens/home`) can never regress. When you clean a file up, run
`lint:styles:accept` to lock the lower number in. A `PostToolUse` hook
(`.claude/hooks/style-gate.sh`) runs the same check on every `.tsx` write.

---


### Browser smoke (Playwright, T-M12-25)

`npm run e2e` (= `npx playwright test`) drives the main routine in a real browser, light and dark (fa): login by OTP →
onboarding → log a period through the log sheet → home ring «روز ۱» → calendar marks today. Selectors are the autopilot
`pilot()` names; every page is checked for raw `namespace.key` text (`e2e/support/raw-keys.ts`). Specs are
`e2e/*.e2e.ts` (vitest never runs them). It needs a LOCAL stack, never `ritme_test`:

1. A private DB: `CREATE DATABASE ritme_e2e` on the docker test stack, goose up, plus the personal-access
   `oauth_clients` row (local-dev skill §2a).
2. backend-go on it: `SMS_PROVIDER=log CYCLE_ENGINE=v6 DB_DATABASE=ritme_e2e HTTP_ADDR=:8020
   CORS_ALLOWED_ORIGINS=http://localhost:3100 make run` (another port + `E2E_API_URL` if 8020 is busy).
3. `cd frontend && npm run e2e` — starts `next dev -p 3100` itself (or reuses one); the OTP is read from
   `otp_verifications`.

Env: `E2E_API_URL` (http://127.0.0.1:8020), `E2E_WEB_PORT` (3100), `E2E_BASE_URL`, `E2E_DB_NAME` (ritme_e2e),
`E2E_DB_CLI`, `E2E_BROWSER_CHANNEL` (`chrome` locally; CI installs Playwright's chromium). One worker on purpose
(send-otp 5/min per IP). Keep `outputDir` outside `frontend/` (next dev reloads on writes inside it). CI: job
`e2e-smoke`, nightly / on demand only.

## 10. Code conventions

- **Components:** function components only. Presentational components take props
  and render; data-fetching/logic lives in hooks (in `api/` or `model/`). Keep
  the two roles separate so UI stays reusable and testable.
- **Naming:** components `PascalCase`, hooks `useX`, files for components match
  the component name. Slices and segments are `kebab-case` folders.
- **Styling:** classes in `src/app/globals.css` — **never a `style` prop.** See
  §10.1. Use **logical properties** everywhere (`margin-inline-start`,
  `inset-inline-start`, `text-align: start`); hardcoded `left`/`right` breaks
  RTL and is a bug here (§12).
- **Colours:** always a CSS variable from the `:root` block in `globals.css`
  (`var(--brand)`, `var(--text-2)`, `var(--brand-soft)`). **Never a hex literal.**
  A hex bypasses the `[data-theme="dark"]` overrides, so it silently breaks dark
  mode — the single most common visual bug this codebase has had. If no token
  fits, add one to *both* `:root` and `[data-theme="dark"]` first.
- **Types:** prefer explicit return types on exported functions; model domain
  shapes as `type`/`interface` in the entity's `model/`. Validate external data
  (API responses, form input) with `zod` at the boundary.
- **No barrel imports across layers** except a slice's own `index.ts` (§3.3).
- **Comments:** explain *why*, not *what*. The architecture explains the *what*.

### 10.1 No inline styles

`style={{ … }}` is **not** how this app is styled. Every static rule belongs in
a class — in `src/app/globals.css` (tokens, base, shell, shared primitives) **or** in the CSS file of the slice
that renders it (`<slice>/ui/<prefix>.css`, or `shared/ui/styles/<prefix>.css` for a family several slices
use), imported by every file that renders the family (T-M12-27). Keep screen CSS out of `globals.css`:
everything the root layout reaches statically is downloaded by **every** route, and layout-reachable CSS loads
*before* `globals.css`, so it must win by specificity, not by source order (cascade ties go to `globals.css`).
Do not import a sheet/screen/feature barrel from the layout: use `next/dynamic` (`app/sheets/registry.tsx`,
`app/LazyGlobals.tsx`). `npm run check:bundle` (after `npm run build`) enforces the budgets.

Why this is a hard rule and not a preference:

- **`:hover`, `:focus-visible`, `:disabled`, `:not(:last-child)` and media
  queries cannot be expressed inline at all.** The app shipped with almost no
  keyboard focus affordance purely because of this — an accessibility defect,
  not a style opinion.
- A `style` object is a **new object identity on every render**, so it defeats
  `React.memo` on any child that receives it.
- Inline styles **re-ship in every HTML response** instead of being cached once
  as CSS, and they force `style-src 'unsafe-inline'` in the CSP.
- The same card/pill/row gets **re-typed in each screen** and then drifts.

**The one allowed exception: a value that comes from data.** A marker colour, a
percentage offset, a gradient angle — anything the component cannot know until
it has the data. Keep the geometry in the class and pass only the datum:

```tsx
// ✅ correct — class holds the shape, inline holds the datum
<span className="home-bar-fill" style={{ width: `${todayPos}%` }} />

// ❌ wrong — static geometry inlined
<span style={{ position: 'absolute', top: 0, bottom: 0, left: 0,
               borderRadius: 99, width: `${todayPos}%` }} />
```

Prefer a **modifier class** over a conditional inline value when the states are
known up front (`is-open`, `is-loading`, `has-action`), and prefer driving
visuals from an ARIA attribute you already set — e.g.
`.toggle[aria-expanded="true"] .chev { transform: rotate(180deg); }` — so the
state is declared once.

`src/screens/home/ui/HomePage.tsx` is the reference implementation: 110 inline
style objects reduced to 13, all of them data-driven.

**Enforcement:** `npm run lint:styles` fails on a static `style` prop or a hex
literal. A `PostToolUse` hook runs it automatically on every `.tsx` write, and
it is part of the definition of done (§9).

### 10.2 Colour palette — Night & Bloom (non-negotiable)

Since B-N1-02 the app wears the **Night & Bloom** palette in both themes. The
source of truth is `docs/night-bloom/tokens.md` (light = dialect A of the
`nbl_` artboards, dark = `nbd_`); `globals.css` implements it. Every colour on
screen traces back to a token below — no new hues.

| Role | Light → Dark | Tokens | Use for | Never for |
| ---- | ------------ | ------ | ------- | --------- |
| **Canvas / surfaces** | `#F7F3FF` / `#FFFFFF` → `#17112B` / `#221A3D` | `--page`, `--surface`, `--surface-2/3` (tints, translucent at night), `--surface-4` (opaque inset), `--surface-glass` (nav, sticky footers + blur) | page background; cards, sheets | text |
| **Text** | `#231B3B` `#5E5873` `#6A6480` `#8C82AD` → `#F6F1FF` `#B8AED6` `#8C82AD` `#6E6392` | `--text-1` … `--text-4` | titles/body, secondary, captions; `--text-4` placeholder/disabled only (below AA) | — |
| **Lines** | `#E7E1F4` / `#DDD6FA` → `#34295A` / `#3B2F63` | `--line`, `--line-strong`, `--track` | 1px card border (cards are flat, **no shadow**), selected outline, rails | — |
| **Primary** | `#6E54F0` → `#B9A6FF` | `--brand` (text/icon), `--brand-fill` (fill), `--on-brand` (label on the fill: white → `#17112B`), `--brand-strong`, `--brand-ink` (text on `--brand-soft`), `--brand-soft`, `--brand-2` | CTAs, FAB, switch on, selected tab/chip, active nav, links | large surfaces |
| **Data (turquoise)** | `#0FA3C9` → `#4CE0C3` | `--data`, `--data-deep` (text), `--data-soft` | measured/predicted markers, ovulation, chart lines, «طبیعی» verdicts | decoration, buttons |
| **Warm (amber)** | `#F5A623` → `#FFB86B` | `--warm`, `--warm-deep` (text), `--warm-soft`, `--caution-soft` | fertile window, streaks, Plus, warnings | text (use `-deep`) |
| **Period (red)** | `#E8436F` → `#FF6B8B` | `--period`, `--period-deep`, `--period-soft`, `--period-line` | menstruation — see below | general UI |
| **Bloom** | `#FF6FAE` → `#FFD5B8` (peach at night) | `--bloom`, `--bloom-deep`, `--avatar-grad` | illustration, avatars, companion accent | — |
| **Status** | success `#0F7B6C` → `#7FE0A8`; danger `#C42D57` → `#FF8FA3` | `--success*`, `--danger`, `--danger-deep`, `--danger-soft` | done/taken; errors | — |

**Gradients are soft tints only.** The saturated purple→pink brand gradient is
retired: CTAs and the FAB are **solid `--brand-fill`** with `--on-brand` text
and `--shadow-cta` / `--shadow-fab`. Allowed gradients: `--hero-tint` (hero
cards), the tint pattern `linear-gradient(135deg, <accent> 15–20%,
var(--surface) 70%)`, `--avatar-grad`, `--bg-glow`. The old names
(`--gradient-brand`, `--grad-start`, `--grad-end`) survive as aliases that paint
the solid fill — don't use them in new code.

**The dark sky.** `.nb-sky` is the reusable background layer: the violet
`--bg-glow` circle (both themes) plus nine static stars painted in `--star`
(transparent by day, white at night). Static by design, so nothing to disable
under reduced motion.

**Legacy aliases.** Yesterday's names (`--ink*`, `--muted*`, `--violet`,
`--pink-bg`, `--teal*`, `--amber*`, `--green*`, `--care-rose*`, `--fert-*`, …)
are re-pointed at the semantic tokens (tokens.md §3) so unported screens restyle
for free. New code uses the semantic names.

**Shape, type and spacing tokens** (non-colour, one `:root` block): radii
`--r-xs … --r-nav` (cards `--r-card` 24px), type `--fs-num-xl … --fs-2xs` +
`--font-display` (Lalezar: numerals *and* display titles; Vazirmatn 600/700/800
for UI), spacing `--gutter` 16 / `--gutter-form` 20, touch sizes `--h-btn` 54,
`--size-icon-btn` 44, `--size-fab` 56, `--h-nav` 70.

#### Menstruation (period) — the one sanctioned red

**Bleeding days must read as red on every calendar surface** — a purple or pink
period day is a comprehension bug. Tokens: `--period` (marker/dot/fill),
`--period-deep` (text), `--period-soft` (day cell) + `--period-soft-faint` /
`--period-soft-strong`. Defined once in `entities/cycle/model/markers.ts`; never
re-define a period colour in a component.

- Turquoise owns ovulation, amber the fertile window, violet PMS — red is *only*
  menstruation.
- **Danger shares the rose ramp in the design** (`--danger` `#C42D57`). Tell an
  error apart by icon + copy + the `--danger-soft` surface, never by a period
  marker shape (docs/night-bloom/gaps.md #11).

Hard rules:

- **Solid primary, one label token.** Anything on `--brand-fill` uses
  `--on-brand` (it flips to dark text at night). `--on-accent` (theme-stable
  white) is only for the fills that stay saturated in both themes:
  `--success-fill`, the pregnancy hero (`--preg-hero-*`), the splash
  (`--splash-from/to`).
- **Text uses the `-deep` token.** `--data` (2.96:1), `--warm` (2.03:1),
  `--period` (3.84:1) and `--text-4` fail AA as text on white.
- **Never re-introduce a retired hex** (`#7B61FF`, `#3DD6F3`, `#F2ECFF`,
  `#2F2F35`, `#131022`, …) — `lint:dark` fails on them in `src/` and
  `public/offline.html`. Retheming happens by changing token *values*.
- All the §10 rules still apply: tokens only, no hex literals in `src/`, every
  token defined in both `:root` and `[data-theme="dark"]` (or derived, or
  listed as theme-stable).

**Enforcement:** `npm run lint:dark` (§10.3) is the mechanical half; the
**`check-colors` skill** audits the rest for UI work (§9).

### 10.3 Dark mode

Every screen ships **light and dark**, and the default is **follow the system**.
The preference lives in `shared/theme` (`localStorage['ritme_theme']` =
`light | dark | system`, a Zustand store). Fresh installs get `system`; a stored
`light`/`dark` is kept (bloom/QUESTIONS.md #2).

- **Preference vs resolved.** The store holds `preference` (what the user
  chose) and `theme` (what is painted, `light | dark`). `setPreference()` sets
  either of the three; `setTheme('light' | 'dark')` is the one-tap toggle.
  Components read `theme`, never `matchMedia`.
- **One attribute.** `applyTheme` writes the *resolved* theme onto
  `<html data-theme>`; every dark value is a token override under
  `[data-theme="dark"]` in `globals.css`. No dark-specific JSX.
- **`system` follows the OS live.** `ThemeApplier` listens to
  `(prefers-color-scheme: dark)` changes and the store re-resolves (ignored for
  an explicit choice). It also syncs a change made in another tab.
- **`themeInitScript` runs before first paint** (inline, top of `<body>`) with
  the same rule — stored light/dark wins, anything else follows the OS — so no
  one sees a flash. It duplicates `resolveTheme` in plain JS; a unit test runs
  it against a fake DOM, and `lint:dark` checks key, default and media query.
- **`color-scheme` is declared in both blocks** — it darkens scrollbars, native
  controls, the caret and autofill.
- **`<meta name="theme-color">` is created at runtime, never by Next.**
  `chromeInitScript` creates one meta from the live `--page` (`#F7F3FF` /
  `#17112B`) and the store rewrites it on every switch. Don't set
  `viewport.themeColor` (React 19 would add a stale twin, audit I-6). Dark also
  gets `apple-mobile-web-app-status-bar-style: black-translucent`.
  `manifest.ts` (`#F7F3FF`) is the only sanctioned colour literal in `src/`.
  iOS startup images (`public/splash/{light,dark}-*.png`) are the two canvases
  with the logo; regenerate them if `--page` changes.
- **Every token needs both values** — or is derived (at any depth) from tokens
  that flip, or is listed in `THEME_STABLE` in `scripts/check-dark-mode.mjs`
  *with the reason*.
- **A translucent white overlay is only allowed on a saturated fill** (the
  `ON_GRADIENT` selectors). Over a surface, use `color-mix(in srgb,
  var(--surface) N%, transparent)` so it flips.
- **Contrast is checked in both themes.** `lint:dark` scores ~60 token pairs
  against WCAG; dark may not fall materially below light unless it still clears
  AA or the pair is marked `design` (values taken from the artboards). The `⚠`
  block lists the design's sub-AA marker colours — use the `-deep` token for text.
- **The offline page (`public/offline.html`) carries its own copy** of the
  bootstrap and the palette: it is served from the service-worker cache with no
  bundle.

---

## 11. Sensitive-domain rules (non-negotiable)

Ritme handles menstrual, fertility, and pregnancy data — some of the most
sensitive personal data there is, and especially sensitive for this user base.
Treat every line of code with that in mind.

- **Privacy by default.** Collect the minimum data needed for a feature to work.
  Don't add tracking, fields, or third-party calls "just in case."
- **Never log health data.** Cycle dates, symptoms, pregnancy status, and
  predictions must never appear in logs, analytics events, error reports, URLs,
  or crash payloads. Scrub them from anything that leaves the device/server.
- **Not medical advice.** Predictions and educational content are informational.
  Copy must avoid diagnostic or prescriptive phrasing, and health content should
  be reviewed and, where appropriate, attributed to reputable sources. When
  generating or editing health copy, flag anything that reads as medical advice.
- **Respectful, inclusive, accurate language** in both locales. Run domain
  terminology past the product/content owner rather than inventing it.
- **Accessibility & safety:** content should be calm and non-alarming; support
  data export/delete; default to the most private sharing setting.

If a request would weaken any of the above, do it the safe way and tell the user
why.

### 11.1 Session rules (auth)

Users must stay signed in for a year. The background is in
`docs/investigations/session-logout.md`.

- **Lifetime:** the API issues 365-day JWTs. `localStorage['ritme_token']` is the
  client's source of truth; never drop it because the flag cookie is missing.
- **Flag cookie:** `ritme_auth=1` (never the token) is **server-set** by the
  same-origin route `POST /api/session/flag`. That avoids WebKit's 7-day cap on
  cookies written from script. `SessionGuard` re-asserts it on start, route change
  and resume. Keep the `document.cookie` write only as a fallback.
- **Refresh window:** `SessionRefresher` calls `POST /auth/refresh-session` when the
  JWT `exp` is less than 30 days away, and stores the new token. The call is
  single-flight, backs off 10 minutes after a failure and stops on 404/405. Don't
  add other refresh paths.
- **What may clear a session:** only `endsSession()` (`shared/session/unauthorized.ts`).
  That means a **JSON** 401 for a request that carried the **current** token,
  with `error_code` ∈ `token_expired` / `token_revoked` / `unauthenticated` (or
  Laravel's legacy `"Unauthenticated."` body). Also explicit logout and account
  deletion. A non-JSON (proxy or Basic-auth) 401, a 5xx, a network error, or a 401
  for a token the app no longer holds must **never** clear it. Never call
  `localStorage.clear()`.
- **Keys:** Passport `backend/storage/*.key` never ship (excluded from rsync and
  Docker). Rotating them signs out every user, so it needs a plan. See `backend/CLAUDE.MD`.

## 11.5 AI gate — every AI surface goes through `features/ai-gate`

AI features cost **credits** («اعتبار», `docs/ai-credits.md`): a running subscription is required, the period's
credits are spent first, then the extra ones. The server is the authority; the frontend only explains refusals and
keeps the meter right. The word is always «اعتبار» / "credit" — **never "token"**, never a token count.

**The rule:** every AI surface — now and in the future — uses `useAiGate(featureKey)` or `<AiGate feature="…">`
from `@/features/ai-gate`. A 402 / 403 of an AI endpoint is **never** handled by hand (no `plusDenialOf`, no custom
paywall card, no free-quota line). `plusDenialOf` / `PlusFeatureGate` stay for the non-AI Plus features (companion,
analysis range, PDF share) only.

- `aiDenialOf(error)` (`@/entities/ai-credit`) is the ONE normaliser: kinds `subscription` · `credits` · `disabled` ·
  `consent` · `busy` · `unavailable`. It takes an `ApiError` or the `error_code` of an SSE `error` event.
- `useAiGate(key)` → `{ status, cost, balance, canUse, guard, handleError, denial, refresh }`. It **fails open**
  while the balance is unknown — never block on a failed read.
- `guard()` **before** starting the action (false = the gate sheet opened, don't start).
- `handleError(error)` in **every** catch / `onError`: true = the gate sheet took it (show nothing else); false =
  show your own consent / busy / unavailable state from `gate.denial` / `aiDenialOf`.
- `refresh()` after **every success**, so the credit meter stays right.
- `<AiGateSheet />` is mounted once in the shell (`app/[locale]/layout.tsx`). A safety net (QueryClient
  `MutationCache` / `QueryCache`, and the SSE stream that bypasses `apiClient`) opens it for a 402 a surface forgot.
- A *feature* slice cannot import `features/ai-gate`: the composing screen / widget calls `useAiGate` and passes it
  down as the `AiGateHooks` prop (`entities/ai-credit`) — see `widgets/voice-log-slot`, `features/upload-lab`.
- The balance UI is shared: `CreditMeter` (entity), `/profile/credits`, `/plus/credits` (purchase = a **route**, the
  gate sheet is only the short decision). Amber `--warm` is the credits accent.

**Add a new AI feature in 5 steps**

1. Backend: `ai.RegisterFeature("foo", ai.CapChat)` + a policy, and `platform.guard.Chain(FeatureFoo)` on the route
   (`docs/ai-credits.md`). The slot then appears in `GET /ai/credits` `features[]` with its `credit_cost`.
2. Frontend: the surface's request lives in its slice's `api/` (a `useMutation`); no gate code in there.
3. The screen calls `const gate = useAiGate('foo')` and wraps the entry point in `<AiGate feature="foo">` (or calls
   `gate.guard()` in the click handler).
4. Every failure path: `onError: (e) => { if (gate.handleError(e)) return; /* consent / busy / unavailable */ }`.
5. Every success path: `onSuccess: gate.refresh`. Show a cost only from `gate.cost` / `features[].credit_cost` — never hard-code one.

```tsx
const gate = useAiGate('foo');
const run = useFoo();
const start = () => {
  if (!gate.guard()) return;                       // opens «اشتراک لازم است» / «اعتبار تمام شد»
  run.mutate(input, {
    onSuccess: gate.refresh,                       // keep the meter right
    onError: (e) => {
      if (gate.handleError(e)) return;             // gate sheet took it
      const d = aiDenialOf(e);                     // consent / busy / unavailable: your own state
    },
  });
};
return <AiGate feature="foo"><button onClick={start}>…</button></AiGate>;
```

## 11.6 UI autopilot — every control is named and catalogued

The autopilot (`docs/autopilot.md`) runs a JSON script against the live UI like a person: opens the page,
finds the field, types, presses, saves. A model will write those scripts from what the user says, so the
app must be able to tell it **what exists**. That only works if every control has a stable name.

- `@/shared/pilot` — the vocabulary: `pilot(id, kind?)`, `pilotOption(code)`, the script schema, the catalog
  shape, `buildAutopilotManifest`. Pure; any layer may import it.
- `@/shared/autopilot` — the engine: `runAutopilot(script, options?)`, `AUTOPILOT_CONFIG`
  (`shared/autopilot/config.ts` is the ONLY place speeds, visuals, interruption and safety are tuned),
  `<AutopilotHost>` (mounted once in the layout).
- `src/app/autopilot/catalog.ts` + `groups/*.ts` — **the catalog**: every route, sheet and target, with how to reach it
  (titles: `messages/*/autopilot-catalog.json`, namespace `autopilotCatalog`).
  `docs/autopilot-catalog.md` is generated from it (`npm run autopilot:catalog`).
- `useAutopilotManifest()` (`widgets/autopilot-manifest`) — the manifest sent to the model: the catalog
  plus this user's log taxonomy.

**The rule: a new interactive control is named and registered in the same change.**

1. Put a target id on it — `<area>.<type>` or `<area>.<type>:<key>`, lower-case, built from stable codes
   (never a label, a class or a `useId()`): `{...pilot('pay.submit')}`; a chip row is
   `<ChipGroup pilot={pilot(id, 'choice')}>` with `{...pilotOption(code)}` per chip; `Switch`, `Checkbox`,
   `Accordion`, `NumberStepper`, `SegmentedTabs` take a `pilot` prop. A new `shared/ui` control accepts
   `pilot?: PilotAttrs` (or spreads `...rest`).
2. Register the id (or its `family:*`) in the matching `app/autopilot/groups/*.ts` with its `surface`, a `reveal` if
   something can hide it, a `titleKey` in `messages/*/autopilot-catalog.json`, and `options` / `min` / `max` when fixed.
   Destructive, payment and auth controls are NOT named (docs/autopilot.md §7); a button that commits the user to
   something goes in `safety.confirmTargets`.
3. A new route or sheet users could ask for goes in the catalog's `routes` / `sheets`.
4. Fields rendered from data get a manifest builder (see `features/log-day/model/pilot-targets.ts`).
5. `npm run autopilot:catalog`, then `npm run test` — `app/autopilot/catalog.test.ts` fails on a marked id
   that isn't catalogued, a catalogued id nobody marks, a missing title, a route/sheet that doesn't exist,
   or a stale doc.

**Writing it right — check each point before you call a screen done**

| Point | Right | Wrong |
| ----- | ----- | ----- |
| Id | `med.dose-unit`, `care.tab:${tab}` — area unique to the screen family, key from an enum / API code / numeric id | a label, a class, an index into a translated list, `useId()`, the areas `nav.` / `sheet.` / `log.` outside their owners |
| One control, one id | a component reused on several screens keeps one id scheme; if it appears twice on a screen it takes an id-prefix prop (`pilotId`) | two elements on screen with the same `data-pilot` |
| Where the attribute goes | on the element that receives the click or the value (the `<button>`, the `<input>`, the group for a choice) | on a wrapper `<div>` around a button; on an icon |
| A choice | group: `pilot(id, 'choice')`; every option: `pilotOption(code)` **and** a real state (`aria-pressed` / `aria-checked` / `aria-selected`) | options without `pilotOption`; a hand-rolled chip with no ARIA state (the engine can't tell on from off) |
| Something that opens | the toggle carries `aria-expanded`; whatever it hides lists it in `reveal` | a target hidden behind a section, a panel or an inline sheet with no `reveal` |
| Kind | let the DOM say it; pass a kind only where it can't: `'choice'`, `'number'` (commits on blur), `'stepper'`, `'wheel'` (+ `pilotValues`), `'ruler'`, `'calendar'`, `'select'` | `'text'` on an input that only commits on blur; a wheel without `pilotValues` when the rows mean years or minutes |
| Catalog entry | exact id: `titleKey` + `surface`; fixed choice: `options` (code + title key); number: `min` / `max` / `unitKey`; data-keyed ids: one `family:*` entry | an exact entry per member of a data-driven list; a `surface` on a route with `[id]` (leave it out) |
| Options from data | a manifest builder next to the slice (`logTargetsOf`) wired into `useAutopilotManifest` | hard-coding server options in the catalog |
| Titles | `messages/{fa,en}/autopilot-catalog.json` (namespace `autopilotCatalog`), copied to `backend-go/resources/translations/` | catalog titles in `autopilot.json` — that namespace ships with every page |
| Risky controls | pay, purchase, delete, revoke, sign-out, app lock, auth, AI submits, chat send, uploads: **no id**. A button that commits the user outside the app: id + `safety.confirmTargets` | naming a destructive control "because the user might ask" |

Then: `npm run autopilot:catalog` and `npm run test`. Drive the new form once from `/dev/autopilot` (the
"Agent manifest" tab shows what the model will be told) — the catalog test proves the id is registered, not
that the control can actually be driven.

Hard rules:

- **No values in reports, logs or the manifest** — ids and codes only (§11).
- **A script is untrusted input**: never widen `safety` (raw CSS selectors, blocked routes) to make one pass.
- **Drivers use real DOM events**, never React state. If a control can't be driven, add a driver in
  `shared/autopilot/dom/drivers.ts`; don't special-case a screen.
- **The engine calls no model.** The request that *produces* a script is an AI call and goes through
  `useAiGate` (§11.5); `runAutopilot` itself is never gated.
- Timing, visuals and interruption come from `AUTOPILOT_CONFIG` — no hard-coded delay in a caller.

---

## 12. Anti-patterns — do NOT do these

- ❌ Importing across sibling slices (`entities/cycle` → `entities/symptom`).
- ❌ Importing a slice's internals instead of its `index.ts`.
- ❌ Importing a lower layer from a higher one's perspective, i.e. any upward
  import (`shared` importing from `features`, etc.).
- ❌ Hardcoded user-facing strings (bypassing `next-intl`).
- ❌ Enumerating locales in code — `['fa', 'en']`, `locale === 'fa' ? … : …`
  for direction or labels, a `Record<Locale, …>` name table. The language list
  is data (§6). The only legitimate `locale === 'fa'` checks are Persian digits
  and the Jalali calendar.
- ❌ Hardcoded `margin-left` / `right: 0` / `text-align: left` — use logical
  properties (`margin-inline-start`, `inset-inline-end`, `text-align: start`).
- ❌ `style={{ … }}` for anything static — put it in a class (§10.1). Only a
  value that comes from data may be inline.
- ❌ Hex colour literals (`'#E91E63'`, `#fff`) anywhere in `src/` — use a token
  from `globals.css`, or add one. A hex does not flip in dark mode.
- ❌ `var(--some-name)` for a variable that isn't declared in `globals.css`.
  It silently resolves to nothing: `color` inherits, `background` goes
  transparent. This shipped to production once already.
- ❌ Calling the date library or `Date` formatting directly outside
  `shared/lib/date`; showing Gregorian dates to users.
- ❌ `router.push('/parent')` for a back arrow, a bare `history.back()`, or a
  `popstate` / hardware-back listener of your own — the arrow is
  `router.back('/parent')`, overlays use `useBackLayer` / `AppSheet` (§4.2).
- ❌ Putting server data into Zustand/`useState` instead of TanStack Query.
- ❌ Logging or transmitting health data anywhere it isn't strictly required.
- ❌ Handling a 402 / 403 of an AI endpoint by hand (`plusDenialOf`, a bespoke paywall / quota card), or showing token counts — every AI surface uses `useAiGate` / `<AiGate>` (§11.5); call `handleError` in every catch and `refresh` after success.
- ❌ An interactive control without a `pilot()` name, or a name that isn't in `app/autopilot/catalog.ts` (§11.6); a label, class or `useId()` used as a target; a script value in a log or report.
- ❌ Adding abstraction layers preemptively. Introduce structure when a pattern
  repeats (rule of three), not before.

---

## 13. How to work in this repo (summary for Claude)

1. Read the request and decide the **layer** (§4).
2. Reuse existing slices (§5); don't fork parallel versions.
3. If creating a new slice, follow the **`fsd-slice` skill** for the exact
   scaffold, public API, i18n namespace, and RTL checklist.
4. Route every string through i18n (§6) and every date through `shared/lib/date`
   (§7).
5. Use logical styling for RTL (§12).
6. Respect §11 for anything touching health data.
6b. Name every new interactive control with `pilot()` and register it in the autopilot catalog, following the
   "Writing it right" table in §11.6 — a screen without it is not done.
7. Before finishing: `typecheck`, `lint`, `fsd:lint`, and tests for new logic.
