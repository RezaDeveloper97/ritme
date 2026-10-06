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

## B-N1-14 — Restyle pregnancy v2 screens to Night & Bloom

- **Screens** (all on nb primitives + `SkyLayer`, back → `/pregnancy` via `router.push`):
  `/pregnancy` (PregFull_Main: `HubHeader` Lalezar «بارداری» + soft bell w/ unread dot, 40-dot week ring `ui/WeekRing.tsx`
  with days-to-due in Lalezar 64, trimester + confidence outline pills, due-date card (edit → setup), 4 tone tiles,
  «یادآورهای امروز» card (v13_Preg_Home; B-N1-15's widget, gutter neutralised), next visit, tinted smart tip, care list);
  `/pregnancy/weeks[/n]` (PregFull_Week: header «بارداری — هفتهٔ N» + bookmark, `SegmentedTabs` prev/this/next week
  (route-driven, swipe kept), bloom-tint size hero, Lalezar stat cards, `InfoNote`s, baby / «ممکنه حس کنی» / tasks
  sections, warning, reviewer note + sources); `/pregnancy/log` (PregFull_Log: mood + symptom `PillChip`s (brand/warm),
  severity radios, 8-glass water row + −/+, big weight input, period-tint spotting toggle card, note, glass sticky save
  footer; v1 `?tab=` page got `ScreenHeader` + `SegmentedTabs`); `/pregnancy/calendar` (PregFull_Calendar: header +
  «ویزیت جدید» button, next-visit card with stage pill/prep/stepper/44px reminder+directions pills, care plan rows with
  status pills / «رزرو», month grid, selected day, PDF `SecondaryButton`, source `InfoNote`); `/pregnancy/alerts`
  (PregFull_Alerts: follow-up/urgent = tone-tinted card, inline facts, outline actions + «دیدم، ممنون» text button;
  info/suggestion = compact pill rows; legend with tone discs); `/pregnancy/setup` (PregFull_Setup: 4-step
  `ProgressSteps` in bloom, welcome card, basis chips (radio) + field boxes, history chips + select boxes, result card).
- **States:** `SkeletonGroup` loading, danger-disc error card + retry, `EmptyState` (not active / no alerts) on every
  screen; all hit targets ≥ 44px (chips 44, glasses 44, action pills 44).
- **Widget** `pregnancy-care-checklist` rebuilt (disc rows, `variant="today"|"week"`, `title`); `PregnancyCheckBox`
  export removed.
- **CSS:** one `/* B-N1-14 */` block (`pgn-*`) at the end of `globals.css`. `.pg2-*` left for the illustrations.
- **Copy:** 17 new `pregnancyV2` keys (fa/en) — frontend `messages/{fa,en}/pregnancy-v2.json`, backend seed
  `backend-go/resources/translations/{fa,en}/pregnancy-v2.json`, goldens `internal/i18n/testdata/messages_{fa,en,ar}.json`
  (ar = fa) patched for those keys only; `go test ./internal/i18n` green. Restart the dev API to serve them.
- **Outside `touches`** (flagged): `screens/pregnancy-onboarding` (the real home of `/pregnancy/setup`; `touches` names a
  non-existent `pregnancy-setup`), backend-go translation seed + i18n testdata (acceptance requires sync).
- **Screenshots:** `docs/qa/bloom/B-N1-14/` — 6 screens × light/dark (persona 15), `p14/` (week-8 persona + `/en`
  LTR), `side-by-side_*.png` (artboard light | app light | artboard dark | app dark), `artboards/`.
- **Open:** QUESTIONS #24–#27. Unused, left for deletion (deletion was blocked in this session):
  `widgets/pregnancy-week-carousel/`, `screens/pregnancy-week/ui/WeekStrip.tsx`. v1 Symptoms/Weekly/Movement forms
  inside `?tab=` still use `features/track-pregnancy` old controls. Persona 15 has no booked visit / follow-up alert, so
  those card variants were checked in code only.

## B-N1-15 — Restyle reminders (v13) and checkups (v14) screens

- **Artboards = M3/M4 designs:** every `c-health-record/*_v13_*`/`*_v14_*` file is byte-identical to
  `docs/design/{reminders-v13,checkups-v14}` except `v14_Main` (dialect-A colours only), so this is a fidelity pass on
  the nb primitives rather than a relayout.
- **Screens** reminders hub, medication form, appointment form/detail, checkups list/detail/history/self-exam:
  custom `.rmd-hdr` → `ScreenHeader` (+ `HeaderButton` bell/filter/edit; the checkup-reminder bell and history export
  stay `role=switch`/disabled-able buttons on `.nb-hbtn`), `SkyLayer` on every screen (`.rmd-screen`), `.rmd-tabs` →
  `SegmentedTabs` (reminders/checkups/history; panel labelled by aria-label), `.rmd-cta`/`.btn` → `PrimaryButton` /
  `SecondaryButton` (incl. picker sheet footers, MarkDone, delete confirm = `.rmd-danger-btn`), every hand-rolled
  `role=switch` → nb `Switch compact` with a 44px hit area (`.rmd-hit`), text/`skeleton-line` loading → `Skeleton` +
  `SkeletonGroup`, empty checkups/history → `EmptyState`. Detail rows: 36px round inset discs, «مسیریابی» outlined pill.
- **Home cards:** `checkups-card` header rebuilt per `v14_Main` (ring · title + counts · «همه»), highlight rows r18,
  pill action; `today-reminders` skeleton → `Skeleton`. `.card` inside these screens → radius 24.
- **CSS:** one `/* B-N1-15 */` block after `end B-N1-04` in `globals.css`. No i18n changes (fa/en/backend untouched).
- **Data:** persona 09900000018 got 2 appointments (8 & 20 Mehr), a 2nd medication, 3 checkup records (ritme_dev).
- **Screenshots:** `docs/qa/bloom/B-N1-15/` full page light+dark (reminders, med new/edit, appt new/detail, checkups,
  detail, history, self-exam, home, `en_reminders`, `en_checkups`), `sheets/` (AddChooser, MarkDone), `artboards/`.
- **Outside touches:** only `globals.css` + bloom docs. `screens/checkup-custom-form` (not in touches) still has the old
  `.rmd-hdr` header; it inherits the radius/skeleton CSS only. QUESTIONS #29–#30.

## B-N1-11 — Notification settings — categories, quiet hours, discreet copy

- **Backend (Go):** migration `00011_notification_preferences` (+ Laravel mirror `2026_10_01_000002_…`, schema-diff
  green), `internal/notifications` (prefs load/save, `Decide` = category + quiet hours, `Render` = neutral copy, unit
  tests), `GET/PUT /api/v1/profile/notification-settings` (partial PUT, Laravel-style 422 with localized attributes),
  OpenAPI, contract group `notification-settings` (Go-recorded goldens; empty table added to `contract/fixtures/dump.sql`).
- **No push sender exists yet** anywhere; any future push/web-push sender must call `Load → Decide → Render`.
- **Frontend:** new `screens/notification-settings` (optimistic instant-save, rollback on error) at
  `/profile/notifications`; Me row now links there (was the inbox sheet). Copy `me.notifSettings` fa/en + backend seed +
  goldens. `.ntf-*` block in `globals.css`.
- **Screenshots:** `docs/qa/bloom/B-N1-11/` (fa/en light+dark, artboards/). **Open:** QUESTIONS #31–#36.

## B-N1-07 — Calendar redesign (cycle + TTC variants)

- `screens/calendar` rewritten: month ⇄ year tabs (`?view=year`), two stacked months, legend, swipe, skeleton / empty /
  error, `DayCard` (cycle day + phase pill + logged chips; TTC: pregnancy chance + «ثبت جزئیات این روز» → `/fertility/log`);
  old period actions moved into the day card. `DayLogSummary` deleted. New widget `widgets/cycle-calendar`
  (`MonthCard`, `YearView`, `CalendarLegend`, tones + tests; steiger exception added). Copy fa/en + backend seed.
- **Screenshots:** `docs/qa/bloom/B-N1-07/` (cycle persona 04, TTC persona 12, empty, error, artboards, before).
- **Env:** `ritme_dev` rebuilt from the contract dump (docker was down). **Open:** QUESTIONS #37–#38.

## B-N1-06 — Cycle home redesign — normal / near period / during period

- **Backend:** `GET /api/v1/home/cycle-overview` (`internal/home/cycle_overview.go` + tests incl. 401, OpenAPI). Go-only.
- **Frontend:** `HomePage` hero states from engine `cycle_view.main_phase` (`hero-state.ts` + tests): start/end-period
  prompts (`useStartPeriod`/`useEndPeriod`, per-day dismiss), «پایان پریود هنوز ثبت نشده» banner, error/skeleton,
  `TodayLogCard` (not in TTC), `PredictionsCard`, `PmsInsightCard`, restyled today-challenge. `CycleValuesCard` only
  when profile length differs from recent cycles. `Icon` union + `symptom`, `star`. `/* B-N1-06 */` CSS block.
- **States on dev data (2026-10-01):** normal = persona 04, near = 06, during = 07 (also `X-Test-Now`).
- **Screenshots:** `docs/qa/bloom/B-N1-06/{normal,near,during}/`, `artboards/`. **Open:** QUESTIONS #39–#41.

## B-N1-08 — Edit period, cycle history, symptom pattern and phase sheet

- **Backend (Go-only):** `internal/cycle/insights` — `GET /api/v1/cycle/history` (median cycle/period length,
  variability, regularity verdict, per-cycle bars with in-range flag; last 6 complete cycles) and
  `GET /api/v1/cycle/symptom-pattern` (heat strip per symptom over the typical cycle, groups symptoms/mood/pain, needs
  ≥3 cycles). Built on `cycleservice.Load`, no new queries. Unit + integration (IDOR) tests, OpenAPI. Routes in
  `routes_cycle_insights.go`. No contract golden (Go-only, same as B-N1-06).
- **Frontend:** `features/log-period/PeriodDateEditor` redesigned (month grid, dashed suggestions, first-day tick,
  «فقط لکه‌بینی است» → `spotting` health-log; selection model + tests; props unchanged + optional `intent`).
  `screens/cycle` rewritten (`/cycle` with ScreenHeader, no bottom nav — `/cycle` removed from `NAV_ROOT_PATHS`), new
  `CycleSymptomsPage` at `/cycle/symptoms`, `screens/phase-details` sheet with 5 tabs from phase-contents. Unused
  `BmiCard`, `CycleSummaryCard`, `MyCyclesCard`, `SectionHead` deleted. `/* B-N1-08 */` CSS block.
- **Screenshots:** `docs/qa/bloom/B-N1-08/`. **Open:** QUESTIONS #46–#49.

## B-N1-12 — Privacy & security, support, about, legal

- **Backend (Go-only):** migration `00012_privacy_support` (`user_consents`, `support_reports`, seeded info-section keys
  `privacy/summary`, `terms/summary`, `about/disclaimer`, `support/email`; Laravel mirror `2026_10_01_000003_…`).
  `GET/PUT /api/v1/profile/consents`, `POST /api/v1/support/reports` (multipart or data URL; re-encoded WebP in
  `STORAGE_PATH/app/private/support-reports`, 0600; ≤12 Mpx, 2 concurrent encodes → 503, Redis limiter 5/h → 429,
  bad image → 422), public `GET /api/v1/info-pages/{group}` (new `support` group). Screenshots removed on self and
  admin account deletion (`profile.RemoveSupportFiles`). `GET /profile/export` now includes consents, support reports
  (no file path) and notification settings — **D-33 proposed** (contract allow-list). Contract group `privacy`.
- **Frontend:** `features/app-lock` (PBKDF2 passcode in localStorage, optional WebAuthn local check, lock on open /
  resume after 0/1/5/15 min, pre-paint `html[data-app-locked]`, gate above routes + sheets, app not hydrated while
  locked, lock-out + logout after 10 failures, tests for reload/back bypass), hide-preview blur,
  `features/manage-account` PDF export (on-device) + restyled delete. Screens `privacy`, `support` (+ `ReportSheet`),
  `about` (About + Legal) at `/profile/{privacy,support,about,legal}`; Me rows linked.
- **Security audit:** no Critical/High; M1/M2/L1–L3 fixed; L4 (no push sender uses the notification gate yet) noted.
- **Screenshots:** `docs/qa/bloom/B-N1-12/`. **Open:** QUESTIONS #42–#45, #50; follow-up **B-N1-12b** (admin inbox).

## B-N1-09 — Cycle settings screen with reminder preferences

- **Backend:** migration `00013_cycle_settings` (`notification_preferences.schedule` json, `cycle_preferences.lengths_auto`;
  Laravel mirror `2026_10_01_000004_…`). `GET/PUT /api/v1/profile/cycle-settings` (engine medians or manual values via
  `Service.Save`, bumps `calculation_version`). Engine honours manual lengths: `Profile.LengthsManual` (from
  `GetEngineProfileByUserID` LEFT JOIN), `metrics.resolveManual`, cache key gets `lengths_manual` only when on — auto
  path byte-identical (`make contract ROUTES=all` 1061 passed). TTC predictions read the flag too.
  `internal/notifications` gained `Pill` (not in the settings groups) + `Schedule` (times, days-before).
  New `internal/reminders.Plan` = the day's reminders through `notifications.Decide` (no real sender yet).
  Contract group `cycle-settings` (Go-recorded).
- **Frontend:** `screens/cycle-settings` at `/cycle/settings` (auto toggle, lengths, reminder rows with inline time /
  days-before editor, mode section: pregnancy → `/pregnancy/setup`, others «به‌زودی»). Linked from Me hub and the home
  header gear. `cys-*` CSS block.
- **Screenshots:** `docs/qa/bloom/B-N1-09/`. **Open:** QUESTIONS #51–#53; `checkups` builds its own profile row and
  ignores the manual flag (only coarse dates used).

## B-N1-12b — Admin — support reports inbox and support info group

- **Backend:** `/api/admin/v1/support-reports` list (`status=open|resolved|all`, counts, 160-char preview only), detail,
  `/:id/screenshot` (private stream, `no-store`, path check via shared `profile.SupportFilePath`), `resolve`/`reopen`
  (CSRF, audit log). Any active admin. Info-section box `key` editable (slug, unique per group). No migration
  (status came with 00012). Documented in `docs/go-migration/admin-api.md` §14; integration tests.
- **admin-web:** «گزارش‌های مشکل» list/detail pages + sidebar entry; `support` group + «کلید» field in info-sections.
- **Screenshots:** `docs/qa/bloom/B-N1-12b/` (1440px light+dark). **Tooling TODO:** `shot.mjs --admin` has no admin
  login, wrong theme key (`ritme_admin_theme`) and a clipped RTL capture — fix before the N9 admin tasks.
- **Open:** QUESTIONS #54–#55.

## B-N1-16 — N1 design-fidelity audit and fixes

- `docs/night-bloom/audit-n1.md`: 1 high, 7 med, 9 low; all high/med + 2 low fixed, 7 low open (X4 `--on-danger`, H5
  banner image `onError`, H7 new-user ring dotted path, C4 phase-sheet close row, T2 TTC calendar filter icon, P4
  pregnancy sources contrast, M1 Me mode pill full-width). Deliberate defaults listed as «accepted (QUESTIONS #n)».
- Fixes: column scrollers no longer shrink bottom buttons (`.scroll > * { flex-shrink: 0 }`), sheets on `--surface`,
  LTR chart axes in `/cycle` + `/cycle/symptoms`, BBT value side in RTL, flat home article/reminder/checkup cards,
  `--on-accent` → `--on-brand` on brand fills (onboarding pages, DayTasks, PeriodButton, DailyStatusCard, WeekStrip),
  phase sheet small title sr-only, PregnancyAlerts gear → `/profile/notifications`.
- Screenshots: `docs/qa/bloom/B-N1-16/` (personas 01/04/06/07/12/15/18, public, `side-by-side/`; rendered artboards not
  committed — regenerate with `shot.mjs --files`).
- Env: `ritme_dev` was missing tables of goose 00002/00003/00005 (+ data of 00007/00008) despite goose rows; Up applied
  by hand (dev only).

## B-N2-01 — Profile & onboarding schema v2 and extended life-stage modes

- **Migration** `00014_life_profiles` (+ Laravel `2026_10_01_000005_…`): `user_life_profiles` (gender, life_mode
  cycle|ttc|pregnancy|postpartum|menopause|teen, `ivf_iui`, `track_contraception`, JSON `chronic_illnesses` /
  `gyn_conditions` / `medications` (NULL = skipped, [] = none), menopause stage/last period/surgical/HRT, onboarding
  started/completed). `user_profiles` untouched; no backfill — users without a row resolve exactly as before
  (`enums.ResolveLifeMode`: active pregnancy → stored postpartum/menopause/teen → user_goal ttc → cycle).
- **Endpoints (Go-only):** `GET /onboarding`, idempotent `PUT /onboarding/steps/{name|gender|goal|cycle|menopause|conditions|health}`,
  `POST /onboarding/complete` (422 lists missing steps), `GET|PUT /profile/life-stage` (B-N2-03 must reuse it).
  Export gains `life_profile` — **D-34 proposed**. Message engine: postpartum → empty, menopause/teen → cycle, non-TTC.
- Contract group `onboarding` (20 Go-recorded goldens), existing goldens byte-identical (`ROUTES=all` 1081 passed).
  fa/en enum labels `onboarding.enums.*` in seed + frontend messages + goldens.
- **Open:** QUESTIONS #59–#61.

## B-N1-17 — N1 rollout — verify-all, stage deploy, e2e smoke

- verify green on the combined tree; `stage` pushed (`d7bc901`) and deployed with `./deploy-stage.sh` (all checks ok;
  goose applied 9–13, version 13). Smoke on https://stage.ritmeapp.ir, fa 390px light + dark: 38 checks, 36 pass,
  2 warn, 0 fail; no 4xx/5xx or console errors; admin `/panel/support-reports` received an in-app report end to end.
- Stage has no `is_test` OTP; codes read read-only from `ritme_stage`. Report: `docs/qa/bloom/n1-stage.md`, shots in
  `docs/qa/bloom/n1-stage/`.
- Low bugs: B-1 → B-N2-10 scope; B-2 (pregnancy deep link to /home) and B-3 (pregnancy setup guard) → B-N2-03.
  QUESTIONS #62–#63.

## B-N2-04 — Subscription domain — plans, trials, subscriptions, discounts, entitlements

- **Migration** `00015_plus_subscriptions` (+ Laravel `2026_10_01_000006_…`): `plus_plans` (translatable title/badge,
  duration, `price_rials`, optional `monthly_display_rials`, highlighted, active, sort; 3 seeded per `nbl_Prem_Plans`),
  `plus_trials` (unique user), `plus_discount_codes`, `plus_invoices` (random reference, `vat_rate_bps` snapshot),
  `plus_receipts` (unique gateway+ref), `plus_subscriptions`, `plus_usage_counters`. Money = integer rials.
- **Config:** `PLUS_VAT_RATE_BPS` (1000), `PLUS_TRIAL_DAYS` (7), `PLUS_INVOICE_TTL_MINUTES` (30), `PLUS_CALLBACK_URL`.
- **Endpoints (Go-only, D-35 proposed):** public `GET /plus/plans`; auth `GET /plus/{status,usage,history}`,
  `POST /plus/{trial/start,checkout,verify,cancel,restore}` (checkout `preview:true` for discount check; 100% discount
  settles at once; verify idempotent). Payment port `plus.Gateway` + in-package `FakeGateway`; no gateway in prod →
  503 `payment_unavailable`.
- **Entitlements:** `internal/plus/entitlements.go` (`Resolve`/`ResolveOne` pure + tests), `Service.Entitlement` /
  `Service.Consume` (atomic counters, `ErrNotEntitled` / `ErrQuotaExceeded`) for later tasks.
- **Security tests:** amount tampering, replayed/concurrent verify (row lock), reused bank ref, trial race, IDOR,
  discount caps under concurrency (fixed a REPEATABLE READ bug → READ COMMITTED). Contract group `plus` (25 goldens).
- **Open:** QUESTIONS #64–#67. B-N2-05 must add `PLUS_*` to `.env.stage.example`.

## B-N2-03 — Life-stage mode switcher (6 modes) and minimal menopause/teen homes

- **Backend:** `GET /api/v1/profile/life-stage/loss-copy` (admin `message_contents` `pregnancy_setup/loss_exit`, 7 keys,
  null → bundled fallback; `loss_exit` registered in the admin messages registry). Reuses `GET|PUT /profile/life-stage`;
  pregnancy coordination in the client (enter: PUT mode → `/pregnancy/setup`; exit: `POST /pregnancy/deactivate` → PUT).
- **Frontend:** `entities/user` life-stage hooks (`useLifeStage`, optimistic `useUpdateLifeStage`, `useLossCopy`, local
  hint cleared on logout); `screens/mode` at `/profile/mode` (6 radio cards, IVF/IUI + contraception switches, confirm
  sheets, calm loss exit at `/profile/mode/loss` → cycle mode, no celebration). Bottom nav reads the life-stage mode
  (`useNavMode`, `NAV_READY` flags: postpartum → `/home` + calendar tab; menopause «علائم» → `/cycle/symptoms` until
  B-N3-08). Home is mode-aware: pregnancy → `/pregnancy` (stage bug B-2), `MenopauseHome`, teen = cycle home without
  banners/fertility/PMS, postpartum = cycle home + «به‌زودی» card. Me hub mode row + pill, Plus card hidden for teen;
  cycle-settings mode rows wired; pregnancy setup guard (stage bug B-3). Missing-artboard decisions in
  `docs/night-bloom/README.md`.
- **Screenshots:** `docs/qa/bloom/B-N2-03/` (menopause 0990…81, teen 0990…82, cycle 04, ttc 12, pregnancy 15).
- **Open:** QUESTIONS #68–#70. No contract golden for loss-copy (Go-only; int + OpenAPI tests).

## B-N2-05 — Payment gateway adapter (fake + web bank gateway)

- `internal/payments`: `Provider` (Create/Verify/Refund/ReturnParams) + `Gateway` wrapper (input validation, callback
  allow-list, PAN masking, slog audit `payments: create|verify|refund`, no PII). Providers: `fake` (Redis state, TEST
  page `GET/POST /api/v1/payments/fake/pay/{authority}`, refunds) and `zarinpal` (v4 REST, IRR, code 101 idempotent,
  refunds unsupported). Return hop `GET /api/v1/payments/{provider}/return` (no state change, 303 to allow-listed
  front-end URL, no open redirect); settlement stays in authenticated `POST /plus/verify`.
- Env: `PAYMENT_PROVIDER` (fake outside prod, `none` in prod; fake+production refused at startup),
  `PAYMENT_CALLBACK_BASE_URL`, `PAYMENT_RETURN_URLS`, `PAYMENT_HTTP_TIMEOUT_SECONDS`, `ZARINPAL_MERCHANT_ID` (server
  only), `ZARINPAL_SANDBOX`, `ZARINPAL_BASE_URL`; `PLUS_*` added to `.env.stage.example` and `docker-compose.stage.yml`.
- No migration (00016 unused; invoices/receipts are the durable record). Contract `plus` 25 passed. QUESTIONS #71.

## B-N2-06 — Plus gating, trial offer engine and usage counters

- **Migration** `00018_plus_settings` (key/value, seeded `trial_offer_percent=50`; Laravel `2026_10_01_000018_…`).
  00016 intentionally unused; 00017 belongs to the canvas session (CB-CONTRA-01).
- **Gate:** `plus.NewGate(svc, clock).Require(plus.<Key>)` → 402 `plus_required` (`reason` locked|quota, `limit`,
  `resets_at`) or 429 `plus_quota_exceeded`; handlers `svc.Consume` then `gate.GateError`. OpenAPI responses
  `PlusRequired`, `PlusQuotaExceeded`.
- **Trial offer:** `PlusTrialOffer` (percent, ends_at, seconds_left, countdown, featured plan with offer prices) in
  `GET /home/cycle-overview` `data.plus_trial_offer`, `GET /plus/status` `data.trial_offer`, new `GET /plus/trial`
  (trial, offer, plans with offer prices, usage since trial month). Checkout prices at the offer server-side;
  `discount_source` code|trial_offer on quotes/invoices. Admin hooks `svc.TrialOfferPercent` / `SetTrialOfferPercent`.
- Contract `ROUTES=all` 1111 passed; schema-diff OK. QUESTIONS #72.

## B-N2-02 — Onboarding flow v2 (women path)

- New `screens/onboarding-flow` (flow model + branching, pregnancy dating maths, zod state, API for `GET/PUT/POST
  /onboarding*`, steps Name/Gender/Goal/Cycle/Pregnancy/Menopause/Conditions/Health/Partner(stub)/Ready, tests).
  `features/auth` `OnbFrame` + `useWebOtp`; signup (+98, two consent ticks) and OTP (WebOTP, paste, resend timer)
  rewritten. Routes under `/onboarding/*` (new gender/cycle/menopause/health/partner; old birthday/height/weight →
  health, cycle-len/period-len/cycle-duration → cycle). 11 old onboarding slices deleted.
- Messages: `auth.phone|code`, `common.onbFrame`, `onboarding.flow` (dead keys removed), seed + goldens.
- E2E in browser: TTC → `/home`, pregnancy → `/pregnancy`. Screenshots `docs/qa/bloom/B-N2-02/`. QUESTIONS #73–#75.

## B-N2-09 — Admin — subscriptions & payments module

- **Migration** `00019_plus_admin_actions` (+ Laravel `2026_10_01_000019_…`): audit + manual refund notes + gateway
  refund ids. `plus_settings.vat_rate_bps` override (null → env), read by checkout and `/plus/plans`.
- **Admin API** `/api/admin/v1/plus/*`: plans + discount codes CRUD (delete → deactivate when referenced), settings
  (trial %, VAT), subscriptions list + `extend` (1–365 days, shifts queued periods), payments list/detail + `refund`
  (gateway refund via `payments.Gateway.Refund`, or manual with note; subscription → refunded). Reads any admin, writes
  super; CSRF; audit rows + slog; masked mobiles; never `card_pan`. Docs: `admin-api.md` §15. Int tests (9).
- **admin-web:** «اشتراک‌ها و پرداخت» group (subscriptions, payments, plans, discount codes, settings), toman input.
- **Tooling:** `bloom/bin/shot.mjs --admin` fixed (`--admin-email/--admin-password` login, `ritme_admin_theme`,
  1440×900 tall viewport instead of clipped RTL capture).
- Screenshots `docs/qa/bloom/B-N2-09/`. QUESTIONS #76.

## B-N2-07 — Plus screens — paywall, plans, checkout, success, manage

- `entities/plus` (zod schemas incl. optional B-N2-06 fields, query hooks, `formatToman` rials→toman with Persian
  digits, plan/upgrade/return-param helpers, `PlusCrown`), `features/purchase-plus` (checkout, verify, cancel, restore,
  trial start, `goToGateway` http(s) only), screens `plus-{paywall,plans,checkout,success,manage}`, routes `/plus`,
  `/plus/{plans,checkout,success,return,manage}` (return → `POST /plus/verify`). Me hub Plus card → `/plus` or
  `/plus/manage` (hidden for teen; teen redirected away from paywall/plans/checkout). New `plus` namespace registered.
- Full fake-gateway purchase walked light + dark: `docs/qa/bloom/B-N2-07/{with-code,no-code,me-hub,artboards}`.
- Orchestrator: removed the artboard placeholder testimonial («سارا» + stars); fixed i18n goldens (`me`, `plus`, and the
  stale `care` namespace that made `TestBundle_MatchesLaravelGoldens` red at HEAD).
- Local dev needs `PAYMENT_CALLBACK_BASE_URL=http://127.0.0.1:8020` (APP_URL has no port). QUESTIONS #77.

## B-N2-08 — Trial banner, trial sheet and Plus locks across the app

- `entities/plus` extended: trial-offer / trial-sheet schemas + `usePlusTrial`, `formatTomanThousands`,
  `useServerCountdown` (from server `seconds_left`, minute re-render, `onExpire`), `PlusFeatureGate` (entitlements from
  `/plus/status`, unlock → `/plus`), `usePlusLocked`.
- `shared/ui/plus-gate`: `plusDenialOf(error)` (402 `plus_required` → locked|quota, 429 → exhausted), `upgradeHelps`,
  `<PlusBadge>`, `<PlusGate locked|denial …>` (blurred teaser + unlock). Any route using it needs the `plus` namespace.
- `widgets/plus-trial-banner` (floating above the nav, live countdown, invalidates `plusKeys.all` at zero) and
  `widgets/plus-trial-sheet` (usage list, featured plan with offer price, CTA → `/plus/checkout?plan=`). Wired into cycle
  home (not teen) and menopause home via `screens/home/ui/PlusTrialOffer.tsx`.
- Screenshots `docs/qa/bloom/B-N2-08/`. QUESTIONS #78.

## B-N2-10 — N2 design-fidelity audit and fixes

- `docs/night-bloom/audit-n2.md`: N2 screens 0 high, 2 med, 6 low; fixed 2 med + 2 low (Plus manage ring side, teen
  home fertility/ovulation markers removed, trial-sheet compare link `--brand`). 4 N2 lows + 5 N1 lows open.
- Leftovers: new `--on-danger` token (light/dark, contrast pair in `check-dark-mode.mjs`, used on every text-on-danger),
  banner slide dropped on image `onError`, `/profile/notifications` PMS subtitle reads the real `cycle_day`
  (`usePmsReminderDay` via `/profile/cycle-settings`).
- Screenshots `docs/qa/bloom/B-N2-10/`. QUESTIONS #79.

## B-N3-01 — Log taxonomy v2 (backend) with backfill

- **Migration** `00020_health_log_entries` (+ Laravel `2026_10_01_000020_…`): one row per (user, day, category, param,
  item) with `value_code` / `value_num` / `value_text`, `source` manual|legacy|voice; unique + analysis index. Backfill
  from `daily_health_logs` generated from the Go mapping (`taxonomy.BackfillSQL()`), idempotent, Down drops only the
  new table (legacy table untouched).
- `internal/healthlog/taxonomy`: 18 categories, typed params, mode lists (6 modes), conditions, `link` tiles for N5.
- **Endpoints (Go-only, D-37 proposed):** `GET /logs/taxonomy[?mode=]`, `GET /logs/days?from&to`,
  `GET|PUT|DELETE /logs/days/{date}` (partial PUT, null clears). Two-way sync with legacy `/health-logs` (byte-identical
  goldens) so cycle/fertility/messages/export engines keep working.
- Labels `log-taxonomy` namespace (seed + frontend copies; registration in message-scopes/bundled is B-N3-03).
- Contract `ROUTES=all` 1150 passed; schema-diff OK. QUESTIONS #80.

## B-N3-02 — Log preferences — quick tiles, order, custom items

- **Migration** `00021_health_log_preferences` (+ Laravel twin): `health_log_preferences` (per user+mode, JSON
  `category_order` / `hidden` / `pinned`, NULL = default) and `health_log_custom_items` (label ≤40, soft delete; item
  code `custom_<id>` in `health_log_entries`).
- **Endpoints:** `GET|PUT|DELETE /api/v1/logs/preferences[?mode=]` (≤8 pinned tile keys = category or
  `category.param`; defaults per mode and today's phase in `taxonomy/prefs.go`), `GET|POST /logs/custom-items`,
  `PATCH|DELETE /logs/custom-items/{id}` (≤20 active, unique label per category, 404 on others' ids). `PUT /logs/days`
  validation now accepts only the user's custom items on host params.
- Contract `logs` 34 passed; schema-diff OK. QUESTIONS #81.

## B-N3-07 — Analysis engine endpoints

- `internal/analysis` (pure computation + thin handlers): `GET /api/v1/analysis/{summary,cycle,period,symptoms,
  correlations,body}` and `/analysis/monthly/:ym[?calendar=]`, `range` 7d|2w|1m|3m|6m(default)|1y|all. Sections are
  `{plus,locked,ready,data}`; Plus (`plus.deep_analysis`): mood_by_phase, sleep_mood, correlations, labs (labs empty
  until B-N6-06); monthly `pdf` gated by `plus.pdf_share`.
- FIGO 2018 constants (cycle 24–38, period ≤8, variation 9/7/9 by age, not applied outside 18–45), min-data rules
  (patterns ≥3 cycles, trends ≥2 points), top finding phrases (namespace `analysis`), correlations phi / Cramér's V with
  Cohen strengths, ≥20 paired days, `not_causal: true`; 7-day moving-average weight.
- 41 engine goldens (fixed clock), 24 contract goldens (`ROUTES=all` 1200 passed), D-39 proposed. QUESTIONS #82.

## B-N2-11 — N2 rollout — stage deploy and e2e

- verify green on HEAD (worktree); `stage` pushed and deployed (goose 14–19 applied; frontend rev 46fdb93). Smoke:
  39 checks — 35 pass, 4 warn, 0 fail; no 5xx/console errors; N1 bugs B-1/B-2/B-3 confirmed fixed. Report
  `docs/qa/bloom/n2-stage.md`, shots `docs/qa/bloom/n2-stage/`.
- New reusable `bloom/bin/stage-smoke.mjs` (gate cookie + OTP read at runtime over ssh, light/dark shots with PII
  masking, per-page 4xx/5xx/console summary; `--mobile`, `--new-user`, `--admin`).
- Bugs: B-1 (medium, male lands on women home) → B-N4-05 scope; B-2..B-5 (low) → new task B-N2-11b.
- Stage test data: 5 users + 4 half-onboarded, 4 fake invoices, code `QAN2SMOKE` (deactivated).

## B-N3-08 — Analysis hub and tab routing

- `entities/analysis` (summary schema tolerant per section, `HUB_RANGES` 3m/6m/1y, `useAnalysisSummary`, layout
  helpers, `analysisKeys.report|monthly` for B-N3-09..12; schema tests run against the Go engine goldens).
- `screens/analysis` hub at `/analysis`: range tabs, category chips, top finding, 8 cards with `PlusBadge` /
  `PlusGate` blurred locks, skeleton / error / `no_data` / not-enough-data per card, accessible charts. Mode choice:
  cycle/teen/menopause hub; ttc/postpartum → cycle hub for now; pregnancy → placeholder. Stub detail routes
  (`AnalysisDetailStub`) for cycle/period/symptoms/correlations/body/labs/monthly — B-N3-09/10 swap the bodies.
- Entries: home predictions «جزئیات» + PMS card → `/analysis`, calendar segment «تحلیل», Me row «تحلیل و گزارش‌ها».
  `NAV_READY.analysis` still false (flip in B-N3-09 when `/analysis/symptoms` is real).
- Screenshots `docs/qa/bloom/B-N3-08/`. QUESTIONS #83. `dev-up.sh web` now kills only :3000 (was killing admin-web).

## B-N3-03 — Log sheet v2 — quick tiles, accordion, detail panels, body map

- `entities/health-log` v2 API (`useLogTaxonomy`, `useLogPreferences`, `useLogDay(s)`, `saveLogDay`); new
  `features/log-day` (pure draft/diff/search model + tests, optimistic save invalidating logs/health-log/cycle/fertility/
  message caches, generic `ParamField` for all 9 param types, date strip, quick tiles, accordion, search, summary
  footer, voice tab with `plus.voice_log` lock, detail panels bleeding / pain / measurements); `widgets/body-map`
  (token-only SVG, each region an `aria-pressed` button). `/log?date=` and the `log` sheet (now full height) render v2;
  home «ثبت امروز» rows and calendar «ثبت جزئیات» open it; companion keeps the old sheet. Old `CategorySheet`/`FieldRow`
  removed. Namespaces `logSheet` (+ `logTaxonomy` bundled) registered; `plus` added to the shell.
- `bloom/bin/shot.mjs` gained `--click 'sel||text=..'` and `--name`.
- Screenshots `docs/qa/bloom/B-N3-03/`. QUESTIONS #84. Customize gear waits for B-N3-04.

## B-N2-11b — N2 stage smoke follow-ups

- B-2: checkout counts a code as applied only when `discount_source == 'code'` (`plus-checkout/lib/code-state.ts` +
  test); when the trial offer wins it shows «پیشنهاد ویژه‌ات تخفیف بیشتری دارد…» and doesn't send the code.
- B-3: teen/menopause get no fertility copy in `cycle_view.daily_card` — engine profile joins `user_life_profiles.life_mode`,
  `Profile.NoFertilityCopy`, cache key extended only for those modes; unit + int tests; contract goldens unchanged.
- B-4: `LifeMode.TracksCycle()` (false for menopause) → checkups plan by interval, `TimingLabel` «یک روز ثابت از ماه»,
  detail cycle-day hint null for menopause.
- B-5: `shared/config/app-version.ts` (`NEXT_PUBLIC_APP_VERSION`) in Me hub + About.
- Screenshots `docs/qa/bloom/B-N2-11b/`. QUESTIONS #85.

## B-N3-04 — Log customization screen

- `/log/customize` (`screens/log-customize` + `features/customize-log`): drag or ↑↓/Home/End reorder with focus kept
  and live announcement, pin ≤8 (counter, amber at cap, reason when full), show/hide, custom items add / inline rename /
  delete with 422 → copy mapping, reset (DELETE), single PUT of changed lists, optimistic + rollback, unsaved-changes
  sheet on back. Gear in `features/log-day` now links here. `entities/health-log/api/log-prefs.ts` mutations;
  `apiClient.patch` added. Namespace `logCustomize`.
- Screenshots `docs/qa/bloom/B-N3-04/`. Minor: a pinned+hidden tile isn't returned by GET, so it shows unpinned until
  reload after unhiding. Add `/log/customize` to `docs/night-bloom/routes.md`.

## B-N3-09 — Analysis details — cycle, period, symptoms, correlations, body

- `widgets/charts` (pure SVG, tokens, `role=img` + `aria-label` + sr-only tables; LTR time axes): ColumnChart,
  PhaseBar, CycleDots, HeatGrid, SeriesBars, StripRows, TrendLine, blocks, ReportFrame, geometry + tests.
- Screens `analysis-{cycle,period,symptoms,correlations,body}` replacing the B-N3-08 stubs, footnotes (FIGO, not-causal,
  moving average), range tabs per artboard, Plus lock on correlations (teen: no upsell), all states.
  `entities/analysis` report schemas/hooks tested against Go goldens. `NAV_READY.analysis = true` (menopause «علائم» →
  `/analysis/symptoms`). Scope `analysisReport`.
- Backend: moods excluded from symptom patterns (`isMoodKey`), one engine golden updated.
- Screenshots `docs/qa/bloom/B-N3-09/`. QUESTIONS #86.

## B-N3-05 — Voice logging (Plus) — record, transcribe, parse, review

- `internal/ai` (ports `Transcriber`, `LogParser`; `ai.Client` tags calls with `Feature` and reports `Usage` to a
  `Recorder` — slog only, no payload/user; providers `fake` (deterministic fixtures, lexicon NLU incl. custom items) and
  `gemini` (REST, key header, no error bodies); `AI_PROVIDER` fake outside prod, `none` in prod, fake refused in prod).
  B-N6-05 extends it (chat, vision, consent, DB usage/cost).
- `POST /api/v1/logs/voice` (`internal/voicelog`): Plus gate before reading, throttle 6/min 60/h, streamed multipart
  ≤2048 KB, sniffed audio types, no temp files, audio + raw body zeroed right after transcription (tested), suggestions
  validated with `taxonomy.Parse` (≥0.5 confidence, ≤20), never auto-saved; quota consumed after success.
  `PUT /logs/days` accepts `voice_params` → rows tagged `source=voice`. D-44 proposed.
- Frontend `features/voice-log` (MediaRecorder + permission/unsupported/mic errors, timer, upload, «این‌ها را فهمیدیم»
  chips opening the manual section with merged values) slotted into the log sheet's voice tab; `apiClient` FormData +
  `timeoutMs`. Env `AI_*`, `GEMINI_*` in `.env.stage.example` + `docker-compose.stage.yml`. `shot.mjs --fake-media`.
- Screenshots `docs/qa/bloom/B-N3-05/`. QUESTIONS #87.

## B-N3-10 — Monthly report and labs trend screen

- `/analysis/monthly/[ym]?calendar=` (`screens/analysis-monthly`): Jalali/Gregorian month title + stepper (no future,
  ≤24 back, calendar switch via `convertParts`), headline, metrics table with coloured deltas, top symptoms, next-month
  suggestion, running-month subtitle, empty state; «ساخت PDF برای پزشک» = Plus lock or «به‌زودی» (B-N6-04).
  `/analysis/labs` (`screens/analysis-labs`) empty state until B-N6-06 with Plus note. `entities/analysis` monthly
  schema tested against 8 Go goldens.
- Screenshots `docs/qa/bloom/B-N3-10/`. QUESTIONS #88.

## Run stop — stage deploy of db481a2 (2026-10-02)

- Per user: finished in-flight tasks only, then stopped. `stage` = db481a2 (bloom faba825 + canvas CB-NAV-02 / CB-MENO-12)
  deployed with `./deploy-stage.sh` (all checks ok; goose at 24; frontend rev db481a2; backend-go image unchanged from
  the 2026-10-01 build of the same tree — `/logs/voice`, `/analysis/*`, `/search` routes answer). No e2e smoke run for
  the N3 work (B-N3-14 release task still todo). Next runnable bloom tasks: B-N3-06, B-N3-11, B-N3-12.

## B-N3-06 — Pregnancy and postpartum log sheets on the new taxonomy

- `features/log-day` mode presets (`model/mode-presets.ts` + tests, `ui/presets/ModePreset.tsx`, kick count sub-label):
  pregnancy sheet (week/trimester heading, 8 tiles, bleeding alert row, inline symptom groups, measurements, meds; kicks →
  `/pregnancy/log?tab=movement` with today's count; contractions «به‌زودی») and postpartum sheet (mother: lochia-first
  bleeding panel, pain & stitches with body map, breasts, mood; baby feed/sleep/diapers «به‌زودی» until B-N5-07).
- `/pregnancy/log` default = v2 sheet page; `?tab=day|focus` = old pregnancy day log (still the only path that raises
  pregnancy alerts — linked from the sheet; `pregnancy-alerts` `log_symptoms` → `?tab=day`). Postpartum uses `?sheet=log`.
- Screenshots `docs/qa/bloom/B-N3-06/`. QUESTIONS #90. Test user 09900000063 (postpartum) on ritme_dev.

## B-N4-01 — Companion & family schema

- **Migration** `00026_companions` (+ Laravel `2026_10_02_000026_…`; 00025 intentionally unused): `companions`
  (owner, companion user, type partner|spouse, status invited|active|revoked, revoked_by), `companion_invites` (HMAC
  code hash, optional bound phone, attempts lock at 5, 24 h expiry), `companion_grants` (cycle|symptoms|meds|
  appointments|pregnancy × view|edit; no row = none), `families` + `family_children` (child FK added in B-N5-02),
  `companion_audit_logs` (no health payload). Codes are varchar validated in Go (extensible for teen/IVF/loss).
- `internal/companion` service: CreateInvite / RenewInvite / Accept / Revoke / SetGrants / SetSharedChildren / List* /
  Level / CanRead / CanWrite / Audit — unit + int tests (expiry, one-time + concurrent accept, grant matrix, revoke).
  No HTTP yet (B-N4-02). D-47 proposed. QUESTIONS #91.

## B-N3-12 — Pregnancy analysis hub and weight-gain screen

- `GET /api/v1/analysis/pregnancy` (409 `pregnancy_not_active` outside pregnancy): sections weight_gain (IOM 2009
  singleton bands by pre-pregnancy BMI, weekly piecewise band, estimated baseline with source), blood_pressure
  (≥140/90 high, ≥160/110 severe — ACOG PB 222), glucose (Plus; ADA 2024 targets), kicks (10 in 120 min from week 28),
  symptoms by trimester (Plus), visits. Thresholds with sources in `internal/analysis/pregnancy_thresholds.go`, echoed
  in the payload. Engine + contract goldens; D-46 proposed.
- Frontend `screens/analysis-pregnancy` (hub + `/analysis/pregnancy-weight`), wired into `/analysis` for pregnancy
  mode; namespace `analysis-pregnancy`. Screenshots `docs/qa/bloom/B-N3-12/` (QA user 09900000121). QUESTIONS #92.

## B-N3-11 — TTC analysis hub and BBT/ovulation confirmation

- `GET /api/v1/analysis/ttc` (trying cycles/months, referral by age ASRM 2020 / ACOG 781, free bbt/lh/regularity,
  Plus timing/mucus/luteal) and `GET /api/v1/analysis/fertility[?cycle=]` (chart, coverline, three-over-six highs via
  the existing `internal/fertility/bbt`, window, ovulation source, LH, intercourse, luteal). LH/mucus merged from
  `fertility_logs` + `health_log_entries`. 11 engine + 17 contract goldens; D-45 proposed.
- Frontend `screens/analysis-ttc` (TTC hub via `AnalysisPage` `ttcHub` slot, `/analysis/fertility` with BBT chart).
  Screenshots `docs/qa/bloom/B-N3-11/`. persona 12 got 71 days of TTC logs on ritme_dev. QUESTIONS #93.

## B-N3-13 — N3 design-fidelity audit and fixes

- `docs/night-bloom/audit-n3.md`: 0 high, 2 med (fixed: log sheet header gear → `/log/customize` on every mode;
  body-map pin positions/labels), 6 low open (pregnancy sheet CTA copy, panel gear, trimester ring, analysis share
  buttons, phase-bar legend dots, IOM row height). Gear sits inside the sheet `h2` until `AppSheet` gets an `action` slot.
- Screenshots `docs/qa/bloom/B-N3-13/` (personas p04/p06/p12/p121/p063/p081/p082, side-by-side for the two meds).

## B-N4-02 — Invite, accept, revoke and access-filtered companion APIs

- Owner routes `GET/POST /api/v1/companions`, `GET/DELETE /companions/{id}`, `POST /companions/{id}/renew`,
  `PUT /companions/{id}/grants`, `PUT /companions/{id}/children` (422 until B-N5-02), `GET /companions/audit`; companion
  routes `POST /companions/accept`, `GET /companions/links`, `DELETE /companions/links/{id}`,
  `GET /companions/links/{id}/sections/{cycle|symptoms|meds|appointments|pregnancy}` (minimised views via
  `companion/shared`). `for_user_id` on care medication/appointment show/create/update (view/edit; prep stays owner-only).
- Security: uniform 404/422, accept throttle 10/h user + 30/h IP, invite SMS gates (2 min resend gap, 3/24 h per
  recipient hash, 5/24 h per owner, 20/h create+renew), audit written before every delegated read/write (fail closed),
  owner inbox notices (logged on failure), prod 503 without `COMPANION_CODE_PEPPER`, fake SMS refused in prod.
- `internal/sms` adapter (none/fake/gateway=Kavenegar `KAVENEGAR_TEMPLATE_COMPANION_INVITE`). Env in
  `.env.stage.example` + compose. Contract `companion` 16; D-48 proposed. QUESTIONS #94.

## B-N4-03 — Male account path and companion home aggregate

- `GET /api/v1/companion/home` (male companion only; 403 otherwise): per active link partner name + granted section
  views via `companion/shared` (null without grant, audited before build), phase (pregnancy → cycle phase → general),
  tips per phase from `message_contents` `companion_tip` (registered in the admin registry; embedded fa/en fallback),
  up to 4 articles; empty state with `enter_code` when no links.
- `enums.ResolveAccountMode` → `companion` for gender=male; `PUT /profile/life-stage` mode 422; `/home*`,
  `/messages/daily|mode` → 409 `companion_account`. Contract `companion-home` 7; existing goldens unchanged. D-49.

## B-N4-04 — Owner flow — companions list, type, access, shared children, invite, done

- `entities/companion` (schemas, hooks + key factory, create/grants/renew/revoke mutations, grant helpers,
  CompanionCard/FamilyStrip/PersonBubble), `features/invite-companion` (wizard Type → Access (default none) → Children
  (spouse; «به‌زودی» until B-N5-02) → Invite by phone/SMS or code-only with copy/Web Share → Done; phone normalisation
  mirrors Go), screens `companion-list|invite|detail` at `/companions`, `/companions/new`, `/companions/[id]` (grants
  editor, renew shows a new code once, last 5 audit entries, revoke confirm). Codes live only in component state.
- Me hub row «همدم‌ها و خانواده» (hidden in companion mode) and privacy «ریتمی همراه» rows per active companion.
- Screenshots `docs/qa/bloom/B-N4-04/`. persona 04 has an active spouse (persona 12) on ritme_dev.

## B-N4-07 — Admin — companion tips content and link overview

- Admin API `/api/admin/v1/companions/tips[/:phase]` (GET/PUT/DELETE reset per locale; fixed `companion_tip` slots
  note + ≤3 tips per phase, fallback row → default locale → embedded copy, `source` per slot; audit) and
  `/companions/links` (read-only, masked names/phones, counts by status/type, `grants_count` only — no codes, no
  section names). Any active admin. Docs `admin-api.md` §16; int tests. No migration, no deviation.
- admin-web «همدم» group: «نکته‌های همدم» (phase/locale tabs, reorder, preview card) and «اتصال‌ها» (counts, filters).
- Screenshots `docs/qa/bloom/B-N4-07/`.

## B-N4-05 — Male onboarding and companion panel home

- Onboarding partner step = real 6-char code entry (`CompanionCodeField`, paste/IME, Persian digits) → accept →
  `/onboarding/partner/linked` → `/companion`; `landingRoute(…, gender)` sends males to `/companion` (stage bug B-1).
- `screens/companion-home`: `/companion` (partner card day/phase/next period or pregnancy week, shared sections with
  access level, tips, articles, child placeholder, partner chips, empty state with code entry; women redirected to
  `/home`) and `/companion/links` (enter code, links, leave). Companion nav امروز · خدمات · من (no FAB/mode tab) via
  `use-nav-mode` (`companion` or 409 `companion_account`); `HomePage` redirects males. Me hub for males hides Plus
  upsell, life-stage and cycle settings; adds «کد همدم». `entities/companion` companion-side hooks + code helpers;
  `LifeStage.companion` flag. Namespace `companionHome`.
- Screenshots `docs/qa/bloom/B-N4-05/`. Test users 09900000551 (linked to p04) / 552 (no link). QUESTIONS #96.

## B-N3-14 — N3 rollout — stage deploy and e2e

- verify green on HEAD (worktree); `stage` 201eafa pushed + deployed (goose 26). Smoke: 41 checks — 35 pass, 6 warn,
  0 fail; no 5xx / console errors. Report `docs/qa/bloom/n3-stage.md`, shots `docs/qa/bloom/n3-stage/`.
- Bugs: B-1 (high, past-day bleeding splits periods), B-2/B-3 (medium), B-5..B-9 (low) → new task B-N3-14b;
  B-4 (postpartum home shows cycle home) → B-N5-04 scope.

## B-N4-06 — «ثبت برای چه کسی؟» in medication and appointment forms

- `entities/companion` record-for logic (`recordTargets`, `showRecordForPicker`, `canRecordFor`, `useRecordFor`,
  `RecordForSheet` / `RecordForRow` / `RecordedFor`) + tests; manage-medication/appointment mutations take
  `forUserId` (prep and care_item_key stripped for owner records), `useMedicationFor` / `useAppointmentFor` show via
  `?for_user_id=`. Forms: picker only with edit on that section, `?for=` preselect, delegated edit hides delete/prep,
  view-only message, 403 → back to self + refetch links, «برای {name} ثبت شد» panel; companion home «افزودن … برای
  {name}» entries. Scope `reminderForm`.
- Screenshots `docs/qa/bloom/B-N4-06/`. Appointment detail page doesn't support delegation (success panel instead).

## B-N5-01 — Postpartum mode backend — activation, recovery log, EPDS

- **Migration** `00030_postpartum` (+ Laravel twin; must be applied after canvas 00027–00029): `postpartum_profiles`
  (one per user) and `epds_checks` (unique user+kind+day).
- `GET /api/v1/postpartum` (status: days/weeks since birth, phase, 6-week countdown, check-in due, today's recovery,
  alerts, week tip, call-when), `POST /postpartum/activate` (closes an active pregnancy as delivered; life mode
  postpartum), `GET|PUT /postpartum/recovery` (view over taxonomy v2 rows; taxonomy gained lochia none/spotting,
  `sleep.hours`, `baby.feeds_count`), `GET /postpartum/epds/questions`, `POST|GET /postpartum/epds` (Q10>0 or full ≥13 →
  urgent safety with 115/123/1480; answers never returned or logged). `/messages/daily` for postpartum now gives the
  week tip with alert overrides (admin groups registered). 32 contract goldens; D-54 proposed. QUESTIONS #97.

## B-N4-08 — Companion security review

- `docs/security/bloom-companion.md`: threat model + findings. Earlier M-1/M-2 fixes hold; no IDOR/escalation found.
  Release blockers CMP-H1 (pregnancy/postpartum leak via cycle/symptoms views) and CMP-M1 (teen owner with adult
  partner; fertile phase sent) plus CMP-M2/M3/L1/L5/L6 → follow-up **B-N4-08b** (runs after canvas CB-TEEN-01/LOSS-01
  land in internal/companion). B-N4-10 now depends on B-N4-08b.

## B-N4-09 — N4 design-fidelity audit and fixes

- `docs/night-bloom/audit-n4.md`: 0 high, 1 med (companion home phase/access pills → taller outlined pills), 7 low
  (1 fixed: male avatar icon on the linked screen; W5, P2, M2, R2 + 2 open). Compared against the B-N4-04..07
  screenshots (UI unchanged since) because the dev API was held down during the migration-order freeze; after-fix
  shots pending.

## B-N3-14b — N3 stage smoke follow-ups

- B-1: `checkAndUpdatePeriodStart` rewritten — bleeding inside/≤2 days after a period extends it (end never shrinks),
  ≤2 days before the next start moves that start back, true new periods get non-negative `cycle_length`, back-dated
  rows between periods become closed 1-day periods, LMP moves only for the newest row; repro + 3 scenario int tests;
  Laravel side-effect golden rewritten via documented `applyD55` (D-55 proposed).
- B-2: `PromoteLoggedStart` — the newest unconfirmed `user_logged` row counts as confirmed in the engine (cache schema 3).
- B-3: fertility reads LH / mucus merged from `fertility_logs` + `health_log_entries` (strongest LH, most fertile
  mucus); BBT via existing write-back; tests. Chance of pregnancy is engine-based (unchanged).
- Lows: menopause `keep_logging` finding (no «log 3 cycles»), pregnancy weight «until date», fertility subtitle bidi
  isolation, Persian digits in fa voice labels, teen voice tab hidden; TTC int test expectation fixed. QUESTIONS #99.

## B-N5-04 — Postpartum home, recovery and mood check screens

- `entities/postpartum` (schemas tested against contract goldens; safety payload never dropped; EPDS submit kept out of
  the mutation cache), screens `postpartum` (`/postpartum` home: hero with completed weeks + day X/42 ring, alerts,
  mood chips saved to taxonomy rows, bleeding/feeds/sleep tiles → postpartum log sheet, upcoming visits incl. 6-week
  suggestion and vaccines placeholder, «کی فوراً تماس بگیرم؟», week tip; `/postpartum/setup` activation),
  `postpartum-recovery` (partial PUT, bleeding alert card), `postpartum-mood` (EPDS flow from API; result or safety
  screen with call buttons first, local safety fallback with 115/123/1480 on request failure).
- Home redirects postpartum users to `/postpartum` (N3 stage bug B-4); nav «امروز» → `/postpartum`
  (`NAV_READY.children` false until B-N5-05); pregnancy «زایمان کردم» card from week 20.
- Screenshots pending (dev API held during the migration-order freeze). QUESTIONS #100.

## B-N4-08b — Companion security fixes

- CMP-H1: `shared.Reader.ReadFor` mode-aware (pregnancy/postpartum without pregnancy grant and menopause → neutral cycle
  view; symptoms limited to cycle-mode options unless pregnancy granted; days_late/cycle_day null past 14 d).
- CMP-M1: fertile → follicular for NoFertilityCopy owners; teen-mode owner's partner/spouse links grant nothing while
  she is teen (`teen_guard.go`, no revoke; `parent` links unaffected).
- CMP-M2: delegated GET/PUT only for active meds / upcoming non-private appointments (404 otherwise), 120/h throttle.
- CMP-M3: global failed-accept circuit breaker (Redis; 30/min or 200/h → 15 min pause, fail closed).
- CMP-L1: read audits coalesced per 15 min, audit cursor + action filter, home/section throttle 300/h. CMP-L5 notice
  name sanitising. CMP-L6 admin links overview super-only, no user ids, 2-digit mask.
- `docs/security/bloom-companion.md` statuses updated. QUESTIONS #101.

## B-N5-02 — Children — profiles, WHO growth percentiles, vaccines, milestones

- **Migration** `00031_children` (+ Laravel twin): children, measurements, vaccine doses, milestone checks; FK
  `family_children_child_id_foreign`; 132 catalog seed rows (`child_vaccines` 21, `child_milestones` 57, activities,
  notes, age notes, learn — all `needs_review`). WHO LMS tables (0–1856 days, weight/length/head, boys/girls) embedded
  in `backend-go/seeds/who`, LMS maths in `internal/children/growth` (restricted method beyond ±3 SD), golden-tested
  against WHO P3/P50/P97 + SD columns.
- `/api/v1/children/*`: list/create (≤10)/show(home)/update/delete, measurements CRUD with percentile/z/in_band, growth
  reference curves, vaccines (mark visit / dose), milestones by month, learn, `/children/reminders` (3 days before →
  30 after). Spouse access via family_children, read-only, audited before build. Companion invite/children endpoints
  accept the owner's own children; companion home `child` card filled. 18 contract goldens; D-56 proposed.
- QUESTIONS #102.

## B-N6-05 — AI adapter platform — providers, consent, usage and cost

- **Migration** `00032_ai_platform` (renumbered from 00033 so the sequence has no gap; + Laravel twin):
  `user_consents.version`, `ai_usage_logs` (no content; user SET NULL on delete).
- `internal/ai`: `Chatter` (streaming over a channel, SSE-ready) and `Extractor` (image/PDF → schema fields with
  confidence) ports + fake fixtures + Gemini (httptest only); PII filter (email, Iranian mobiles, national id, long
  digit runs, the user's name) on prompts and outputs; pricing table; `internal/ai/usage` (DB usage log, fail-closed
  daily budget, admin aggregates `GET /api/admin/v1/ai/usage`); `internal/ai/access` gate (versioned consent + Plus
  key per feature → 403 `consent_required`, 503 `ai_budget_exhausted`).
- `internal/consent`: versioned consent catalog + `GET /api/v1/consents`, `GET|PUT /consents/{code}` (409 stale);
  `/profile/consents` unchanged in shape. Voice logging goes through the gate. Env `AI_DAILY_COST_CAP_USD`,
  `AI_PRICES`. 15 contract goldens; D-58 proposed. QUESTIONS #103. Security review pending.

## B-N5-03 — Baby logs and pregnancy tools backend

- **Migration** `00033_baby_logs` (+ Laravel `2026_10_04_000033_…`): `baby_feeds` / `baby_sleeps` / `baby_diapers`
  (per child), `pregnancy_kick_sessions`, `pregnancy_contraction_sessions` + `pregnancy_contractions`; one running
  session = `active_lock` + unique index; seeds `pregnancy_alert / contractions_511` (fa, en).
- `internal/babylog` + `routes_babylog.go`: `/api/v1/children/{id}/feeds` (GET day + active + last + next_side, POST
  manual, `/start`, `/{fid}/side`, `/{fid}/stop`, PUT/DELETE), same for `/sleeps`, `/diapers` CRUD,
  `/baby-logs?days=` (today card, days, averages, L/R %). Child access via `children.Service` (spouse read-only).
  Child home `today` filled (`children.Handlers.SetToday`). Feeds own the mother's `baby.feeds_count` (postpartum only).
- `internal/pregnancy/tools` + `routes_pregnancy_tools.go`: `/api/v1/pregnancy/kick-sessions` (start, `/{id}/kicks`
  ±1, `/{id}/stop` → day total into `pregnancy_fetal_movements`, delete) and `/api/v1/pregnancy/contractions`
  (`/start`, `/stop` → `{session, alerts}`, `/sessions/{id}` get/finish/delete). Maths `internal/pregnancy/labor`;
  5-1-1 = alert engine rule `contractions_511` (admin-editable, 9 rules now).
- 21 Go-recorded goldens (`babylog`, `pregnancy-tools`) + `children/child_home` re-recorded; D-60 proposed.
  QUESTIONS #104. Frontend B-N5-07/08: flip the «به‌زودی» tiles (feeding/diapers/baby sleep/contractions).

## B-N5-05 — Children list, add child and child home

- `entities/child` (schemas tested on contract goldens, hooks, mutations, local-only photos in IndexedDB — never
  uploaded, wiped on logout/delete; `ChildAvatar`, `ChildStatusChips`, `ChildrenStrip`), screens `children`
  (`/children`, shared child «فقط مشاهده»), `child-add` (`/children/new`, `/children/[id]/edit`, prefilled from the
  postpartum profile, 422 mapping, delete confirm), `child-home` (`/children/[id]`: age hero, measurements with
  percentile chips, next vaccine, tiles, «این هفته {name}», today row «به‌زودی»; stubs for growth/vaccines/milestones/
  learn until B-N5-06). Nav «کودک» tab live (`NAV_READY.children`, `childIds` prop); postpartum hero children row +
  real vaccine rows; companion home shared-children card. Namespace `children`.
- Screenshots `docs/qa/bloom/B-N5-05/`. Test users 09900005501/02/03 on ritme_dev.

## B-N4-10 — N4 rollout — stage deploy and two-account e2e

- verify green on HEAD (worktree); `stage` pushed and deployed (aa11509; goose 27–32 in order, version 32; new route
  groups answer 401 unauthenticated). Two-account e2e: 24 checks — 22 pass, 1 fail (B-1: no UI to share children),
  1 API-only; 0 5xx / console errors. CMP-H1 neutral view, grant change within 1 s, revoke, uniform 422 codes verified.
- Report `docs/qa/bloom/n4-stage.md`, shots `docs/qa/bloom/n4-stage/`. Follow-ups → **B-N4-10b**. QUESTIONS #104b.

## B-N5-08 — Kick counter and contraction timer

- `features/pregnancy-tools` (queries/mutations under `pregnancyKeys`, server-skew-corrected timing, queued taps/undo,
  haptics, 17 timing tests), screens `log-kick` (`/pregnancy/kicks`: tap target, ring to 10, elapsed, 2-hour guidance,
  low-count card with call CTA, history, today total) and `log-contraction` (`/pregnancy/contractions`: waiting /
  between / contracting states, 60-min averages from the server, start·duration·interval table, admin-parameterised
  guidance, 5-1-1 alert card with call + ack, persistent «الگوی تماس» after reload). Sessions restored from the server.
  Log sheet tiles now link here. Namespace `pregnancyTools`.
- Screenshots `docs/qa/bloom/B-N5-08/` (test user 09900000058). QUESTIONS #105.

## B-N4-10b — N4 stage smoke follow-ups

- B-1: spouse wizard children picker (`ChildrenPicker`, `child_ids` on create) + «فرزند مشترک» editor on
  `/companions/[id]` (`useUpdateCompanionChildren` → `PUT /companions/{id}/children`); family strip shows children.
- B-2 male Ready copy; B-3 pregnancy mode saved only at the end of setup (`finishPregnancySetup`); B-4 Persian list
  join without «، و»; B-6 resume cookie cleared on onboarding complete; B-7 admin «پارتنر»; I-1 invite hours rounding.
- Screenshots `docs/qa/bloom/B-N4-10b/`.

## B-N5-07 — Feeding timer, baby sleep/diapers and postpartum analysis hub

- `features/baby-log` (schemas/queries, live timer from the server's active feed, L/R switching, bottle/pump ml,
  manual entry, sleep start/stop/manual across midnight, one-tap diapers, today list), `screens/log-feed`
  (`/children/[id]/feeding`, resolver `/children/feeding?section=` for log sheet tiles, child picker, spouse read-only),
  `screens/analysis-postpartum` (EPDS bands trend, lochia strip, feeds L/R, Plus mother/baby sleep, percentile chips,
  weight since birth) via the `postpartumHub` slot. Child home «امروز» rows show real values. Namespaces `babyLog`,
  `analysisPostpartum`.
- Screenshots `docs/qa/bloom/B-N5-07/` (QA user 09900000171). QUESTIONS #106.

## B-N6-05b — AI platform security fixes

- H1/H2: `Client.Chat` returns `*ai.ChatStream{Events, Close(), Wait()}` with its own ≤120 s context; usage always
  recorded (cumulative Gemini usage or conservative estimate, `Usage.Estimated`), Plus quota reserved first;
  `ProviderHTTPClient` with dial/TLS/header/idle timeouts. M1: thoughts/tool-use tokens counted, thinkingBudget 0 +
  maxOutputTokens for parse/extract/transcribe. M2: `AI_USER_DAILY_COST_CAP_USD` (429 `ai_user_budget_exhausted`),
  per-feature throttles in `Guard.Chain`. M3: throttle → Plus → consent → reserve (refund on failure), per-user
  semaphore (429 `ai_busy`); voice wired. M4: normalised PII matching, separator-tolerant digit runs, Latin↔Persian
  names, identity keys rejected in extraction schemas (lab fixture key `name` → `marker`). M5: `/profile/consents`
  version-aware, AI grants need `versions.<code>`, export carries versions + `ai_usage`. L2/L3/L4/L5 done (usage
  `user_id` anonymised after 90 days on write). Privacy screen sends versions. QUESTIONS #107.

## B-N5-06 — Growth, vaccines, milestones and child learn screens

- Screens `child-growth` (indicator tabs, P3–P97 band + median + child points via `widgets/charts`, history,
  add/edit/delete sheet with 422 mapping), `child-vaccines` (ring + next visit, برنامه / کارت / یادداشت tabs, mark dose
  or visit given, «ثبت نوبت» → appointment form via handoff), `child-milestones` (month chips, progress ring,
  non-judgemental copy, play ideas, doctor note), `child-learn` (topic chips, weekly pick, tip/article sheets);
  spouse read-only. Parsers tested on Go goldens. Replaces the B-N5-05 stubs.
- Screenshots `docs/qa/bloom/B-N5-06/` (user 09900005501, child 1).

## B-N5-09 — Admin — vaccine schedule, milestones, WHO tables, postpartum content

- `GET /api/admin/v1/children/who` (read-only WHO LMS viewer with derived P3–P97, month/week/day sampling); child
  catalog groups and postpartum message groups are super-admin-only for writes (`catalog.Admin.WithSuperGroups`,
  `registry.SuperOnlyGroups`; `GET /messages` exposes `super_only_groups`); 5-1-1 rule gains optional `contact_phone`
  (alert `contact {text, phone}`). Docs `admin-api.md` §18 + §13. Int tests.
- admin-web «کودک و پس از زایمان»: vaccines (grouped by visit), milestones by month (4 sections), learn, WHO table,
  shared item form with per-group meta + «بازبینی شد», postpartum content overview → messages editor (read-only for
  editors); alert-rules editor shows `contact_phone`. `FormPage` `readOnly` prop. Screenshots `docs/qa/bloom/B-N5-09/`.
- Pre-existing red: `internal/catalog/api_int_test.go` (3 tests) because canvas migrations seed catalog rows — canvas-owned.

## B-N6-06 — Lab analysis backend

- **Migration** `00034_labs` (+ Laravel twin): lab_reports, lab_files, lab_markers, lab_jobs; 26 `lab_markers` catalog
  rows (needs_review; super-admin writes).
- `/api/v1/labs/*` (17 ops): multipart upload (1–5 files, sniffed, images re-encoded to WebP, AES-256-GCM at rest with
  `LAB_FILE_KEY` / `LAB_FILE_KEY_PREVIOUS`, private 0600 storage, deleted with lab/page/account + orphan sweep),
  manual lab, status poll, verify (202), markers CRUD, trends, catalog, feedback, owner-only download.
  Gate: throttle → `plus.lab_ai` → consent `ai_lab_analysis` → cost/busy → reserve; 10 uploads/day. In-process
  DB-backed worker (`lab_jobs`, lease, 3 tries). Interpretation via `Client.Chat` (non-streaming, redacted, ≤3 per lab)
  with rule-based red flags/doctor questions and rules fallback. Analysis summary `labs` section filled.
- 23 contract goldens (fake provider), D-61 proposed. QUESTIONS #108. Security review pending.
