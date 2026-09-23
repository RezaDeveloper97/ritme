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
  'common',
  'notifications',
  'phaseDetails',
  'profile',
  'profileEdit',
  'profileInfo',
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
  home: ['banners', 'care', 'challenge', 'common', 'home', 'logPeriod', 'nav', 'profileEdit'],
  calendar: ['calendar', 'common', 'log', 'logPeriod', 'nav'],
  log: ['common', 'log', 'nav'],
  cycle: ['common', 'cycle', 'home', 'logPeriod', 'nav'],
  profile: ['common', 'nav', 'profile', 'profileEdit'],
  pregnancy: ['care', 'common', 'nav', 'pregnancy'],
  pregnancyLog: PREGNANCY,
  pregnancyOnboarding: ['common', 'nav', 'pregnancy', 'profileEdit'],
  splash: AUTH,
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
