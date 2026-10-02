import type { AbstractIntlMessages } from 'next-intl';

import enAccount from '../../../messages/en/account.json';
import enAnalysis from '../../../messages/en/analysis.json';
import enAnalysisPregnancy from '../../../messages/en/analysis-pregnancy.json';
import enArticles from '../../../messages/en/articles.json';
import enAuth from '../../../messages/en/auth.json';
import enBanners from '../../../messages/en/banners.json';
import enCalendar from '../../../messages/en/calendar.json';
import enCare from '../../../messages/en/care.json';
import enChallenge from '../../../messages/en/challenge.json';
import enCheckups from '../../../messages/en/checkups.json';
import enCommon from '../../../messages/en/common.json';
import enContraception from '../../../messages/en/contraception.json';
import enCycle from '../../../messages/en/cycle.json';
import enDayTasks from '../../../messages/en/day-tasks.json';
import enFertility from '../../../messages/en/fertility.json';
import enHome from '../../../messages/en/home.json';
import enLog from '../../../messages/en/log.json';
import enLogPeriod from '../../../messages/en/log-period.json';
import enLogCustomize from '../../../messages/en/log-customize.json';
import enLogSheet from '../../../messages/en/log-sheet.json';
import enVoiceLog from '../../../messages/en/voice-log.json';
import enLogTaxonomy from '../../../messages/en/log-taxonomy.json';
import enMe from '../../../messages/en/me.json';
import enMenopause from '../../../messages/en/menopause.json';
import enNav from '../../../messages/en/nav.json';
import enNotifications from '../../../messages/en/notifications.json';
import enOnboarding from '../../../messages/en/onboarding.json';
import enPhaseDetails from '../../../messages/en/phase-details.json';
import enPregnancy from '../../../messages/en/pregnancy.json';
import enPregnancyV2 from '../../../messages/en/pregnancy-v2.json';
import enProfile from '../../../messages/en/profile.json';
import enProfileEdit from '../../../messages/en/profile-edit.json';
import enProfileInfo from '../../../messages/en/profile-info.json';
import enPwa from '../../../messages/en/pwa.json';
import enReminders from '../../../messages/en/reminders.json';
import enWelcome from '../../../messages/en/welcome.json';
import enServices from '../../../messages/en/services.json';
import enSearch from '../../../messages/en/search.json';
import enPlus from '../../../messages/en/plus.json';
import faAccount from '../../../messages/fa/account.json';
import faAnalysis from '../../../messages/fa/analysis.json';
import faAnalysisPregnancy from '../../../messages/fa/analysis-pregnancy.json';
import faArticles from '../../../messages/fa/articles.json';
import faAuth from '../../../messages/fa/auth.json';
import faBanners from '../../../messages/fa/banners.json';
import faCalendar from '../../../messages/fa/calendar.json';
import faCare from '../../../messages/fa/care.json';
import faChallenge from '../../../messages/fa/challenge.json';
import faCheckups from '../../../messages/fa/checkups.json';
import faCommon from '../../../messages/fa/common.json';
import faContraception from '../../../messages/fa/contraception.json';
import faCycle from '../../../messages/fa/cycle.json';
import faDayTasks from '../../../messages/fa/day-tasks.json';
import faFertility from '../../../messages/fa/fertility.json';
import faHome from '../../../messages/fa/home.json';
import faLog from '../../../messages/fa/log.json';
import faLogPeriod from '../../../messages/fa/log-period.json';
import faLogCustomize from '../../../messages/fa/log-customize.json';
import faLogSheet from '../../../messages/fa/log-sheet.json';
import faVoiceLog from '../../../messages/fa/voice-log.json';
import faLogTaxonomy from '../../../messages/fa/log-taxonomy.json';
import faMe from '../../../messages/fa/me.json';
import faMenopause from '../../../messages/fa/menopause.json';
import faNav from '../../../messages/fa/nav.json';
import faNotifications from '../../../messages/fa/notifications.json';
import faOnboarding from '../../../messages/fa/onboarding.json';
import faPhaseDetails from '../../../messages/fa/phase-details.json';
import faPregnancy from '../../../messages/fa/pregnancy.json';
import faPregnancyV2 from '../../../messages/fa/pregnancy-v2.json';
import faProfile from '../../../messages/fa/profile.json';
import faProfileEdit from '../../../messages/fa/profile-edit.json';
import faProfileInfo from '../../../messages/fa/profile-info.json';
import faPwa from '../../../messages/fa/pwa.json';
import faReminders from '../../../messages/fa/reminders.json';
import faWelcome from '../../../messages/fa/welcome.json';
import faServices from '../../../messages/fa/services.json';
import faSearch from '../../../messages/fa/search.json';
import faPlus from '../../../messages/fa/plus.json';

