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

// B-N2-02: every onboarding route mounts the one onboarding-flow screen slice
// (`profileEdit` comes with the birthday wheels of features/edit-profile).
const ONBOARDING = ['common', 'onboarding', 'profileEdit'] as const satisfies readonly MessageNamespace[];
const AUTH = ['auth', 'common'] as const satisfies readonly MessageNamespace[];
const PREGNANCY = ['common', 'nav', 'pregnancy'] as const satisfies readonly MessageNamespace[];

/** Per route: the namespaces its screen (and everything it imports) uses. */
export const ROUTE_NAMESPACES = {
  home: ['articles', 'banners', 'care', 'challenge', 'checkups', 'common', 'fertility', 'home', 'log', 'logPeriod', 'nav', 'plus', 'profileEdit'], // B-N2-08 trial banner + sheet
  calendar: ['calendar', 'common', 'log', 'logPeriod', 'nav'],
  log: ['common', 'log', 'nav'],
  cycle: ['common', 'cycle', 'logPeriod'], // B-N1-08 cycle history (back header, no nav)
  cycleSymptoms: ['common', 'cycle', 'logPeriod'], // B-N1-08 /cycle/symptoms (screen slice shares the editor)
  cycleSettings: ['common', 'me'], // B-N1-09 /cycle/settings (copy lives under me.cycleSettings)
  profile: ['account', 'common', 'me', 'nav', 'plus', 'profile', 'profileEdit'], // B-N2-08: entities/plus (Me Plus card) carries the PlusFeatureGate copy
  profileAccount: ['account', 'common', 'me', 'nav', 'plus', 'profile', 'profileEdit'], // B-N1-10 /profile/account (+ plus: same Me hub slice)
  profileAppearance: ['common', 'me'], // B-N1-10 /profile/appearance
  profileLanguage: ['common', 'me'], // B-N1-10 /profile/language
  profileNotifications: ['common', 'me'], // B-N1-11 /profile/notifications
  profilePrivacy: ['account', 'common', 'me'], // B-N1-12 /profile/privacy (DeleteAccountConfirm = account)
  profileSupport: ['common', 'me'], // B-N1-12 /profile/support
  profileAbout: ['common', 'me'], // B-N1-12 /profile/about
  profileLegal: ['common', 'me'], // B-N1-12 /profile/legal
  profileMode: ['common', 'contraception', 'me'], // B-N2-03 /profile/mode (copy under me.mode; CB-CONTRA-02 manage row)
  profileModeLoss: ['common', 'contraception', 'me'], // B-N2-03 /profile/mode/loss (same screen slice as /profile/mode)
  plusPaywall: ['common', 'nav', 'plus'], // B-N2-07 /plus (teen guard reads widgets/bottom-nav)
  plusPlans: ['common', 'nav', 'plus'], // B-N2-07 /plus/plans
  plusCheckout: ['common', 'nav', 'plus'], // B-N2-07 /plus/checkout
  plusSuccess: ['common', 'plus'], // B-N2-07 /plus/success + gateway return /plus/return
  plusManage: ['common', 'plus'], // B-N2-07 /plus/manage
  pregnancy: ['care', 'common', 'nav', 'pregnancyV2'],
  pregnancyLog: [...PREGNANCY, 'pregnancyV2'],
  pregnancyWeek: ['common', 'nav', 'pregnancyV2'],
  pregnancyAlerts: ['common', 'nav', 'pregnancyV2'],
  pregnancyCalendar: ['care', 'common', 'nav', 'pregnancyV2'],
  pregnancyOnboarding: ['common', 'pregnancy', 'pregnancyV2', 'profileEdit'],
  pregnancySetup: ['common', 'pregnancy', 'pregnancyV2', 'profileEdit'],
  reminders: ['care', 'common'],
  checkups: ['checkups', 'common'],
  contraception: ['common', 'contraception'], // CB-CONTRA-02 /contraception (pill pack)
  contraceptionSetup: ['common', 'contraception'], // CB-CONTRA-02 /contraception/setup
  contraceptionMissed: ['common', 'contraception'], // CB-CONTRA-03 /contraception/missed
  contraceptionOther: ['common', 'contraception'], // CB-CONTRA-03 /contraception/other
  uiKit: ['common'], // dev-only /dev/ui-kit showcase (B-N1-03)
  services: ['common', 'nav', 'services'], // «خدمات» tab (B-N1-04)
  fertilityLog: ['common', 'fertility'],
  fertilityBbt: ['common', 'fertility'],
  fertilityInsights: ['common', 'fertility'],
  splash: [...AUTH, 'welcome'], // B-N1-05: the splash ring comes from widgets/intro-carousel
  signup: AUTH,
  otp: AUTH,
  welcome: ['common', 'welcome'],
  onboardingName: ONBOARDING,
  onboardingGender: ONBOARDING,
  onboardingIntention: ONBOARDING,
  onboardingCycle: ONBOARDING,
  onboardingPregnancyBasis: ONBOARDING,
  onboardingMenopause: ONBOARDING,
  onboardingConditions: ONBOARDING,
  onboardingHealth: ONBOARDING,
  onboardingPartner: ONBOARDING,
  onboardingSettingUp: ONBOARDING,
} as const satisfies Record<string, readonly MessageNamespace[]>;

export type MessageRoute = keyof typeof ROUTE_NAMESPACES;
