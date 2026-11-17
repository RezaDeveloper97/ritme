import type enAccount from '../messages/en/account.json';
import type enAnalysis from '../messages/en/analysis.json';
import type enAnalysisPregnancy from '../messages/en/analysis-pregnancy.json';
import type enAnalysisPostpartum from '../messages/en/analysis-postpartum.json'; // B-N5-07
import type enBabyLog from '../messages/en/baby-log.json'; // B-N5-07
import type enArticles from '../messages/en/articles.json';
import type enAuth from '../messages/en/auth.json';
import type enBanners from '../messages/en/banners.json';
import type enCalendar from '../messages/en/calendar.json';
import type enChallenge from '../messages/en/challenge.json';
import type enCommon from '../messages/en/common.json';
import type enContraception from '../messages/en/contraception.json';
import type enCycle from '../messages/en/cycle.json';
import type enDayTasks from '../messages/en/day-tasks.json';
import type enHome from '../messages/en/home.json';
import type enCare from '../messages/en/care.json';
import type enCheckups from '../messages/en/checkups.json';
import type enChildren from '../messages/en/children.json';
import type enFertility from '../messages/en/fertility.json';
import type enLog from '../messages/en/log.json';
import type enLogPeriod from '../messages/en/log-period.json';
import type enLogCustomize from '../messages/en/log-customize.json';
import type enLogSheet from '../messages/en/log-sheet.json';
import type enAlerts from '../messages/en/alerts.json';
import type enVoiceLog from '../messages/en/voice-log.json';
import type enAiCredit from '../messages/en/ai-credit.json';
import type enAiGate from '../messages/en/ai-gate.json';
import type enAutopilot from '../messages/en/autopilot.json';
import type enAutopilotCatalog from '../messages/en/autopilot-catalog.json';
import type enCredits from '../messages/en/credits.json';
import type enVitals from '../messages/en/vitals.json'; // B-N6-02
import type enTodo from '../messages/en/todo.json'; // B-N6-08
import type enLearn from '../messages/en/learn.json'; // B-N8-03
import type enDoctors from '../messages/en/doctors.json'; // B-N7-05
import type enAssistant from '../messages/en/assistant.json'; // B-N7-07
import type enLogTaxonomy from '../messages/en/log-taxonomy.json';
import type enHealthRecord from '../messages/en/health-record.json';
import type enRecordExport from '../messages/en/record-export.json';
import type enRecord from '../messages/en/record.json';
import type enLoss from '../messages/en/loss.json';
import type enConditions from '../messages/en/conditions.json';
import type enPelvic from '../messages/en/pelvic.json'; // CB-PELV-02
import type enMe from '../messages/en/me.json';
import type enIvf from '../messages/en/ivf.json'; // CB-IVF-02
import type enInsurance from '../messages/en/insurance.json'; // CB-INS-03
import type enMenopause from '../messages/en/menopause.json';
import type enNav from '../messages/en/nav.json';
import type enNotifications from '../messages/en/notifications.json';
import type enOnboarding from '../messages/en/onboarding.json';
import type enPhaseDetails from '../messages/en/phase-details.json';
import type enPregnancy from '../messages/en/pregnancy.json';
import type enPregnancyV2 from '../messages/en/pregnancy-v2.json';
import type enPostpartum from '../messages/en/postpartum.json';
import type enPregnancyTools from '../messages/en/pregnancy-tools.json'; // B-N5-08
import type enLabs from '../messages/en/labs.json'; // B-N6-07
import type enLabConsent from '../messages/en/lab-consent.json'; // B-N6-07
import type enPwa from '../messages/en/pwa.json';
import type enProfile from '../messages/en/profile.json';
import type enProfileEdit from '../messages/en/profile-edit.json';
import type enProfileInfo from '../messages/en/profile-info.json';
import type enReminders from '../messages/en/reminders.json';
import type enTeen from '../messages/en/teen.json'; // CB-TEEN-02
import type enWelcome from '../messages/en/welcome.json';
import type enServices from '../messages/en/services.json';
import type enShop from '../messages/en/shop.json'; // B-N10-03
import type enCityServices from '../messages/en/city-services.json'; // B-N10-05
import type enWebShell from '../messages/en/web-shell.json'; // CB-DIR-11
import type enWebDirectory from '../messages/en/web-directory.json'; // CB-DIR-11
import type enWebBusiness from '../messages/en/web-business.json'; // CB-DIR-12
import type enWebShop from '../messages/en/web-shop.json'; // CB-SHOP-10
import type enCityJoin from '../messages/en/city-join.json'; // CB-DIR-10
import type enHomeLab from '../messages/en/home-lab.json'; // B-N10-04
import type enSearch from '../messages/en/search.json';
import type enPlus from '../messages/en/plus.json';
import type enCompanions from '../messages/en/companions.json';
import type enCompanionHome from '../messages/en/companion-home.json';

