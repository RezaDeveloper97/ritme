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
| Forms + validation | **react-hook-form** + **zod**                                 |
| Dates / calendar   | **Locale-aware** (Jalali for `fa`, Gregorian for `en`) via a centralized date layer (§7) |
| HTTP               | single shared `axios`/`fetch` client in `shared/api`          |

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

The app has **exactly five routed screens**: the bottom-nav tabs (`/home`,
`/calendar`, `/log`, `/cycle`, `/profile`), plus the sign-up + onboarding flow
(`/splash`, `/welcome`, `/signup`, `/otp`, `/onboarding/*`) and the pregnancy
section. **Every other screen is a sheet** — a panel that rises from the bottom
over whatever the user was already looking at. Do not add a route for one.

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

Sheets are addressed by `?sheet=<id>` and stack, so the hardware/browser back
button dismisses the top one and deep links work. Navigating to another route
closes them.

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

### 4.2 Back never navigates

The browser/hardware back button does **not** walk the app's screen stack — in
the browser, the installed PWA and the Android WebView shell alike.
`shared/back-guard` traps it by keeping a duplicate history entry above every
screen, so a back press lands on the same URL and nothing moves. The Android
shell asks the page first (`window.__ritmeBack`) and otherwise treats back as
"leave the app", instead of `WebView.goBack()`.

This exists because Ritme's screens are a tab bar, not a document trail. The
visible bug was signing out: one back press resurrected the signed-in screens
from the bfcache, rendering the previous user's data against a dead session.

Consequences for new code:

- **Never call `router.back()`** — `useRouter()` from `@/shared/i18n` no longer
  exposes `back`/`forward`. An on-screen back arrow navigates *forwards* to an
  explicit route (`previousOnboardingRoute(...)`, `/signup`), so the destination
  is known rather than whatever happens to be in history.
- **Sheets are the one exception**: they live in the query string, the guard
  lets that pop through, and back still dismisses the top one.
- **Sign-out is a full document replace** (`window.location.replace`), not a
  client-side one, so no React tree or query cache survives it.

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
  editing keys there, run `php artisan translations:import` in the backend so
  languages created later inherit the new keys.
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

The backend is a separate service. **The full API is documented as an OpenAPI
3.0 spec** you should treat as the source of truth for endpoints, request/
response shapes, enums, and auth:

- **Spec (OpenAPI/Swagger JSON):** `https://api.ritme.app/docs/api-docs.json`
  (title: *Ritme Salamat API*, version `1.0.0`). The host serves `https` (TLS
  terminates at the nginx proxy; `http` 301-redirects). **Never point the app
  at an `http://` API** — the PWA is served over https, so an http API call is
  mixed content and the browser blocks it.
- **Base URL:** `https://api.ritme.app/api/v1/`. The frontend is served from a
  **different origin** (`web.ritme.app`), so every browser call is cross-origin
  and depends on the API's CORS allow-list (`backend/config/cors.php`). Adding a
  new frontend origin means adding it there too. The admin panel lives on a
  third hostname, `adpanell.ritme.app`.
- **Auth:** JWT **bearer** token in the `Authorization` header
  (`Authorization: Bearer <token>`). Login is **OTP-based**: `POST
  /auth/send-otp` → `POST /auth/verify-otp` returns the access token. The shared
  `axios`/`fetch` client in `shared/api` attaches the token; never scatter auth
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
```

**Definition of done for any change:** `typecheck`, `lint`, `fsd:lint`,
`lint:styles` and `lint:dark` all pass, and new domain logic has tests. For any
change that touches UI/colours, additionally run the **`check-colors` skill**
(§10.2).

`lint:styles` is a **ratchet**: `scripts/styles-baseline.json` records the
violations each file still carries, and the gate fails only when a file goes
*above* its baseline. So legacy screens don't block you, but a clean file (like
`screens/home`) can never regress. When you clean a file up, run
`lint:styles:accept` to lock the lower number in. A `PostToolUse` hook
(`.claude/hooks/style-gate.sh`) runs the same check on every `.tsx` write.

---

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
  (`var(--brand)`, `var(--muted-2)`, `var(--pink-bg)`). **Never a hex literal.**
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
a class in `src/app/globals.css`.

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

### 10.2 Colour palette — the Ritme brand system (non-negotiable)

The app has exactly **three brand hues + three neutrals**. Every colour on
screen must trace back to one of these groups via a token in `globals.css`.
Do not introduce new hues; status colours (danger/success/warning) and the
amber fertile-window marker are the only sanctioned exceptions.

| Group | Value | Tokens | Use for | Never for |
| ----- | ----- | ------ | ------- | --------- |
| **Primary / Brand gradient** | `#7B61FF → #FF6FAE` | `--gradient-brand`, `--grad-start`, `--grad-end` (solid fallback: `--brand`) | Main CTAs ("Log Symptom" etc.), FAB, active progress bars, active bottom-nav tab, hero/branding headers, loading animation | Large surfaces, body text, more than ~2 elements per screen — scarcity is what makes it read as premium |
| **Secondary / Data accent** | turquoise `#3DD6F3` | `--data`, `--data-deep` (text-safe), `--data-soft` (surface) | Data & active/new states only: current cycle status, new-notification dots, ovulation-day marker, insight lines ("Fertile window starts today"), algorithmic prediction curves in charts | Decoration, buttons, backgrounds unrelated to data |
| **Background / Canvas** | lavender `#F2ECFF` | `--page` (dark: deep-lavender) | App/page background, onboarding & long-read screens, calm "breathing" space | Text, borders on white |
| **Neutral: white** | `#FFFFFF` | `--surface`, `--on-accent` | Cards, modals/sheets, text & icons on gradient/saturated fills | — |
| **Neutral: dark gray** | `#2F2F35` | `--ink`, `--ink-2` | Primary text, headings, active icons | — |
| **Neutral: mid/light grays** | ramp | `--muted*`, `--ink-3`, `--line*`, `--track`, `--field-border` | Secondary text, dividers, inactive icons, borders | — |

