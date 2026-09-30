# Night & Bloom — progress notes

One `## B-Nx-NN` section per finished task: what shipped, commands/env vars, migrations, open items.

## B-N1-01 — Design import audit

- Shipped `docs/night-bloom/` — `README.md` (summary + per-task reading list) linking `tokens.md` (light/dark colour
  table with proposed CSS variables + legacy aliases, gradients, starfield/glow, type, radii, shadows, spacing,
  contrast), `components.md` (inventory → artboards), `routes.md` (183 artboards → route/sheet, restyle vs NEW,
  owner task), `nav.md` (per-mode tabs, visibility, deviating boards), `gaps.md` (19 gaps with defaults). No code changed.
- Key findings: two light dialects (canonical `#6E54F0`); brand gradient gone (solid CTAs/FAB); dark primary fill is
  lavender with dark text (`--on-brand` flips); Lalezar used for display titles too (current subset font too small);
  no analysis entry point in the nav; 55 restyle / 119 NEW / 9 reference-only screens.
- Task scopes updated: B-N1-02, B-N1-03, B-N1-04, B-N1-14 (Design), B-N2-03, B-N3-08, B-N4-05, B-N8-03.
- Open: `bloom/QUESTIONS.md` #1–#5; B-N1-02 `touches` lacks `frontend/src/app/fonts` + `frontend/src/app/manifest.ts`
  and B-N1-04 `touches` names a non-existent `(app)` route group — adjust frontmatter when those tasks start.

## B-N1-02 — Night & Bloom tokens, light/dark/system theme and gates

- **Tokens** (`frontend/src/app/globals.css`): tokens.md §2 table implemented in `:root` (dialect A) and
  `[data-theme="dark"]`. New semantic names: `--text-1..4`, `--surface-4`, `--surface-glass`, `--line-strong`,
  `--on-brand` (white → `#17112B`), `--brand-ink`, `--brand-soft`, `--brand-2`, `--warm*`, `--period-line`, `--bloom*`,
  `--caution-soft`, `--danger-deep`, `--splash-from/to`, `--star`; tints `--hero-tint`, `--avatar-grad`, `--bg-glow`;
  shadows `--shadow-cta/fab/float/hero/modal/knob`, `--glow-data`; non-colour `--r-*`, `--fs-*`, `--font-display`,
  `--lh-body`, `--gutter*`, `--h-btn*`, `--size-*`, `--h-nav`, `--pad-nav-clear`.
- **Legacy aliases** (tokens.md §3): `--ink*`, `--muted*`, `--violet`, `--pink-bg`, `--indigo*`, `--teal*`, `--blue*`,
  `--amber*`, `--green*`, `--care-rose*`, most `--fert-*`, `--checkup-*` re-pointed via `var()`. `--gradient-brand`,
  `--grad-start/end` now paint solid `--brand-fill` (CTAs/FAB/active tab/progress go solid for free). Rule bodies on
  primary fills switched `--on-accent` → `--on-brand`; old brand-rgba shadows → `--shadow-cta/fab` / color-mix.
  Period editor sheet re-scoped onto `--period*`; phase-details hero → brand-fill→bloom tint with `--on-brand`;
  splash on theme-stable `--splash-from/to`. Shell/stage background now `--page`.
- **Sky layer:** `.nb-sky` (glow + 9 static stars in `--star`, invisible by day) and `.nb-card` utility — defined,
  not yet mounted (B-N1-04 shell / screen tasks).
- **Theme** (`shared/theme`): `light | dark | system`, default `system`; store exposes `preference` + resolved `theme`,
  `setPreference`, `setTheme` (explicit toggle, existing callers untouched), `syncSystem`; `ThemeApplier` follows
  `prefers-color-scheme` live + cross-tab; `themeInitScript` resolves the same way before paint (unit-tested against
  a fake DOM). Storage key unchanged (`ritme_theme`), so `bloom/bin/shot.mjs` needed no change.
- **Gates:** `check-dark-mode.mjs` — plumbing now requires default `system` + media query in the init script + a live
  listener; parity derivation is transitive; new PAIRS for semantic tokens (60); ratchet waived when dark still clears
  AA or the pair is `design`; new check 6 bans the retired palette hexes in `src/` + `public/offline.html`.
  `lint:styles` baseline unchanged.
- **Font:** `Lalezar-Subset.woff2` rebuilt from Lalezar-Regular 1.0 — Basic Latin + whole Arabic block + ZWNJ/punct,
  all OT features (45 KB, not preloaded, `swap`). Rebuild command in the `@font-face` comment.
- **Chrome:** `manifest.ts` → `#F7F3FF`; iOS startup images recoloured (`#F7F3FF` / `#17112B`); `public/offline.html`
  on the new palette + system resolution. `frontend/CLAUDE.md` §10.2/§10.3 rewritten.
