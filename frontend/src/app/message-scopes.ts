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
  'children', // CB-NAV-03: the log sheet's «ثبت برای <child>» row (widgets/log-later → entities/child barrel)
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
  'labConsent', // B-N6-07: `?sheet=lab-consent` (screens/lab-consent; the consent text itself comes from the API)
  'teen', // CB-NAV-04b: quick-access sheet reads entities/teen (parent link)
  'todo', // B-N6-08: `?sheet=todo-add` (screens/todo-add)
] as const satisfies readonly MessageNamespace[];

// B-N2-02: every onboarding route mounts the one onboarding-flow screen slice
// (`profileEdit` comes with the birthday wheels of features/edit-profile; `companions` with the
// entities/companion barrel the partner step imports for the code field, B-N4-05).
const ONBOARDING = ['common', 'companions', 'onboarding', 'profileEdit'] as const satisfies readonly MessageNamespace[];
const AUTH = ['auth', 'common'] as const satisfies readonly MessageNamespace[];
const PREGNANCY = ['common', 'nav', 'pregnancy'] as const satisfies readonly MessageNamespace[];

/** Per route: the namespaces its screen (and everything it imports) uses. */
export const ROUTE_NAMESPACES = {
  home: ['articles', 'banners', 'care', 'challenge', 'checkups', 'common', 'companions', 'conditions', 'fertility', 'home', 'homeLab', 'log', 'logPeriod', 'loss', 'menopause', 'nav', 'plus', 'profileEdit', 'search', 'teen', 'todo', 'learn'], // B-N8-03 widgets/learn-home-card; CB-NAV-03 «برای امروز» to-do row (entities/todo barrel); B-N2-08 trial banner + sheet; CB-NAV-02 header search button; CB-MENO-05 menopause home; CB-TEEN-02 teen home; CB-TEEN-03 widgets/linked-teen-card (entities/companion barrel); CB-LOSS-03b care re-entry row (entities/loss)
  calendar: ['calendar', 'common', 'log', 'logPeriod', 'nav'],
  log: ['children', 'common', 'logSheet', 'nav', 'plus', 'voiceLog'], // CB-NAV-03: children = widgets/log-later via screens/log; // B-N3-03: /log renders the log sheet v2 as a page
  logCustomize: ['common', 'logCustomize', 'logSheet', 'plus'], // B-N3-04 /log/customize (the log sheet's gear; categoryLook comes via features/log-day)
  cycle: ['common', 'cycle', 'logPeriod'], // B-N1-08 cycle history (back header, no nav)
  cycleSymptoms: ['common', 'cycle', 'logPeriod'], // B-N1-08 /cycle/symptoms (screen slice shares the editor)
  cycleSettings: ['common', 'me'], // B-N1-09 /cycle/settings (copy lives under me.cycleSettings)
  profile: ['account', 'common', 'companions', 'me', 'nav', 'plus', 'profile', 'profileEdit', 'teen'], // B-N2-08: entities/plus (Me Plus card) carries the PlusFeatureGate copy; B-N4-04 entities/companion (Me row); CB-TEEN-04b entities/teen (teen's parent row)
  profileAccount: ['account', 'common', 'companions', 'me', 'nav', 'plus', 'profile', 'profileEdit', 'teen'], // B-N1-10 /profile/account (+ plus: same Me hub slice; CB-TEEN-04b teen)
  profileAppearance: ['common', 'me'], // B-N1-10 /profile/appearance
  profileDevices: ['common', 'me'], // B-N10-07 /profile/devices
  profileBackup: ['account', 'common', 'me'], // B-N10-07 /profile/backup (export hooks reuse me.privacy.mine copy)
  profileLanguage: ['common', 'me'], // B-N1-10 /profile/language
  profileNotifications: ['common', 'me'], // B-N1-11 /profile/notifications
  profilePrivacy: ['account', 'common', 'companions', 'me', 'recordExport'], // B-N1-12 /profile/privacy (DeleteAccountConfirm = account; B-N4-04 companions section)
  companions: ['children', 'common', 'companions', 'teen'], // B-N4-10b family strip children (features/invite-companion → entities/child); B-N4-04 /companions (Hamdam_List); CB-TEEN-03 parent code card (widgets/linked-teen-card)
  companionsNew: ['children', 'common', 'companions'], // B-N4-10b ChildrenPicker (entities/child); B-N4-04 /companions/new (Hamdam_Type → Access → Children → Invite → Done)
  companionDetail: ['children', 'common', 'companions'], // B-N4-10b children editor (entities/child); B-N4-04 /companions/[id] (grants, renew, revoke)
  companion: ['children', 'common', 'companionHome', 'companions', 'nav', 'teen'], // B-N4-05 /companion (Hamdam_Home, companion nav; companions = entities/companion barrel); CB-TEEN-03 linked teen cards
  companionLinks: ['children', 'common', 'companionHome', 'companions', 'nav', 'teen'], // B-N4-05 /companion/links (code entry + leave a link, Me «کد همدم»); CB-TEEN-03 same slice as /companion
  profileSupport: ['common', 'me'], // B-N1-12 /profile/support
  insurance: ['common', 'insurance', 'me', 'nav'], // B-N10-06 /insurance (info copy under me.insurance); CB-INS-03 tracker home + setup (insurance namespace, nav for the home state)
  insuranceCoverage: ['common', 'insurance'], // CB-INS-03 /insurance/coverage (nbl_Ins_Coverage, back header, no nav)
  insuranceCenters: ['common', 'insurance'], // CB-INS-06 /insurance/centers (nbl_Ins_Centers, back header, no nav)
  profileAbout: ['common', 'me'], // B-N1-12 /profile/about
  profileLegal: ['common', 'me'], // B-N1-12 /profile/legal
  profileMode: ['common', 'contraception', 'loss', 'me'], // B-N2-03 /profile/mode (copy under me.mode; CB-CONTRA-02 manage row; CB-LOSS-03b care re-entry row)
  plusPaywall: ['common', 'nav', 'plus'], // B-N2-07 /plus (teen guard reads widgets/bottom-nav)
  plusPlans: ['common', 'nav', 'plus'], // B-N2-07 /plus/plans
  plusCheckout: ['common', 'nav', 'plus'], // B-N2-07 /plus/checkout
  plusSuccess: ['common', 'plus'], // B-N2-07 /plus/success + gateway return /plus/return
  plusManage: ['common', 'plus'], // B-N2-07 /plus/manage
  pregnancy: ['care', 'common', 'loss', 'nav', 'pregnancyV2', 'search', 'teen', 'todo'], // CB-NAV-04b widgets/quick-access reads entities/teen; CB-NAV-03b «برای امروز» to-do row (widgets/for-today); CB-NAV-02 header search button; B-N5-10 post-loss care row (entities/loss)
  pregnancyLog: [...PREGNANCY, 'logSheet', 'plus', 'pregnancyV2', 'voiceLog'], // B-N3-06: default tab = log sheet v2 (pregnancy preset)
  pregnancyWeek: ['common', 'nav', 'pregnancyV2'],
  pregnancyAlerts: ['common', 'nav', 'pregnancyV2'],
  pregnancyKicks: ['common', 'pregnancyTools'], // B-N5-08 /pregnancy/kicks (Log_Kick)
  pregnancyContractions: ['common', 'pregnancyTools'], // B-N5-08 /pregnancy/contractions (Log_Contraction)
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
  services: ['common', 'conditions', 'nav', 'services'], // «خدمات» tab (B-N1-04, hub B-N7-01); CB-COND-07b: entities/condition barrel ships the nudge card
  programs: ['common', 'conditions', 'nav'], // CB-COND-02 /programs (nbl_Cond_Hub, services hub, bottom nav)
  pelvic: ['common', 'nav', 'pelvic'], // CB-PELV-02 /programs/pelvic (nbl_Pelvic_Plan, bottom nav) + /kegel below
  pelvicKegel: ['common', 'pelvic'], // CB-PELV-02 /programs/pelvic/kegel (nbl_Pelvic_Kegel, full-screen flow, no nav)
  programsBleeding: ['common', 'conditions'], // CB-COND-05 /programs/bleeding (nbl_Cond_Bleed, PBAC form, no nav)
  programsPain: ['common', 'conditions'], // CB-COND-03 /programs/pain (nbl_Cond_Endo, form, no nav)
  programsPmdd: ['common', 'conditions'], // CB-COND-04 /programs/pmdd (nbl_Cond_PMDD, form, no nav)
  search: ['common', 'search'], // CB-NAV-02 /search (global search, flow — no nav)
  menopauseStage: ['common', 'menopause'], // CB-MENO-05 /menopause/stage (stage form, no nav)
  menopauseHotFlash: ['common', 'menopause'], // CB-MENO-07 /menopause/hot-flash (timer flow, no nav)
  menopauseAlert: ['common', 'menopause'], // CB-MENO-09 /menopause/alert (bleeding alert, back header, no nav)
  menopauseTreatment: ['common', 'menopause'], // CB-MENO-10 /menopause/treatment (treatment & care, back header, no nav)
  menopauseLog: ['common', 'logSheet', 'menopause', 'plus', 'voiceLog'], // CB-MENO-06 /menopause/log (log sheet v2 menopause preset as a page, no nav)
  menopauseScore: ['common', 'menopause', 'nav'], // CB-MENO-08 /menopause/score (menopause tab «علائم», bottom nav)
  menopauseScoreQuestionnaire: ['common', 'menopause', 'nav'], // CB-MENO-08 /menopause/score/questionnaire (form, no nav; same screen slice as /menopause/score)
  menopauseReport: ['common', 'healthRecord', 'menopause', 'plus', 'record', 'recordExport'], // CB-MENO-11 /menopause/report (doctor report on bloom's builder, back header, no nav; record: the record-export barrel's CB-REC-05 code page)
  postpartum: ['care', 'children', 'common', 'logSheet', 'nav', 'plus', 'postpartum', 'teen', 'todo'], // CB-NAV-04b quick-access → entities/teen; CB-NAV-03b «برای امروز» (widgets/today-reminders → care, widgets/for-today → todo) + «دسترسی سریع»; B-N5-04 /postpartum (v15_Main; mood chips save through features/log-day); B-N5-05 hero children row + child vaccines (entities/child)
  postpartumSetup: ['care', 'children', 'common', 'logSheet', 'nav', 'plus', 'postpartum', 'teen', 'todo'], // CB-NAV-04b quick-access → entities/teen; CB-NAV-03b same screen slice as /postpartum; B-N5-04 /postpartum/setup (same screen slice as /postpartum)
  postpartumRecovery: ['common', 'postpartum'], // B-N5-04 /postpartum/recovery (v15_Recovery, back header)
  postpartumMood: ['common', 'postpartum'], // B-N5-04 /postpartum/mood (v15_MoodCheck, EPDS + safety)
  children: ['children', 'common', 'nav'], // B-N5-05 /children (v15_Children; «کودک» tab for 0 / ≥2 children)
  childForm: ['children', 'common'], // B-N5-05 /children/new + /children/[id]/edit (v15_AddChild, form, no nav)
  childHome: ['babyLog', 'children', 'common', 'nav'], // B-N5-05 /children/[id] (v16_ChildHome, «کودک» tab root); B-N5-07 «امروز» rows (features/baby-log)
  logFeed: ['babyLog', 'children', 'common'], // B-N5-07 /children/[id]/feeding + /children/feeding (nbl_Log_Feed, close header, no nav; children = entities/child barrel)
  childSection: ['children', 'common'], // B-N5-06 /children/[id]/{growth,vaccines,milestones,learn} (v16_Growth / _Vaccines / _Milestones / _Learn; back header, no nav)
  ivf: ['care', 'common', 'ivf', 'nav', 'teen', 'todo'], // CB-NAV-04b quick-access → entities/teen; CB-NAV-03b «برای امروز» (care reminders + to-do row) + «دسترسی سریع»; CB-IVF-02 /ivf (nbl_IVF_Home, TTC IVF sub-mode home)
  ivfScan: ['common', 'ivf'], // CB-IVF-04 /ivf/scan (nbl_IVF_Scan, back header, no nav)
  lossStart: ['common', 'companions', 'loss'], // CB-LOSS-02 /loss (Loss_Start, full screen, no nav; companions = entities/companion barrel)
  lossCare: ['common', 'loss'], // CB-LOSS-02 /loss/care (Loss_Care; «ثبت» opens the shell's log sheet)
  lossNext: ['common', 'loss'], // CB-LOSS-02 /loss/next (Loss_Next)
  recordExport: ['common', 'healthRecord', 'menopause', 'plus', 'record', 'recordExport'], // B-N6-04 /record/export + /record/export/preview (CB-MENO-11: menopause = the paper's menopause section)
  sharedReport: ['common', 'healthRecord', 'menopause', 'plus', 'record', 'recordExport'], // B-N6-04 public /shared/report/[token] (CB-MENO-11: menopause = the paper's menopause section; CB-REC-05: record = a summary's documents, also /d/[token])
  healthRecord: ['common', 'healthRecord', 'profileEdit', 'record'], // B-N6-03 /record (Record_Summary; profileEdit = the height/weight QuickEditSheet); CB-REC-04 record = home extras + upload sheet
  recordTimeline: ['common', 'record'], // CB-REC-04 /record/timeline (nbl_Rec_Timeline, back header, no nav)
  recordDocument: ['common', 'record'], // CB-REC-04 /record/documents/[id] (nbl_Rec_Doc, back header, no nav)
  recordShare: ['common', 'record'], // CB-REC-05 /record/share (nbl_Rec_Share, back header, no nav)
  recordEmergency: ['common', 'record'], // CB-REC-05 /record/emergency (nbl_Rec_Emergency, back header, no nav)
  doctorCode: ['common', 'healthRecord', 'menopause', 'plus', 'record', 'recordExport'], // CB-REC-05 public /d (code entry → the shared report paper)
  publicEmergency: ['common', 'record'], // CB-REC-05 public /emergency/[token] (minimal card; card strings in common.emergencyCard)
  ivfMeds: ['common', 'ivf', 'nav'], // CB-IVF-03 /ivf/meds (nbl_IVF_Meds, IVF stage tab «درمان»)
  ivfMedForm: ['common', 'ivf', 'nav'], // CB-IVF-03 /ivf/meds/new, /ivf/meds/[id] (form, no nav; same screen slice as /ivf/meds)
  ivfCycle: ['common', 'ivf'], // CB-IVF-06b /ivf/cycle/new (setup) + /ivf/cycle (stage + dates editor); forms, no nav
  ivfTww: ['common', 'ivf'], // CB-IVF-05 /ivf/tww (nbl_IVF_TWW; back button, no nav — sensitive; «ثبت علائم» opens the global log sheet)
  labs: ['common', 'labs', 'plus'], // B-N6-07 /labs + /labs/new (Lab_Intro / Lab_Upload; back header, no nav; plus = PlusFeatureGate copy)
  labDetail: ['common', 'labs'], // B-N6-07 /labs/[id], /labs/[id]/{processing,verify}, /labs/[id]/markers/[mid] (Lab_Result / _Processing / _Verify / _Marker)
  analysis: ['analysis', 'common', 'nav', 'plus'], // B-N3-08 /analysis hub + /analysis/* stubs (one screen slice; PlusGate copy = plus.gate)
  analysisLabs: ['analysis', 'common', 'labs', 'nav', 'plus'], // B-N6-09 /analysis/labs (An_Labs rows: «مرجع: …» = labs.result.reference)
  analysisHub: ['analysis', 'analysisPostpartum', 'analysisPregnancy', 'babyLog', 'children', 'common', 'nav', 'plus'], // B-N3-12 /analysis itself: + the pregnancy hub (screens/analysis-pregnancy); B-N5-07 + the postpartum hub (screens/analysis-postpartum, features/baby-log, entities/child)
  analysisPregnancyWeight: ['analysisPregnancy', 'common', 'nav', 'plus'], // B-N3-12 /analysis/pregnancy-weight (An_PregWeight; the slice's hub cards carry plus.gate)
  analysisReport: ['analysis', 'common', 'nav'], // B-N3-09 /analysis/{cycle,period,symptoms,body} (no Plus gate; correlations keeps `analysis`)
  teenOnboarding: ['common', 'teen'], // CB-TEEN-02 /teen/onboarding (nbl_Teen_Onb, form — no nav)
  teenParent: ['children', 'common', 'companions', 'teen'], // B-N4-10b features/invite-companion barrel pulls entities/child; CB-TEEN-03 /teen/parent (nbl_Teen_Parent, flow — no nav; companions = InviteCodeCard + entities/companion)
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
  vitals: ['common', 'nav', 'vitals'], // B-N6-02 /vitals (Vitals_Hub; plan editor sheet; bottom nav)
  vitalsAdd: ['common', 'vitals'], // B-N6-02 /vitals/{bp,glucose,heart-rate}/new (features/add-vital; CalendarPicker = common)
  vitalsReport: ['common', 'vitals'], // B-N6-02 /vitals/{bp,glucose,heart-rate} reports
  todo: ['common', 'nav', 'todo'], // B-N6-08 /todo (Todo_Home / Todo_Empty; bottom nav)
  todoList: ['common', 'todo'], // B-N6-08 /todo/lists/[id] (Todo_List; CalendarPicker of the add sheet = common)
  learn: ['common', 'learn', 'nav'], // B-N8-03 /learn (Learn_Hub; bottom nav)
  learnCourse: ['common', 'learn'], // B-N8-03 /learn/[course]
  learnLesson: ['common', 'learn'], // B-N8-03 /learn/[course]/[lesson] (video / audio / PDF)
  learnUnlocked: ['common', 'learn'], // B-N8-03 /learn/unlocked/[grant]
  learnDownloads: ['common', 'learn', 'nav'], // B-N8-04 /learn/downloads (Learn_Downloads; bottom nav, offline-capable)
  homeLab: ['common', 'homeLab'], // B-N10-04 /services/home-lab, /services/home-lab/new, /services/home-lab/[id]
  shop: ['common', 'shop'], // B-N10-03 /shop, /shop/[category] (categories, products, link-out interstitial)
  cityServices: ['common', 'cityServices'], // B-N10-05 /city-services, /city-services/[id], /city-services/[id]/request, /profile/bookings
  doctors: ['common', 'doctors'], // B-N7-05 /services/doctors*, /services/bookings* (directory, profile, book, return, booked, chat)
  assistant: ['common', 'nav', 'assistant'], // B-N7-07 /assistant (hub; bottom nav)
  assistantDept: ['common', 'assistant'], // B-N7-07 /assistant/[dept]
  assistantChat: ['common', 'assistant'], // B-N7-07 /assistant/chat/[id]
  assistantSummary: ['common', 'assistant'], // B-N7-07 /assistant/chat/[id]/summary
  assistantHandoff: ['common', 'assistant', 'doctors'], // B-N7-07 /assistant/chat/[id]/handoff (own bookings use the doctors copy)
  assistantHistory: ['common', 'assistant', 'doctors'], // B-N7-07 /assistant/history
  assistantProfile: ['common', 'assistant'], // B-N7-07 /assistant/profile
} as const satisfies Record<string, readonly MessageNamespace[]>;

export type MessageRoute = keyof typeof ROUTE_NAMESPACES;
