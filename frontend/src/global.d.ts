import type enAccount from '../messages/en/account.json';
import type enAnalysis from '../messages/en/analysis.json';
import type enAnalysisPregnancy from '../messages/en/analysis-pregnancy.json';
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
import type enFertility from '../messages/en/fertility.json';
import type enLog from '../messages/en/log.json';
import type enLogPeriod from '../messages/en/log-period.json';
import type enLogCustomize from '../messages/en/log-customize.json';
import type enLogSheet from '../messages/en/log-sheet.json';
import type enVoiceLog from '../messages/en/voice-log.json';
import type enLogTaxonomy from '../messages/en/log-taxonomy.json';
import type enMe from '../messages/en/me.json';
import type enIvf from '../messages/en/ivf.json'; // CB-IVF-02
import type enMenopause from '../messages/en/menopause.json';
import type enNav from '../messages/en/nav.json';
import type enNotifications from '../messages/en/notifications.json';
import type enOnboarding from '../messages/en/onboarding.json';
import type enPhaseDetails from '../messages/en/phase-details.json';
import type enPregnancy from '../messages/en/pregnancy.json';
import type enPregnancyV2 from '../messages/en/pregnancy-v2.json';
import type enPostpartum from '../messages/en/postpartum.json';
import type enPwa from '../messages/en/pwa.json';
import type enProfile from '../messages/en/profile.json';
import type enProfileEdit from '../messages/en/profile-edit.json';
import type enProfileInfo from '../messages/en/profile-info.json';
import type enReminders from '../messages/en/reminders.json';
import type enTeen from '../messages/en/teen.json'; // CB-TEEN-02
import type enWelcome from '../messages/en/welcome.json';
import type enServices from '../messages/en/services.json';
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
  fertility: typeof enFertility;
  auth: typeof enAuth;
  banners: typeof enBanners;
  onboarding: typeof enOnboarding;
  phaseDetails: typeof enPhaseDetails;
  pregnancy: typeof enPregnancy;
  pregnancyV2: typeof enPregnancyV2;
  postpartum: typeof enPostpartum;
  pwa: typeof enPwa;
  me: typeof enMe;
  menopause: typeof enMenopause;
  ivf: typeof enIvf;
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
  log: typeof enLog;
  logPeriod: typeof enLogPeriod;
  logCustomize: typeof enLogCustomize;
  logSheet: typeof enLogSheet;
  voiceLog: typeof enVoiceLog; // B-N3-05 features/voice-log
  logTaxonomy: typeof enLogTaxonomy;
  teen: typeof enTeen;
  welcome: typeof enWelcome;
  services: typeof enServices;
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