#### Menstruation (period) — the one sanctioned red

Period days are the single deliberate exception to the palette. **Bleeding days
must read as red on every calendar surface.** Users have decades of convention
attached to that colour; a purple or pink period day is a comprehension bug, not
a style choice. The red is *tuned to the theme* — it sits at the rose end of the
brand gradient rather than being a raw fire-engine red — so it belongs to the
system instead of fighting it.

- Tokens: `--period` (`#E8436F`, marker / dot / accent), `--period-deep`
  (text-safe on light), `--period-soft` (day-cell surface), plus the
  intensity steps `--period-soft-faint` / `--period-soft-strong`. All flip in
  dark mode.
- **Scope:** calendar period markers and period-specific indicators (day cells,
  legend dot, period badges/labels) — defined once in
  `entities/cycle/model/markers.ts`, which both the calendar screen and the home
  mini-calendar read from. Never re-define a period colour in a component.
- **Do not** use `--period*` for general UI (buttons, headers, generic pink
  tints — those stay on the brand gradient / `--pink-bg`), and **do not** use a
  brand-gradient or turquoise colour for a period day.
- Turquoise still owns ovulation, amber still owns the fertile window, violet
  still owns PMS — red is *only* menstruation.

Hard rules:

- **The gradient is scarce by design.** If a screen already shows the gradient
  twice, the next element takes a neutral or a soft tint (`--pink-bg`,
  `--surface-2`), not the gradient again.
- **Turquoise means "this is data"** (measured, detected, or predicted by the
  algorithm). If the element isn't data or an active/new state, turquoise is
  the wrong colour.
- **Red means menstruation** — nothing else may be red except genuine
  error/danger states (`--danger*`).
- **Never re-introduce the legacy pink-brand palette** (`#E91E63` era) or any
  off-palette hue. Retheming happens by changing token *values* in
  `globals.css`, never by adding parallel colour systems.
- All the §10 rules still apply: tokens only, no hex literals in `src/`, every
  token defined in both `:root` and `[data-theme="dark"]`.

**Enforcement:** run the **`check-colors` skill** (`/check-colors`) after any
change that touches colours, styles, or new UI — it audits token conformance,
off-palette hues, and dark-mode coverage. It is part of the definition of done
for UI work (§9). The mechanical half of that audit is automated as
`npm run lint:dark` (§10.3).

### 10.3 Dark mode

The app ships **light, dark and follow-the-system**. The preference lives in
`shared/theme` (`localStorage['ritme_theme']`, a Zustand store) and is exposed
to the user as Profile → «ظاهر و پوسته» (`?sheet=appearance`).

How it works, and the rules that keep it working:

- **One switch, one attribute.** `applyTheme` writes the *resolved* theme onto
  `<html data-theme>`; every dark value in the app is a token override under
  `[data-theme="dark"]` in `globals.css`. Nothing else branches on the theme —
  no `useTheme()` in a component, no dark-specific JSX.
- **`themeInitScript` runs before first paint** (rendered inline at the top of
  `<body>`), so a dark-mode user never sees a white flash. It is a deliberate
  duplicate of `applyTheme` in plain JS; if you change the storage key or the
  resolution rule, change both — `lint:dark` fails when they drift apart.
- **`color-scheme` is declared in both blocks.** It is the only way to darken
  what CSS variables cannot reach: scrollbars, native form controls, the caret,
  the autofill highlight.
- **`<meta name="theme-color">` is rewritten at runtime.** It cannot hold a CSS
  variable, so the layout ships one per `prefers-color-scheme` for the first
  paint and the store overwrites both once a preference disagrees with the OS.
  Those two hex values, plus `manifest.ts`, are the *only* sanctioned colour
  literals in `src/`.
- **Every token needs both values.** A token in `:root` with no
  `[data-theme="dark"]` value must be either derived from tokens that do flip,
  or listed in `THEME_STABLE` in `scripts/check-dark-mode.mjs` *with the reason*
  (e.g. `--on-accent` stays white because the fill under it stays saturated).
- **`--brand` is text; `--brand-fill` is a fill.** Dark mode lifts `--brand` so
  it stays readable as text on a dark card. A saturated fill that carries white
  text must therefore use `--brand-fill`, which does not move. The gradient
  (`--grad-start`/`--grad-end`) is identical in both themes by design.
- **A translucent white overlay is only allowed on a saturated fill** (the
  gradient heroes), because that fill is the same in both themes. Over a
  surface, use `color-mix(in srgb, var(--surface) N%, transparent)` so it flips.
- **Dark may never read worse than light.** `lint:dark` resolves ~57
  foreground/background token pairs in *both* themes, scores them against WCAG,
  and fails when dark drops materially below light. It also reports the pairs
  that are under AA in light mode today — those are pre-existing brand-palette
  decisions, held at their current value rather than silently drifting down.
- **The offline page (`public/offline.html`) carries its own copy** of the
  bootstrap: it is served straight from the service-worker cache with no bundle,
  so it re-reads `ritme_theme` itself.

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
- ❌ `router.back()` / `history.back()` for screen navigation — back is trapped
  app-wide (§4.2). Navigate forwards to an explicit route.
- ❌ Putting server data into Zustand/`useState` instead of TanStack Query.
- ❌ Logging or transmitting health data anywhere it isn't strictly required.
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
7. Before finishing: `typecheck`, `lint`, `fsd:lint`, and tests for new logic.
