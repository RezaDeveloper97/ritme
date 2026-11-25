import fsd from '@feature-sliced/steiger-plugin';
import { defineConfig } from 'steiger';

// Feature-Sliced Design boundary linter. See CLAUDE.md §3 for the rules this
// enforces (downward-only imports, no sibling cross-imports, public API).
export default defineConfig([
  ...fsd.configs.recommended,
  {
    // Tests live next to the code they cover; they are not slices.
    ignores: ['**/*.test.ts', '**/*.test.tsx'],
  },
  {
    // The route-composition layer is named `screens` (not `pages`) so the
    // Next.js build doesn't mistake it for the legacy Pages Router. steiger
    // only recognizes `pages` as a layer name, so references coming FROM
    // screens are invisible to it — these slices are all consumed by screens.
    files: [
      './src/entities/user/**',
      './src/features/auth/**',
      './src/widgets/bottom-nav/**',
      './src/widgets/smart-tip/**',
      './src/widgets/bbt-chart/**',
      './src/widgets/cycle-calendar/**', // B-N1-07: consumed by screens/calendar
      './src/widgets/pregnancy-week-carousel/**',
      './src/widgets/pregnancy-care-checklist/**',
      './src/widgets/pregnancy-dating/**', // sign-up pregnancy form: screens/onboarding-flow + screens/pregnancy-onboarding
      './src/features/skip-period/**', // T-M9-47: consumed by screens/home
      './src/widgets/plus-required-sheet/**', // consumed by screens/companion-invite (Plus-only invite code)
    ],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // Profile-section slices consumed by the profile-* screens (and by each
    // other one layer up) — invisible to steiger for the same reason as the
    // block above (references coming FROM `screens` don't count).
    files: [
      './src/features/edit-profile/**',
      './src/features/manage-account/**',
      './src/features/manage-reminders/**',
      './src/features/read-notifications/**',
      './src/entities/reminder/**',
      './src/entities/notification/**',
    ],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // Care-reminder mutation slices (M3). Consumed by the reminders screens
    // and the home card — references coming FROM `screens` are invisible to
    // steiger (same reason as the blocks above).
    files: [
      './src/features/manage-medication/**',
      './src/features/manage-appointment/**',
      './src/features/log-intake/**',
      './src/entities/care-reminder/**',
    ],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N6-03 health record: consumed only by screens/health-record today (B-N6-04's report builder reuses it) —
    // references coming FROM `screens` are invisible to steiger (same reason as the blocks above).
    files: ['./src/entities/health-record/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N8-03 learning: entities/course is consumed by screens/learn* and widgets/learn-home-card, the widget by
    // screens/home — references coming FROM `screens` are invisible to steiger (same reason as the blocks above).
    files: ['./src/entities/course/**', './src/widgets/learn-home-card/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N10-02 sponsored home slots: widgets/ad-slot is consumed by screens/home only (references from `screens` are
    // invisible to steiger, same reason as the blocks above).
    files: ['./src/widgets/ad-slot/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N7-05 doctors: entities/doctor + features/book-visit are consumed only by screens/doctor* — references coming
    // FROM `screens` are invisible to steiger (same reason as the blocks above).
    files: ['./src/entities/doctor/**', './src/features/book-visit/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N7-07 assistant: entities/assistant + features/assistant-chat are consumed only by screens/assistant* —
    // references coming FROM `screens` are invisible to steiger (same reason as the blocks above).
    files: ['./src/entities/assistant/**', './src/features/assistant-chat/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N7-01 services hub: consumed only by screens/services — references coming FROM `screens` are invisible to
    // steiger (same reason as the blocks above).
    files: ['./src/entities/service-hub/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-REC-04 record documents (upload sheet + record-home additions): consumed by screens/health-record and
    // screens/record-timeline — references coming FROM `screens` are invisible to steiger (same reason as above).
    files: ['./src/features/record-documents/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // Checkups foundation (M4, T-M4-05). Consumed by the checkups screens and
    // the home card (T-M4-06…09) — references coming FROM `screens` are
    // invisible to steiger (same reason as the blocks above).
    files: [
      './src/entities/checkup/**',
      './src/features/record-checkup/**',
      './src/features/manage-custom-checkup/**',
      './src/widgets/checkups-card/**',
    ],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // Fertility / TTC foundation (M5, T-M5-04). Consumed by the fertility
    // tiles widget and the log / BBT / insights screens (T-M5-05…08) —
    // references coming FROM `screens` are invisible to steiger (same reason
    // as the blocks above).
    files: [
      './src/entities/fertility/**',
      './src/features/log-fertility-day/**',
      './src/widgets/fertility-tiles/**',
    ],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // `entities/cycle` is a canonical domain entity (CLAUDE.md §5) that will be
    // consumed by more screens as cycle/API wiring lands. Don't nag about it
    // having a single reference today.
    files: ['./src/entities/cycle/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // `entities/phase-content` maps the DB-driven Phase Details content
    // (GET /cycle/phase-content/{phase}). Its only consumer is the
    // `phase-details` screen, and references coming FROM `screens` are
    // invisible to steiger (same reason as the blocks above).
    files: ['./src/entities/phase-content/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // `entities/message` maps the API's `messages` endpoint group (CLAUDE.md
    // §8.1: daily personalized messages + current mode). The home screen is its
    // first consumer; mode-awareness and the pregnancy screens will follow.
    files: ['./src/entities/message/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // `entities/health-log` maps the API's `health-logs` endpoint group
    // (CLAUDE.md §8.1: daily health log CRUD + form enums). The Add Log screen
    // is its first consumer; the calendar day-sheet and insights will follow.
    files: ['./src/entities/health-log/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // `entities/pregnancy` + the pregnancy features/screens map the API's
    // `pregnancy` endpoint group (CLAUDE.md §8.1: mode, onboarding, weekly
    // content & logs, symptoms, fetal movement, alerts). Consumed only from
    // `screens`, which steiger can't see (same reason as the blocks above).
    files: [
      './src/entities/pregnancy/**',
      './src/features/track-pregnancy/**',
    ],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // `features/log-period` is a canonical feature (CLAUDE.md §5): the "start
    // period" action on the home screen. Consumed only from `screens`, which
    // steiger can't see (same reason as the blocks above).
    files: ['./src/features/log-period/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // `entities/language` maps GET /languages — the locales the app currently
    // ships (CLAUDE.md §6). `features/switch-locale` is its consumer today;
    // one reference is enough for a slice this foundational.
    files: ['./src/entities/language/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // `features/switch-locale` is a canonical feature (CLAUDE.md §5). The
    // profile screen is its first consumer; onboarding and the app header are
    // expected to reuse it. Don't nag about the single reference today.
    files: ['./src/features/switch-locale/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // Home-page promo banners: the `banner-slideshow` widget is mounted in the
    // home screen (three slots), and `entities/banner` feeds it. Both are
    // consumed only from `screens`, which steiger can't see (same reason as the
    // blocks above), so their references look like zero/one.
    files: [
      './src/entities/banner/**',
      './src/widgets/banner-slideshow/**',
    ],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // Log sheet v2 (B-N3-03): `features/log-day` and `widgets/body-map` are
    // composed by `screens/log` (the `?sheet=log` panel and the `/log` route)
    // and later by the pregnancy / postpartum sheets — references coming FROM
    // `screens` are invisible to steiger (same reason as the blocks above).
    files: ['./src/features/log-day/**', './src/widgets/body-map/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // Log customisation (B-N3-04): `features/customize-log` is composed by
    // `screens/log-customize` only — references FROM `screens` are invisible
    // to steiger (same reason as the blocks above).
    files: ['./src/features/customize-log/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // Alerts inbox (T-M10-15): `entities/alert` is read by `screens/alerts` only — references FROM `screens`
    // are invisible to steiger (same reason as the blocks above).
    files: ['./src/entities/alert/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // Voice logging (B-N3-05): `features/voice-log` is handed to the log sheet as its `VoiceLog` slot by
    // `screens/log` only — references FROM `screens` are invisible to steiger (same reason as above).
    files: ['./src/features/voice-log/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // `widgets/intro-carousel` is the pre-signup welcome slideshow, mounted only
    // by the `welcome` screen — invisible to steiger (same reason as the blocks
    // above), so its single reference reads as zero.
    files: ['./src/widgets/intro-carousel/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // `widgets/day-tasks` is the day-scoped planner (doctor/medication
    // reminders + to-dos), mounted by the `log` and `home` screens — invisible
    // to steiger (same reason as the blocks above), so its references read as
    // zero.
    files: ['./src/widgets/day-tasks/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // «چالش امروز»: the `today-challenge` widget is mounted by the home screen
    // and `features/complete-challenge` is the tick that records a completion.
    // Both are consumed only from `screens`, which steiger can't see (same
    // reason as the blocks above), so their references read as zero/one.
    files: [
      './src/entities/challenge/**',
      './src/widgets/today-challenge/**',
      './src/features/complete-challenge/**',
    ],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // «یادآورهای امروز» (M3): the `today-reminders` widget is mounted by the
    // home and pregnancy screens — invisible to steiger (same reason as the
    // blocks above), so its references read as zero.
    files: ['./src/widgets/today-reminders/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // «خلاصه هفته»: `entities/wellbeing` holds the weekly mood/sleep/energy
    // scores and `widgets/week-summary` renders them on the home feed. Both are
    // consumed only from `screens` (home mounts the widget, the log screen
    // invalidates the entity's cache), which steiger can't see — same reason as
    // the blocks above.
    files: [
      './src/entities/wellbeing/**',
      './src/widgets/week-summary/**',
    ],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // `entities/privacy` maps GET /privacy — the admin-managed privacy-policy
    // boxes rendered by the `profile-info` sheet. Consumed only from `screens`,
    // which steiger can't see (same reason as the blocks above).
    files: ['./src/entities/privacy/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // «بر اساس سیکل فعلی شما»: `entities/article` serves the phase-matched
    // articles the home screen renders (and whose cache `log-period` drops).
    // The home screen is its main consumer and references coming FROM
    // `screens` are invisible to steiger — same reason as the blocks above.
    files: ['./src/entities/article/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N2-07 Ritme Plus: consumed by the plus-* screens and the «من» hub —
    // references coming FROM `screens` are invisible to steiger.
    files: ['./src/entities/plus/**', './src/features/purchase-plus/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N2-08: the trial banner + sheet are mounted by the home screen (cycle +
    // menopause homes) — references FROM `screens` are invisible to steiger.
    files: ['./src/widgets/plus-trial-banner/**', './src/widgets/plus-trial-sheet/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-CONTRA-02: `entities/contraception` is consumed by the contraception
    // screens and the mode screen — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/contraception/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N3-08: `entities/analysis` is consumed by the analysis screens (hub now,
    // B-N3-09…12 details) — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/analysis/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N3-09: `widgets/charts` (analysis report charts + frame) is consumed by
    // the five screens/analysis-* slices — references FROM `screens` are invisible to steiger.
    files: ['./src/widgets/charts/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-MENO-05: `entities/menopause` (the CB-MENO-02/12 API) is consumed by the menopause home and
    // stage screens — references FROM `screens` are invisible to steiger. It is the 21st entity: the
    // layer's 20-slice guideline is lifted rather than splitting one domain across screens (later
    // CB-MENO screens reuse it). [question for the user: group entities instead?]
    files: ['./src/entities/menopause/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    files: ['./src/entities', './src/entities/**'],
    rules: { 'fsd/excessive-slicing': 'off' },
  },
  {
    // B-N4-04: the companion «همدم» owner slices are consumed by screens/companion-*, screens/privacy and
    // screens/profile — references FROM `screens` are invisible to steiger. invite-companion is the 21st feature:
    // the 20-slice guideline is lifted for `features` as it already is for `entities`.
    files: ['./src/entities/companion/**', './src/features/invite-companion/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    files: ['./src/features', './src/features/**'],
    rules: { 'fsd/excessive-slicing': 'off' },
  },
  {
    // B-N5-04: `entities/postpartum` (the B-N5-01 API) is consumed by screens/postpartum, -recovery and -mood —
    // references FROM `screens` are invisible to steiger.
    files: ['./src/entities/postpartum/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-TEEN-02: `entities/teen` is consumed by screens/teen-onboarding and widgets/teen-home, which screens/home
    // mounts — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/teen/**', './src/widgets/teen-home/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-IVF-02: `entities/ivf` (the CB-IVF-01 API) is consumed by screens/ivf now and by the CB-IVF-03..05
    // screens next — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/ivf/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-LOSS-02: `entities/loss` (the CB-LOSS-01 API) is consumed by screens/loss-start, -care and -next —
    // references FROM `screens` are invisible to steiger.
    files: ['./src/entities/loss/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-COND-02: `entities/condition` (the CB-COND-01 / -06b API) is consumed by screens/conditions and
    // screens/services — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/condition/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-PELV-02: `entities/pelvic` (the CB-PELV-01 API) is consumed by screens/pelvic, screens/pelvic-kegel,
    // screens/conditions and screens/services — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/pelvic/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-TEEN-03: `widgets/linked-teen-card` is mounted by screens/home, screens/companion-home and
    // screens/companion-list — references FROM `screens` are invisible to steiger.
    files: ['./src/widgets/linked-teen-card/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N5-05: `entities/child` (the B-N5-02 API) is consumed by screens/children, child-add, child-home,
    // postpartum and companion-home — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/child/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N5-08: `features/pregnancy-tools` is mounted by screens/log-kick and screens/log-contraction —
    // references FROM `screens` are invisible to steiger.
    files: ['./src/features/pregnancy-tools/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N5-07: `features/baby-log` is mounted by screens/log-feed, screens/child-home and
    // screens/analysis-postpartum — references FROM `screens` are invisible to steiger.
    files: ['./src/features/baby-log/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N6-07: `entities/lab` and `features/upload-lab` are consumed by the screens/lab-* screens and
    // screens/analysis-labs — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/lab/**', './src/features/upload-lab/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N6-02: `entities/vital` and `features/add-vital` are consumed by screens/vitals-hub, vitals-add and
    // vitals-report — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/vital/**', './src/features/add-vital/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-NAV-03: the Today header actions (bell + light/dark), «برای امروز», «دسترسی سریع» and the + sheet «برای بعد» rows are
    // mounted by screens/home, pregnancy, postpartum, ivf and log (the quick-access edit sheet by app/sheets) —
    // references FROM `screens` are invisible to steiger.
    files: ['./src/widgets/notification-bell/**', './src/widgets/for-today/**', './src/widgets/quick-access/**', './src/widgets/log-later/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-NAV-03 took `widgets` past the 20-slice guideline (24): lifted as for `entities` and `features` rather
    // than grouping bloom's widgets from this queue. [question for the user: group the Today widgets later?]
    files: ['./src/widgets', './src/widgets/**'],
    rules: { 'fsd/excessive-slicing': 'off' },
  },
  {
    // B-N6-08: `entities/todo` is consumed by screens/todo, todo-list and todo-add — references FROM `screens` are
    // invisible to steiger.
    files: ['./src/entities/todo/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N10-04: `entities/partner-lab` is consumed by screens/home-lab and widgets/home-lab-cards; `widgets/home-lab-cards`
    // by screens/home — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/partner-lab/**', './src/widgets/home-lab-cards/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N10-05: `entities/city-service` is consumed by screens/city-services (list, detail, request, bookings) —
    // references FROM `screens` are invisible to steiger.
    files: ['./src/entities/city-service/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-DIR-07: `widgets/directory-filters` (filter sheet + result card) is consumed by screens/city-services (list and
    // map) — references FROM `screens` are invisible to steiger.
    files: ['./src/widgets/directory-filters/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N10-03: `entities/shop` is consumed by screens/shop only — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/shop/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // B-N8-04: `shared/pwa/offline` is the offline-downloads entry of the pwa segment, kept apart from
    // `shared/pwa` so the update/install UI (and its `pwa` messages) isn't pulled into every learn screen.
    files: ['./src/entities/course/api/offline.ts'],
    rules: { 'fsd/no-public-api-sidestep': 'off' },
  },
  {
    // CB-INS-03: `entities/insurance` is consumed by screens/insurance* only — references FROM `screens` are invisible to steiger.
    files: ['./src/entities/insurance/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-INS-04: `widgets/claim-documents` is used by screens/insurance-claim* only — references FROM `screens` are invisible to steiger.
    files: ['./src/widgets/claim-documents/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-DIR-11: `widgets/web-shell` is used by screens/web-city-services (and later CB-DIR-12 / CB-SHOP-10/11 web
    // screens) only — references FROM `screens` are invisible to steiger.
    files: ['./src/widgets/web-shell/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-DIR-12: `widgets/city-join-form` is used by screens/city-services-join and screens/web-city-services-business
    // only — references FROM `screens` are invisible to steiger.
    files: ['./src/widgets/city-join-form/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // CB-DIR-13: `features/review-city-booking` is used by screens/city-services (bookings list + booked page) only —
    // references FROM `screens` are invisible to steiger.
    files: ['./src/features/review-city-booking/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // companion v2 (member side): `features/join-companion` is used by screens/companion-home and
    // screens/onboarding-flow, `widgets/companion-support` by screens/companion-home|actions|learn — references
    // FROM `screens` are invisible to steiger.
    files: ['./src/features/join-companion/**', './src/widgets/companion-support/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // companion v2 (host side): `widgets/companion-invite-banner` is mounted by screens/home only — references FROM
    // `screens` are invisible to steiger.
    files: ['./src/widgets/companion-invite-banner/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // UI autopilot: `widgets/autopilot-manifest` (the agent manifest hook) is consumed by screens/autopilot-lab today and by
    // the voice surface next — references FROM `screens` are invisible to steiger.
    files: ['./src/widgets/autopilot-manifest/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
  {
    // AI credits: `entities/ai-credit` is read by screens/credits-*, profile, assistant, plus-* and the ai-gate
    // feature; `widgets/voice-log-slot` is mounted by the log screens — references FROM `screens` are invisible
    // to steiger.
    files: ['./src/entities/ai-credit/**', './src/widgets/voice-log-slot/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
]);