// English is the reference locale for key completeness; next-intl uses this
// to type translation keys and ICU params (a wrong key becomes a compile
// error rather than a blank string). Keep namespaces in sync with
// `src/shared/i18n/messages.ts`.
type Messages = {
  common: typeof enCommon;
  contraception: typeof enContraception;
  home: typeof enHome;
  care: typeof enCare;
  checkups: typeof enCheckups;
  children: typeof enChildren;
  fertility: typeof enFertility;
  auth: typeof enAuth;
  banners: typeof enBanners;
  onboarding: typeof enOnboarding;
  phaseDetails: typeof enPhaseDetails;
  pregnancy: typeof enPregnancy;
  pregnancyV2: typeof enPregnancyV2;
  postpartum: typeof enPostpartum;
  pregnancyTools: typeof enPregnancyTools; // B-N5-08 features/pregnancy-tools
  labs: typeof enLabs; // B-N6-07 entities/lab, features/upload-lab, screens/lab-*
  labConsent: typeof enLabConsent; // B-N6-07 ?sheet=lab-consent
  pwa: typeof enPwa;
  healthRecord: typeof enHealthRecord; // B-N6-03 entities/health-record, screens/health-record
  recordExport: typeof enRecordExport; // B-N6-04 screens/record-export
  record: typeof enRecord; // CB-REC-04 record documents (features/record-documents, screens/record-timeline, screens/record-document)
  loss: typeof enLoss;
  conditions: typeof enConditions;
  pelvic: typeof enPelvic;
  me: typeof enMe;
  menopause: typeof enMenopause;
  ivf: typeof enIvf;
  insurance: typeof enInsurance; // CB-INS-03 entities/insurance, screens/insurance
  nav: typeof enNav;
  calendar: typeof enCalendar;
  cycle: typeof enCycle;
  challenge: typeof enChallenge;
  dayTasks: typeof enDayTasks;
  profile: typeof enProfile;
  profileEdit: typeof enProfileEdit;
  profileInfo: typeof enProfileInfo;
  reminders: typeof enReminders;
  notifications: typeof enNotifications;
  account: typeof enAccount;
  analysis: typeof enAnalysis;
  analysisPregnancy: typeof enAnalysisPregnancy;
  analysisPostpartum: typeof enAnalysisPostpartum; // B-N5-07 screens/analysis-postpartum
  babyLog: typeof enBabyLog; // B-N5-07 features/baby-log
  log: typeof enLog;
  logPeriod: typeof enLogPeriod;
  logCustomize: typeof enLogCustomize;
  logSheet: typeof enLogSheet;
  voiceLog: typeof enVoiceLog; // B-N3-05 features/voice-log
  aiCredit: typeof enAiCredit; // entities/ai-credit
  aiGate: typeof enAiGate; // features/ai-gate
  autopilot: typeof enAutopilot; // shared/autopilot (overlay)
  autopilotCatalog: typeof enAutopilotCatalog; // app/autopilot/catalog titles — only where the agent manifest is built
  credits: typeof enCredits; // screens/credits-usage + credits-buy
  alerts: typeof enAlerts; // T-M10-15 entities/alert, screens/alerts, home bell
  vitals: typeof enVitals; // B-N6-02 entities/vital, features/add-vital, screens/vitals-*
  todo: typeof enTodo; // B-N6-08 entities/todo, screens/todo*
  learn: typeof enLearn; // B-N8-03 entities/course, widgets/learn-home-card, screens/learn*
  doctors: typeof enDoctors; // B-N7-05 entities/doctor, screens/doctor*
  assistant: typeof enAssistant; // B-N7-07 entities/assistant, screens/assistant*
  logTaxonomy: typeof enLogTaxonomy;
  teen: typeof enTeen;
  welcome: typeof enWelcome;
  services: typeof enServices;
  shop: typeof enShop; // B-N10-03 entities/shop, screens/shop
  cityServices: typeof enCityServices; // B-N10-05 entities/city-service, screens/city-services
  cityJoin: typeof enCityJoin; // CB-DIR-10 screens/city-services-join
  webShell: typeof enWebShell; // CB-DIR-11 widgets/web-shell
  webDirectory: typeof enWebDirectory; // CB-DIR-11 screens/web-city-services
  webBusiness: typeof enWebBusiness; // CB-DIR-12 screens/web-city-services-business
  webShop: typeof enWebShop; // CB-SHOP-10 screens/web-shop
  homeLab: typeof enHomeLab; // B-N10-04 entities/partner-lab, screens/home-lab, widgets/home-lab-cards
  search: typeof enSearch;
  plus: typeof enPlus;
  companions: typeof enCompanions;
  companionHome: typeof enCompanionHome;
  articles: typeof enArticles;
};

declare global {
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  interface IntlMessages extends Messages {}
}
