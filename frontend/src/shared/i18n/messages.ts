import type { AbstractIntlMessages } from 'next-intl';

import type { Locale } from './routing';

import enAccount from '../../../messages/en/account.json';
import enArticles from '../../../messages/en/articles.json';
import enAuth from '../../../messages/en/auth.json';
import enBanners from '../../../messages/en/banners.json';
import enCalendar from '../../../messages/en/calendar.json';
import enChallenge from '../../../messages/en/challenge.json';
import enCommon from '../../../messages/en/common.json';
import enCycle from '../../../messages/en/cycle.json';
import enDayTasks from '../../../messages/en/day-tasks.json';
import enHome from '../../../messages/en/home.json';
import enLog from '../../../messages/en/log.json';
import enLogPeriod from '../../../messages/en/log-period.json';
import enNav from '../../../messages/en/nav.json';
import enNotifications from '../../../messages/en/notifications.json';
import enOnboarding from '../../../messages/en/onboarding.json';
import enPhaseDetails from '../../../messages/en/phase-details.json';
import enPregnancy from '../../../messages/en/pregnancy.json';
import enPwa from '../../../messages/en/pwa.json';
import enProfile from '../../../messages/en/profile.json';
import enProfileEdit from '../../../messages/en/profile-edit.json';
import enProfileInfo from '../../../messages/en/profile-info.json';
import enReminders from '../../../messages/en/reminders.json';
import enWelcome from '../../../messages/en/welcome.json';
import faAccount from '../../../messages/fa/account.json';
import faArticles from '../../../messages/fa/articles.json';
import faAuth from '../../../messages/fa/auth.json';
import faBanners from '../../../messages/fa/banners.json';
import faCalendar from '../../../messages/fa/calendar.json';
import faChallenge from '../../../messages/fa/challenge.json';
import faCommon from '../../../messages/fa/common.json';
import faCycle from '../../../messages/fa/cycle.json';
import faDayTasks from '../../../messages/fa/day-tasks.json';
import faHome from '../../../messages/fa/home.json';
import faLog from '../../../messages/fa/log.json';
import faLogPeriod from '../../../messages/fa/log-period.json';
import faNav from '../../../messages/fa/nav.json';
import faNotifications from '../../../messages/fa/notifications.json';
import faOnboarding from '../../../messages/fa/onboarding.json';
import faPhaseDetails from '../../../messages/fa/phase-details.json';
import faPregnancy from '../../../messages/fa/pregnancy.json';
import faPwa from '../../../messages/fa/pwa.json';
import faProfile from '../../../messages/fa/profile.json';
import faProfileEdit from '../../../messages/fa/profile-edit.json';
import faProfileInfo from '../../../messages/fa/profile-info.json';
import faReminders from '../../../messages/fa/reminders.json';
import faWelcome from '../../../messages/fa/welcome.json';

const messages = {
  fa: {
    common: faCommon,
    home: faHome,
    auth: faAuth,
    banners: faBanners,
    onboarding: faOnboarding,
    phaseDetails: faPhaseDetails,
    pregnancy: faPregnancy,
    nav: faNav,
    calendar: faCalendar,
    cycle: faCycle,
    challenge: faChallenge,
    dayTasks: faDayTasks,
    profile: faProfile,
    profileEdit: faProfileEdit,
    profileInfo: faProfileInfo,
    reminders: faReminders,
    notifications: faNotifications,
    account: faAccount,
    log: faLog,
    logPeriod: faLogPeriod,
    welcome: faWelcome,
    pwa: faPwa,
    articles: faArticles,
  },
  en: {
    common: enCommon,
    home: enHome,
    auth: enAuth,
    banners: enBanners,
    onboarding: enOnboarding,
    phaseDetails: enPhaseDetails,
    pregnancy: enPregnancy,
    nav: enNav,
    calendar: enCalendar,
    cycle: enCycle,
    challenge: enChallenge,
    dayTasks: enDayTasks,
    profile: enProfile,
    profileEdit: enProfileEdit,
    profileInfo: enProfileInfo,
    reminders: enReminders,
    notifications: enNotifications,
    account: enAccount,
    log: enLog,
    logPeriod: enLogPeriod,
    welcome: enWelcome,
    pwa: enPwa,
    articles: enArticles,
  },
} as const;

export function getLocaleMessages(locale: Locale): AbstractIntlMessages {
  // profileInfo holds arrays of sections (consumed via t.raw), which next-intl
  // supports at runtime but AbstractIntlMessages' index signature can't model.
  return messages[locale] as unknown as AbstractIntlMessages;
}
