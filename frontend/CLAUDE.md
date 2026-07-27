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
default locale and the default text direction is **RTL**. English (`en`) is a
secondary locale.

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
| Dates / calendar   | **Jalali (Shamsi)** via a centralized date layer (§7)         |
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

- **Library:** `next-intl`, integrated with the App Router and RSC so strings
  render on the server and don't bloat the client bundle.
- **Locales:** `fa` (default) and `en`. **Locale lives in the URL path**
  (`/fa/...`, `/en/...`) for SEO and shareable links.
- **Namespaces per slice.** Each feature/widget owns its message namespace and
  loads it lazily. Do NOT dump every string into one giant `common.json` and
  do NOT ship all locales to the client.
- **ICU MessageFormat** for plurals, gender, and number/date formatting. Persian
  has different plural rules than English — write messages with ICU, never
  string-concatenate translated fragments.
- **Type-safe keys.** Message keys are typed; a wrong key is a compile error,
  not a runtime blank. Never hardcode user-facing text in components — every
  visible string goes through the translator.

```tsx
// ✅ correct
const t = useTranslations('cycle.calendar');
return <h2>{t('nextPeriodIn', { days: count })}</h2>;

// ❌ never
return <h2>دوره بعدی تا {count} روز دیگر</h2>; // hardcoded, untranslatable
```

---

## 7. Dates & calendar (Jalali — critical)

Iranian users think in the **Jalali (Shamsi)** calendar. Getting this wrong is a
correctness bug in a product whose entire job is tracking dates.

- **All user-facing dates are Jalali.** Gregorian may exist internally/at the
  API boundary, but never shown raw to the user.
- **One centralized date layer:** `shared/lib/date`. Wrap the date library
  (e.g. `dayjs` + a Jalali plugin) there and expose helpers like
  `formatJalali()`, `toJalali()`, `addDays()`, `diffInDays()`. Components and
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

- **Spec (OpenAPI/Swagger JSON):** `http://ritmeapp.ir/docs/api-docs.json`
  (title: *Ritme Salamat API*, version `1.0.0`). Fetch it over plain `http`
  (the host does not serve `https`).
- **Base URL:** all endpoints are under **`/api/v1/`**.
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
Dates cross this boundary in the API's format; convert to Jalali only in
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
```

**Definition of done for any change:** `typecheck`, `lint`, `fsd:lint` and
`lint:styles` all pass, and new domain logic has tests.

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