/**
 * The locales compiled into the bundle, and their strings.
 *
 * The app can ship any number of languages — the live list comes from the
 * backend at runtime (`shared/i18n/registry`). These two are the *floor*: they
 * are baked in so the app renders correctly before the first API call, during
 * a backend outage, and while pre-rendering static pages at build time.
 *
 * `frontend/messages/**` stays the source of truth for these strings. After
 * editing them, run `php artisan translations:import` in the backend so a
 * language created later inherits the new keys instead of falling back.
 */
export const BUNDLED_LOCALES = ['fa', 'en'] as const;

export type BundledLocale = (typeof BUNDLED_LOCALES)[number];

/** Persian is the product default and renders RTL (CLAUDE.md §1). */
export const DEFAULT_LOCALE: BundledLocale = 'fa';

export const BUNDLED_DIRECTIONS = {
  fa: 'rtl',
  en: 'ltr',
} as const satisfies Record<BundledLocale, 'rtl' | 'ltr'>;

/** Endonyms for the fallback picker, used only when the API is unreachable. */
export const BUNDLED_NAMES = {
  fa: 'فارسی',
  en: 'English',
} as const satisfies Record<BundledLocale, string>;

export function isBundledLocale(value: unknown): value is BundledLocale {
  return (
    typeof value === 'string' &&
    (BUNDLED_LOCALES as readonly string[]).includes(value)
  );
}

const bundled = {
  fa: {
    common: faCommon,
    contraception: faContraception,
    home: faHome,
    auth: faAuth,
    banners: faBanners,
    onboarding: faOnboarding,
    phaseDetails: faPhaseDetails,
    pregnancy: faPregnancy,
    pregnancyV2: faPregnancyV2,
    me: faMe,
    menopause: faMenopause,
    nav: faNav,
    calendar: faCalendar,
    cycle: faCycle,
    challenge: faChallenge,
    dayTasks: faDayTasks,
    profile: faProfile,
    profileEdit: faProfileEdit,
    profileInfo: faProfileInfo,
    reminders: faReminders,
    care: faCare,
    checkups: faCheckups,
    fertility: faFertility,
    notifications: faNotifications,
    account: faAccount,
    analysis: faAnalysis,
    analysisPregnancy: faAnalysisPregnancy,
    log: faLog,
    logPeriod: faLogPeriod,
    logCustomize: faLogCustomize,
    logSheet: faLogSheet,
    voiceLog: faVoiceLog,
    logTaxonomy: faLogTaxonomy,
    welcome: faWelcome,
    services: faServices,
    search: faSearch,
    plus: faPlus,
    pwa: faPwa,
    articles: faArticles,
  },
  en: {
    common: enCommon,
    contraception: enContraception,
    home: enHome,
    auth: enAuth,
    banners: enBanners,
    onboarding: enOnboarding,
    phaseDetails: enPhaseDetails,
    pregnancy: enPregnancy,
    pregnancyV2: enPregnancyV2,
    me: enMe,
    menopause: enMenopause,
    nav: enNav,
    calendar: enCalendar,
    cycle: enCycle,
    challenge: enChallenge,
    dayTasks: enDayTasks,
    profile: enProfile,
    profileEdit: enProfileEdit,
    profileInfo: enProfileInfo,
    reminders: enReminders,
    care: enCare,
    checkups: enCheckups,
    fertility: enFertility,
    notifications: enNotifications,
    account: enAccount,
    analysis: enAnalysis,
    analysisPregnancy: enAnalysisPregnancy,
    log: enLog,
    logPeriod: enLogPeriod,
    logCustomize: enLogCustomize,
    logSheet: enLogSheet,
    voiceLog: enVoiceLog,
    logTaxonomy: enLogTaxonomy,
    welcome: enWelcome,
    services: enServices,
    search: enSearch,
    plus: enPlus,
    pwa: enPwa,
    articles: enArticles,
  },
} as const;

/**
 * Compile-time messages for a bundled locale.
 *
 * profileInfo holds arrays of sections (consumed via `t.raw`), which next-intl
 * supports at runtime but AbstractIntlMessages' index signature can't model.
 */
export function getBundledMessages(locale: BundledLocale): AbstractIntlMessages {
  return bundled[locale] as unknown as AbstractIntlMessages;
}
