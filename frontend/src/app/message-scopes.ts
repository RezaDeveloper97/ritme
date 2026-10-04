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
  'logSheet', // B-N3-03: `?sheet=log` (log sheet v2) opens over any route
  'voiceLog', // B-N3-05: the log sheet's voice tab (features/voice-log)
  'nav', // B-N1-04: the FAB's `?sheet=log` (LogSheet) can open over any route
  'plus', // B-N3-03: the log sheet's voice tab is a PlusFeatureGate (plus.voice_log)
  'pwa',
  'reminders',
] as const satisfies readonly MessageNamespace[];

// B-N2-02: every onboarding route mounts the one onboarding-flow screen slice
// (`profileEdit` comes with the birthday wheels of features/edit-profile; `companions` with the
// entities/companion barrel the partner step imports for the code field, B-N4-05).
const ONBOARDING = ['common', 'companions', 'onboarding', 'profileEdit'] as const satisfies readonly MessageNamespace[];
const AUTH = ['auth', 'common'] as const satisfies readonly MessageNamespace[];
const PREGNANCY = ['common', 'nav', 'pregnancy'] as const satisfies readonly MessageNamespace[];

/** Per route: the namespaces its screen (and everything it imports) uses. */
export const ROUTE_NAMESPACES = {
  home: ['articles', 'banners', 'care', 'challenge', 'checkups', 'common', 'companions', 'fertility', 'home', 'log', 'logPeriod', 'menopause', 'nav', 'plus', 'profileEdit', 'search', 'teen'], // B-N2-08 trial banner + sheet; CB-NAV-02 header search button; CB-MENO-05 menopause home; CB-TEEN-02 teen home; CB-TEEN-03 widgets/linked-teen-card (entities/companion barrel)
  calendar: ['calendar', 'common', 'log', 'logPeriod', 'nav'],
  log: ['common', 'logSheet', 'nav', 'plus', 'voiceLog'], // B-N3-03: /log renders the log sheet v2 as a page
  logCustomize: ['common', 'logCustomize', 'logSheet', 'plus'], // B-N3-04 /log/customize (the log sheet's gear; categoryLook comes via features/log-day)
  cycle: ['common', 'cycle', 'logPeriod'], // B-N1-08 cycle history (back header, no nav)
  cycleSymptoms: ['common', 'cycle', 'logPeriod'], // B-N1-08 /cycle/symptoms (screen slice shares the editor)
  cycleSettings: ['common', 'me'], // B-N1-09 /cycle/settings (copy lives under me.cycleSettings)
  profile: ['account', 'common', 'companions', 'me', 'nav', 'plus', 'profile', 'profileEdit'], // B-N2-08: entities/plus (Me Plus card) carries the PlusFeatureGate copy; B-N4-04 entities/companion (Me row)
  profileAccount: ['account', 'common', 'companions', 'me', 'nav', 'plus', 'profile', 'profileEdit'], // B-N1-10 /profile/account (+ plus: same Me hub slice)
  profileAppearance: ['common', 'me'], // B-N1-10 /profile/appearance
  profileLanguage: ['common', 'me'], // B-N1-10 /profile/language
  profileNotifications: ['common', 'me'], // B-N1-11 /profile/notifications
  profilePrivacy: ['account', 'common', 'companions', 'me'], // B-N1-12 /profile/privacy (DeleteAccountConfirm = account; B-N4-04 companions section)
  companions: ['common', 'companions', 'teen'], // B-N4-04 /companions (Hamdam_List); CB-TEEN-03 parent code card (widgets/linked-teen-card)
  companionsNew: ['common', 'companions'], // B-N4-04 /companions/new (Hamdam_Type → Access → Children → Invite → Done)
  companionDetail: ['common', 'companions'], // B-N4-04 /companions/[id] (grants, renew, revoke)
  companion: ['common', 'companionHome', 'companions', 'nav', 'teen'], // B-N4-05 /companion (Hamdam_Home, companion nav; companions = entities/companion barrel); CB-TEEN-03 linked teen cards
  companionLinks: ['common', 'companionHome', 'companions', 'nav', 'teen'], // B-N4-05 /companion/links (code entry + leave a link, Me «کد همدم»); CB-TEEN-03 same slice as /companion
  profileSupport: ['common', 'me'], // B-N1-12 /profile/support
  profileAbout: ['common', 'me'], // B-N1-12 /profile/about
  profileLegal: ['common', 'me'], // B-N1-12 /profile/legal
  profileMode: ['common', 'contraception', 'me'], // B-N2-03 /profile/mode (copy under me.mode; CB-CONTRA-02 manage row)
  plusPaywall: ['common', 'nav', 'plus'], // B-N2-07 /plus (teen guard reads widgets/bottom-nav)
  plusPlans: ['common', 'nav', 'plus'], // B-N2-07 /plus/plans
  plusCheckout: ['common', 'nav', 'plus'], // B-N2-07 /plus/checkout
  plusSuccess: ['common', 'plus'], // B-N2-07 /plus/success + gateway return /plus/return
  plusManage: ['common', 'plus'], // B-N2-07 /plus/manage
  pregnancy: ['care', 'common', 'nav', 'pregnancyV2', 'search'], // CB-NAV-02 header search button
  pregnancyLog: [...PREGNANCY, 'logSheet', 'plus', 'pregnancyV2', 'voiceLog'], // B-N3-06: default tab = log sheet v2 (pregnancy preset)
  pregnancyWeek: ['common', 'nav', 'pregnancyV2'],
  pregnancyAlerts: ['common', 'nav', 'pregnancyV2'],
  pregnancyCalendar: ['care', 'common', 'nav', 'pregnancyV2'],
  pregnancyOnboarding: ['common', 'pregnancy', 'pregnancyV2', 'profileEdit'],
  pregnancySetup: ['common', 'pregnancy', 'pregnancyV2', 'profileEdit'],
  reminders: ['care', 'common'],
  reminderForm: ['care', 'common', 'companions'], // B-N4-06 medication / appointment forms: «ثبت برای چه کسی؟» (entities/companion barrel)
  checkups: ['checkups', 'common'],
  contraception: ['common', 'contraception'], // CB-CONTRA-02 /contraception (pill pack)
  contraceptionSetup: ['common', 'contraception'], // CB-CONTRA-02 /contraception/setup
  contraceptionMissed: ['common', 'contraception'], // CB-CONTRA-03 /contraception/missed
  contraceptionOther: ['common', 'contraception'], // CB-CONTRA-03 /contraception/other
  uiKit: ['common'], // dev-only /dev/ui-kit showcase (B-N1-03)
  services: ['common', 'nav', 'services'], // «خدمات» tab (B-N1-04)
  search: ['common', 'search'], // CB-NAV-02 /search (global search, flow — no nav)
  menopauseStage: ['common', 'menopause'], // CB-MENO-05 /menopause/stage (stage form, no nav)
  menopauseHotFlash: ['common', 'menopause'], // CB-MENO-07 /menopause/hot-flash (timer flow, no nav)
  menopauseAlert: ['common', 'menopause'], // CB-MENO-09 /menopause/alert (bleeding alert, back header, no nav)
  menopauseLog: ['common', 'logSheet', 'menopause', 'plus', 'voiceLog'], // CB-MENO-06 /menopause/log (log sheet v2 menopause preset as a page, no nav)
  menopauseScore: ['common', 'menopause', 'nav'], // CB-MENO-08 /menopause/score (menopause tab «علائم», bottom nav)
  menopauseScoreQuestionnaire: ['common', 'menopause', 'nav'], // CB-MENO-08 /menopause/score/questionnaire (form, no nav; same screen slice as /menopause/score)
  postpartum: ['common', 'logSheet', 'nav', 'plus', 'postpartum'], // B-N5-04 /postpartum (v15_Main; mood chips save through features/log-day)
  postpartumSetup: ['common', 'logSheet', 'nav', 'plus', 'postpartum'], // B-N5-04 /postpartum/setup (same screen slice as /postpartum)
  postpartumRecovery: ['common', 'postpartum'], // B-N5-04 /postpartum/recovery (v15_Recovery, back header)
  postpartumMood: ['common', 'postpartum'], // B-N5-04 /postpartum/mood (v15_MoodCheck, EPDS + safety)
  ivf: ['common', 'ivf', 'nav'], // CB-IVF-02 /ivf (nbl_IVF_Home, TTC IVF sub-mode home)
  ivfScan: ['common', 'ivf'], // CB-IVF-04 /ivf/scan (nbl_IVF_Scan, back header, no nav)
  lossStart: ['common', 'companions', 'loss'], // CB-LOSS-02 /loss (Loss_Start, full screen, no nav; companions = entities/companion barrel)
  lossCare: ['common', 'loss'], // CB-LOSS-02 /loss/care (Loss_Care; «ثبت» opens the shell's log sheet)
  lossNext: ['common', 'loss'], // CB-LOSS-02 /loss/next (Loss_Next)
  ivfMeds: ['common', 'ivf', 'nav'], // CB-IVF-03 /ivf/meds (nbl_IVF_Meds, IVF stage tab «درمان»)
  ivfMedForm: ['common', 'ivf', 'nav'], // CB-IVF-03 /ivf/meds/new, /ivf/meds/[id] (form, no nav; same screen slice as /ivf/meds)
  analysis: ['analysis', 'common', 'nav', 'plus'], // B-N3-08 /analysis hub + /analysis/* stubs (one screen slice; PlusGate copy = plus.gate)
  analysisHub: ['analysis', 'analysisPregnancy', 'common', 'nav', 'plus'], // B-N3-12 /analysis itself: + the pregnancy hub (screens/analysis-pregnancy)
  analysisPregnancyWeight: ['analysisPregnancy', 'common', 'nav', 'plus'], // B-N3-12 /analysis/pregnancy-weight (An_PregWeight; the slice's hub cards carry plus.gate)
  analysisReport: ['analysis', 'common', 'nav'], // B-N3-09 /analysis/{cycle,period,symptoms,body} (no Plus gate; correlations keeps `analysis`)
  teenOnboarding: ['common', 'teen'], // CB-TEEN-02 /teen/onboarding (nbl_Teen_Onb, form — no nav)
  teenParent: ['common', 'companions', 'teen'], // CB-TEEN-03 /teen/parent (nbl_Teen_Parent, flow — no nav; companions = InviteCodeCard + entities/companion)
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
  onboardingPartnerLinked: ONBOARDING, // B-N4-05
  onboardingSettingUp: ONBOARDING,
} as const satisfies Record<string, readonly MessageNamespace[]>;

export type MessageRoute = keyof typeof ROUTE_NAMESPACES;
