import type { MessageNamespace } from '@/shared/i18n';

/**
 * Which message namespaces each part of the app ships to the client.
 *
 * Every namespace handed to a `NextIntlClientProvider` is serialised into the
 * page, and the app used to hand all of them to every route (~50 KB of `fa`
 * strings, perf baseline §1.3/§3 #9). Now the locale layout ships only the
 * {@link SHELL_NAMESPACES}, and each route wraps its screen in
 * `<RouteMessages route="…">`, which ships that route's list.
 *
 * A nested provider replaces its parent's messages rather than merging them, so
 * each list must cover its whole subtree. `message-scopes.test.ts` walks the
 * import graph from the layout and from every `page.tsx` and fails when a list
 * and the `useTranslations('…')` calls it has to serve disagree — so adding a
 * translated component to a screen means adding its namespace here.
 */

/**
 * What the layout itself renders around every route: the PWA prompts
 * (`pwa`), `AppSheet` (`common`), and every sheet in `app/sheets/registry.tsx`,
 * because `?sheet=<id>` can open any of them over any route.
 */
export const SHELL_NAMESPACES = [
  'articles',
  'care',
  'checkups',
  'common',
  'notifications',
  'phaseDetails',
  'profile',
  'profileEdit',
  'profileInfo',
  'nav', // B-N1-04: the FAB's `?sheet=log` (LogSheet) can open over any route
  'pwa',
  'reminders',
] as const satisfies readonly MessageNamespace[];

const ONBOARDING = ['common', 'onboarding'] as const satisfies readonly MessageNamespace[];
const ONBOARDING_WITH_PROFILE = [
  'common',
  'onboarding',
  'profileEdit',
] as const satisfies readonly MessageNamespace[];
const AUTH = ['auth', 'common'] as const satisfies readonly MessageNamespace[];
const PREGNANCY = ['common', 'nav', 'pregnancy'] as const satisfies readonly MessageNamespace[];

/** Per route: the namespaces its screen (and everything it imports) uses. */
export const ROUTE_NAMESPACES = {
  home: ['articles', 'banners', 'care', 'challenge', 'checkups', 'common', 'fertility', 'home', 'logPeriod', 'nav', 'profileEdit'],
  calendar: ['calendar', 'common', 'log', 'logPeriod', 'nav'],
  log: ['common', 'log', 'nav'],
  cycle: ['common', 'cycle', 'home', 'logPeriod', 'nav'],
  profile: ['account', 'common', 'me', 'nav', 'profile', 'profileEdit'],
  profileAccount: ['account', 'common', 'me', 'nav', 'profile', 'profileEdit'], // B-N1-10 /profile/account
  profileAppearance: ['common', 'me'], // B-N1-10 /profile/appearance
  profileLanguage: ['common', 'me'], // B-N1-10 /profile/language
  pregnancy: ['care', 'common', 'nav', 'pregnancyV2'],
  pregnancyLog: [...PREGNANCY, 'pregnancyV2'],
  pregnancyWeek: ['common', 'nav', 'pregnancyV2'],
  pregnancyAlerts: ['common', 'nav', 'pregnancyV2'],
  pregnancyCalendar: ['care', 'common', 'nav', 'pregnancyV2'],
  pregnancyOnboarding: ['common', 'pregnancy', 'pregnancyV2', 'profileEdit'],
  pregnancySetup: ['common', 'pregnancy', 'pregnancyV2', 'profileEdit'],
  reminders: ['care', 'common'],
  checkups: ['checkups', 'common'],
  uiKit: ['common'], // dev-only /dev/ui-kit showcase (B-N1-03)
  services: ['common', 'nav', 'services'], // «خدمات» tab (B-N1-04)
  fertilityLog: ['common', 'fertility'],
  fertilityBbt: ['common', 'fertility'],
  fertilityInsights: ['common', 'fertility'],
  splash: [...AUTH, 'welcome'], // B-N1-05: the splash ring comes from widgets/intro-carousel
  signup: AUTH,
  otp: AUTH,
  welcome: ['common', 'welcome'],
  onboardingBirthday: ONBOARDING,
  onboardingConditions: ONBOARDING,
  onboardingCycleDuration: ONBOARDING,
  onboardingCycleLen: ONBOARDING,
  onboardingHeight: ONBOARDING,
  onboardingIntention: ONBOARDING,
  onboardingName: ONBOARDING,
  onboardingPeriodLen: ONBOARDING,
  onboardingWeight: ONBOARDING,
  onboardingPregnancyBasis: ONBOARDING_WITH_PROFILE,
  onboardingSettingUp: ONBOARDING_WITH_PROFILE,
} as const satisfies Record<string, readonly MessageNamespace[]>;

export type MessageRoute = keyof typeof ROUTE_NAMESPACES;