- **Screenshots:** `docs/qa/bloom/B-N1-02/` (home, calendar, profile × light/dark, persona 04) + `artboards/`
  (`nbl_`/`nbd_Cycle_Home`).
- **Open:** TSX still using `--on-accent` on brand fills (onboarding checks, DayTasks, TodayChallenge, PeriodButton,
  DailyStatusCard, IntroIllustration, DayLogPage) read white-on-lavender at night → owning screen tasks switch to
  `--on-brand`. Theme picker UI (light/dark/system) for Me belongs to the Me-screen task (store API ready).
  `frontend/.claude/skills/check-colors` still describes the old palette. `admin-web/` still on the old palette (N9).
  QUESTIONS #6–#7.

## B-N1-03 — Shared UI primitives in Night & Bloom

- **Primitives** (`frontend/src/shared/ui/nb/`, all exported from `@/shared/ui`; CSS = `nb-*` block at the end of
  `globals.css`; accent tones via `Tone` = `brand | data | warm | period | bloom | success | danger | neutral`):
  `ScreenHeader` (44px back/close via `onBack`+`backLabel`, title/subtitle, `action`, `center` slot for steps),
  `HeaderButton` (44px round, `variant="soft"`, `badge`), `HubHeader` (date over greeting + actions),
  `Card` (`variant` default/secondary/hero/inset, `padding`, `as`), `HeroCard`, `SectionTitle` (end link),
  `PrimaryButton` / `SecondaryButton` (54px pill; `variant="outline"|"text"`, `loading`, `block`),
  `TileButton` (`layout="compact"` 58px | `"card"` quick-log card with `sub`; `pressed`),
  `PillChip` (`aria-pressed`, `mode="single"|"multi"`, `tone`, `suggested` dashed, `shape="square"`) + `ChipGroup`
  (`layout="fill"`), `SegmentedTabs` (tablist, roving tabindex, RTL-aware arrows, `track="surface"`),
  `NumberStepper` (Lalezar value in locale digits, −/+ clamped, `boxed`), `Switch` (role=switch 48×30, `compact`,
  `labelledBy`), `StatusPill` (`tone`, `solid`), `PlusLock` (blurred inert teaser + «پلاس» pill, `onUnlock`),
  `ListRow` (icon disc, value/`trailing`, `onClick` → button + chevron, `id` → `${id}-title` for Switch labelling) +
  `ListGroup`, `Accordion` (aria-expanded/controls region, `active` tone), `IconCircle` (34/40/44, `outlined`),
  `Avatar`, `InfoNote` (role=note, `source`), `UrgentCard` (role=alert), `EmptyState`, `Skeleton` +
  `SkeletonGroup` (role=status, static under reduced motion), `ProgressRing`, `ProgressSteps`, `DateStrip`
  (Jalali week via `shared/lib/date`, `marker` tone dots, `maxDate`), `LineChart` (series/band/today glow, null =
  gap) / `BarChart` — SVG, `direction:ltr` plot, geometry in `nb/chart-geometry.ts`; `SkyLayer` (`.nb-sky`).
- **BottomSheet** = `AppSheet` (`@/shared/sheet`) restyled in place: `--page` panel, radius `--r-sheet`, 40×5
  `--line` grip, 44px round close, 17/800 title, 16px gutters. No second implementation.
- **Old primitives:** `Button`, `NavBack` marked `@deprecated` (API differs; not replaced in callers — screen
  tasks switch). `Icon` now renders `aria-hidden`/`focusable=false` and gained `lock`, `minus`, `arrowR`.
- **Tests:** `nb/primitives.test.ts` (a11y contract of every primitive on SSR markup) + `nb/chart-geometry.test.ts`;
  `vitest.config.ts` got `esbuild.jsx = 'automatic'` so `.test.ts` files can render components.
- **Showcase:** dev-only route `/fa/dev/ui-kit` (`screens/ui-kit`, 404 in production), theme toggle in the header.
  Screenshots `docs/qa/bloom/B-N1-03/` (light+dark) + `artboards/` (Log_Sheet_Cycle, Cycle_Settings, Onb_Cycle).
- **Outside `touches`** (minimal, flagged): `app/globals.css` (nb block + `.osheet` restyle + `.uikit-*`),
  `app/message-scopes.ts` (`uiKit`), `app/[locale]/dev/ui-kit/page.tsx`, `screens/ui-kit/`, `vitest.config.ts`.
- **Open:** FAB/BottomNav (B-N1-04), home CycleRing with phase arcs + home date strip (B-N1-06), Checkbox/TaskRow,
  Table, AdSlot, Toast not built (not in scope list). Sheets now sit on `--page`; sheet content that painted its own
  `--surface` block may look boxed until its screen task. QUESTIONS #9–#10.
