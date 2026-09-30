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

## B-N1-13 — Restyle TTC screens to Night & Bloom

- **Screens** `/fertility/log`, `/fertility/bbt`, `/fertility/insights` now use the nb primitives: `ScreenHeader`
  (back → `/home`, BBT info = `HeaderButton`), `SkyLayer` (glow + dark stars), `Card`, `SegmentedTabs` (1/3/6 cycles,
  route-driven), `PillChip` (log chips, `ttc-chip` = soft «on» + 44px), `PrimaryButton`/`SecondaryButton`,
  `EmptyState` (no BBT readings / low data), `SkeletonGroup` loading, a shared error card (danger disc + retry).
  Log save footer = `--surface-glass` + blur; BBT steppers 44px icon buttons.
- **Widgets:** `fertility-tiles` (tiles, chance card donut, LH tip, phase pills) and `bbt-chart` moved from Tailwind
  colour utilities to `ttc-*` classes on semantic tokens; tile skeleton = `Skeleton`. Fertile band token
  `--fert-amber-band` = `--warm` @ 12% in both themes (artboard value; was two hexes). Text on white uses `-deep`
  tokens (QUESTIONS #11).
- **CSS:** one `/* B-N1-13 */` block at the end of `globals.css` (+ the `--fert-amber-band` token line). No new i18n
  keys (fa/en/backend seed untouched).
- **Data:** persona 09900000012 got 18 BBT days + LH/mucus/intercourse rows via `PUT /fertility/days/*` (ritme_dev).
- **Screenshots:** `docs/qa/bloom/B-N1-13/` — log/bbt/insights/home × light+dark, `en_fertility_bbt` (LTR),
  `side-by-side_*.png` (artboard light | app light | artboard dark | app dark), `artboards/`.
- **Open:** verify's `npm run test` fails only in `message-scopes.test.ts` (splash/welcome namespaces — B-N1-05 in
  progress), everything else green. `lint:styles:accept` not run (shared baseline, parallel agents). QUESTIONS #11–#12.

## B-N1-04 — App shell and mode-aware bottom nav

- **Nav** (`widgets/bottom-nav`, rewritten): floating glass pill per nav.md (70px, r35, `--surface-glass` + blur,
  `--shadow-float`), tabs 56px / icon 22 / label 11, active = `--text-1` 800 + 16×3 `--brand` bar + `aria-current`,
  solid 56px FAB (`--brand-fill`, `--shadow-fab`, `--on-brand` plus) — goo/liquid-glass sim + `lib/goo-motion.ts` and
  the old `.tabbar/.tab/.goo-*` CSS deleted (`.fab` kept: calendar DayLogSummary uses it). Glyphs copied from the
  artboards (`ui/NavIcon.tsx`). Badges: `<BottomNav badges={{ services: 3, me: true }} />` (count in locale digits
  + sr-only text, or a dot). Mode tab is a placeholder until hydration / while `/messages/mode` loads (no SSR mismatch,
  no flash of the wrong tab); on error it falls back to the cycle nav.
- **Modes** (`model/nav-items.ts`, unit-tested): `resolveNavMode({mode,isTtc})` → cycle | ttc | pregnancy |
  postpartum | menopause | teen | companion (unknown → cycle); `navConfig(mode, {childIds})` gives امروز · mode tab ·
  FAB · خدمات · من per the nav.md table (postpartum «کودک» → `/children/<id>` for one child else `/children`;
  companion: no mode tab, no FAB); `activeTabKey` lights `/services/*` → خدمات and `/cycle`, `/analysis*` → mode tab.
  N2/N4/N5 only need the API to return the new `mode` strings (and pass `childIds` once N5 has children).
- **Where it shows:** `shared/config/app-nav.ts` (`NAV_ROOT_PATHS` / `NAV_ROOT_PREFIXES`, `isNavRootPath`). Screens
  keep mounting `<BottomNav />`; it renders `null` elsewhere (now hidden on `/pregnancy/alerts`, `/pregnancy/log`).
  `/cycle`, `/log`, `/pregnancy/calendar` are listed as *transitional* (no back button yet) — their restyle tasks
  remove them. A new hub = one line there. The nav floats (absolute in `.view`); `.view:has(> .nbnav) > .scroll::after`
  adds the clearing tail, so screens need no padding change.
