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
    // «بر اساس سیکل فعلی شما»: `entities/article` serves the phase-matched
    // articles the home screen renders (and whose cache `log-period` drops).
    // The home screen is its main consumer and references coming FROM
    // `screens` are invisible to steiger — same reason as the blocks above.
    files: ['./src/entities/article/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
]);