- **FAB** → `openSheet('log')`; `log` registered in `app/sheets/registry.tsx` (half) with interim content
  `LogSheet` from the widget (mode's existing log routes as `TileButton` cards + add-reminder sheet). **B-N3-03**:
  build `screens/log-sheet`, point the registry entry at it (size `full`), delete `LogSheet` from the widget.
  `nav` added to `SHELL_NAMESPACES`.
- **`/services`** (`screens/services`, route `app/[locale]/services`, namespace `services`): placeholder hub —
  HubHeader + bell (notifications sheet), «مراقبت سلامت» grid as «به‌زودی» tiles, live rows to `/checkups` and
  `/reminders`, «اورژانس است؟» `tel:115` card, SkyLayer. **B-N7-01** replaces the page body.
- **Docs:** `frontend/CLAUDE.md` §4.1 rewritten (routes = back-header screens, sheets = short decisions; nav rules).
- **i18n:** `messages/{fa,en}/nav.json` (+tabs, badge, logSheet), new `services.json`; copied to
  `backend-go/resources/translations/` and `internal/i18n/testdata/messages_{fa,en,ar}.json` re-recorded for those two
  namespaces only.
- **Outside `touches`** (minimal, flagged): `app/sheets/registry.tsx` (log entry), `shared/i18n/bundled.ts` +
  `src/global.d.ts` (services namespace), `backend-go/resources/translations/{fa,en}/{nav,services}.json`,
  `backend-go/internal/i18n/testdata/messages_*.json`.
- **Screenshots:** `docs/qa/bloom/B-N1-04/` full-page cycle persona 04 (home, services, profile, `?sheet=log`),
  `viewport/` (04 incl. `/en/services`), `ttc/` (12: باروری tab + TTC log sheet), `pregnancy/` (14: بارداری tab on
  `/pregnancy/weeks`, pregnancy log sheet), `artboards/` (Cycle_Home, v17_Main, Me_Hub light/dark).
- **Open:** QUESTIONS #16–#18. Old nav keys `cycle`/`profile` in `nav.json` unused now (kept for admin-edited
  translations). The PWA install banner overlaps the floating nav on first visit (pre-existing `shared/pwa`).

## B-N1-05 — Splash, intro slides and welcome

- **Splash** (`screens/auth-splash`): violet-gradient splash replaced by the `Splash` artboard — page canvas in both
  themes, 29-dot cycle ring around the Lalezar wordmark, centre glow, tagline, three pulsing loading dots
  (`role=status`, static under reduced motion). Timer, tap-to-continue and the route's no-JS fallback unchanged.
- **Intro** (`widgets/intro-carousel`): 5 swipeable slides `Intro_1…5` (cycle ring · pregnancy 40-week ring · 2×2
  health tiles · privacy rows · «همیشه رایگان» card + chips). Header page-dots + «رد کردن» (jumps to slide 5, as drawn);
  footer `PrimaryButton` «بعدی / شروع کن» + text button «حساب دارم · ورود». Track follows the reading direction (RTL:
  next slide from the left, rightward swipe advances); off-screen slides `inert` + `aria-hidden`; live region.
  Rings are inline SVG from token classes: `ui/DotRing.tsx` (`CycleDotRing` exported for splash/welcome,
  `PregnancyDotRing`), geometry in `lib/ring.ts` (unit-tested against artboard coordinates). Icons = artboard paths.
- **Welcome** (`screens/welcome`): `Onb_Welcome` card (`WelcomeCard`). `/welcome` shows the slides to first-time
  visitors, the card once the intro was seen or with `?step=welcome`; `?slide=N` opens a slide (QA/deep link) —
  `lib/view.ts` (+ test). Start / sign-in both mark the intro seen → `/signup`.
- **Copy:** `welcome` namespace rewritten (fa/en, artboard text); `auth.splash` → `brand/tagline/loading`
  (`copyright` dropped). Backend seed `backend-go/resources/translations/{fa,en}/{welcome,auth}.json` synced and
  `internal/i18n/testdata/messages_{fa,en,ar}.json` patched for those two keys only (`go test ./internal/i18n` green).
  The Laravel-era `contract/golden/public/languages_messages_*` goldens were not re-recorded (already stale, see
  their `source` note). The running dev API embeds the seed → restart it to serve the new strings.
- **CSS:** one `/* B-N1-05 … */` block in `globals.css` (`ib-*`) replacing the old `.splash*` and `.ic-*` rules.
- **Outside `touches`** (minimal): `app/message-scopes.ts` (`splash` + `welcome`, the splash imports the widget),
  `messages/{fa,en}/{welcome,auth}.json`, backend-go translation seed + i18n testdata.
- **Screenshots:** `docs/qa/bloom/B-N1-05/` (splash, slides 1–5, welcome card × light/dark, `en` slide 5) +
  `artboards/`. Fidelity: layout, sizes and colours match; the artboards' 54px fake status bar is not reproduced
  (safe-area inset instead), light uses dialect A tokens.
- **Open:** QUESTIONS #13–#15 (welcome-card placement / signup back link, dropped AI disclaimer, skip target).
  `--splash-from/to` tokens are now unused (left for B-N1-02 owners). No loading/error states apply (static,
  no data).

## B-N1-10 — «من» hub, account, appearance, language

- **Hub** `/profile` (`screens/profile/ui/ProfilePage.tsx`, rewritten on nb primitives): Lalezar «من» + appearance
  `HeaderButton`, profile card (initial avatar, masked phone `۰۹۱۲ ••• ••۴۵` via `me-format.ts`, mode pill toned
  per `resolveNavMode` — drawn inside the card, the artboard's nested `<a>` pushed it out), Plus card (`isPremium`
  from `/messages/mode`, else «به‌زودی»), groups من و خانواده · کارها و خریدها · داده و دستگاه · تنظیمات ·
  پشتیبانی, «خروج از حساب» text button. Live rows: mode (pregnancy setup / back to cycle, as before), backup →
  `GET /profile/export`, privacy → `?sheet=info&privacy`, notifications → `?sheet=notifications`, appearance,
  language, help/about → info sheet. «به‌زودی» (static, not buttons): companions, children, courses, todo,
  bookings, orders, gadgets, support chat. Skeleton (rendered until mount → no hydration mismatch) + error card.
- **Account** `/profile/account` (`AccountPage`, Me_Profile): 96px initial avatar + camera badge, field card
  (name/birth date → `QuickEditSheet`; family name, phone change, email = «به‌زودی»), subscription + devices rows,
  the old inline cycle & health rows as a group below (QUESTIONS #20), period-soft logout pill, delete account →
  `DeleteAccountConfirm`.
- **Appearance** `/profile/appearance` (`screens/appearance`): theme radiogroup with dark/light previews (roving
  arrows) + «هماهنگ با تنظیمات گوشی» `Checkbox` = `system`; text-size range (5 stops); reduce-motion `Switch`;
  haptics row hidden on web. **Language & calendar** `/profile/language`: language `RadioCardGroup` from
  `GET /languages` + read-only calendar row (QUESTIONS #21).
- **shared/theme:** new `display.ts` store (`useDisplayStore`: `textScale` index into `TEXT_SCALES`, `motion`
  `system|reduce`; keys `ritme_text_scale`, `ritme_motion`), applied by `ThemeApplier` (+ cross-tab) and before
  paint by `themeInitScript` (unit-tested against a fake DOM, `display.test.ts`). CSS: `.view > .scroll { zoom:
  var(--text-scale) }`, `html[data-motion="reduce"]` kill switch.
- **CSS/tokens:** `/* B-N1-10 */` block at the end of `globals.css`; theme-stable preview tokens `--theme-pv-*` in
  both token blocks.
- **i18n:** new `me` namespace (`messages/{fa,en}/me.json`, `bundled.ts`, `global.d.ts`), copied to
  `backend-go/resources/translations/{fa,en}/me.json`; `internal/i18n/testdata/messages_{fa,en,ar}.json` got the
  `me` block (golden test green). Routes `profileAccount/Appearance/Language` in `message-scopes.ts`.
- **Outside `touches`** (minimal, flagged): `shared/ui/Icon.tsx` (+10 glyphs from the artboards: users, gradCap,
  todo, box, watch, help, chat, crown, smartphone, modeRing), `app/[locale]/profile/{account,appearance,language}`,
  the shared i18n/css files above.
- **Screenshots:** `docs/qa/bloom/B-N1-10/` fa hub/account/appearance/language × light+dark, `en_profile*`,
  `side-by-side_Me_{Hub,Profile,Appearance}.png` (artboard light | app light | artboard dark | app dark), `artboards/`.
- **Open:** QUESTIONS #19–#23. `screens/profile-personal` / `profile-language` sheets are now unreferenced from the
  hub (still registered for old `?sheet=` links). The PWA install banner still overlaps the nav (pre-existing).
