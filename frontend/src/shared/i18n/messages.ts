import type { AbstractIntlMessages } from 'next-intl';

import type { Locale } from './routing';

import enAuth from '../../../messages/en/auth.json';
import enCalendar from '../../../messages/en/calendar.json';
import enCommon from '../../../messages/en/common.json';
import enCycle from '../../../messages/en/cycle.json';
import enHome from '../../../messages/en/home.json';
import enLog from '../../../messages/en/log.json';
import enNav from '../../../messages/en/nav.json';
import enOnboarding from '../../../messages/en/onboarding.json';
import enProfile from '../../../messages/en/profile.json';
import faAuth from '../../../messages/fa/auth.json';
import faCalendar from '../../../messages/fa/calendar.json';
import faCommon from '../../../messages/fa/common.json';
import faCycle from '../../../messages/fa/cycle.json';
import faHome from '../../../messages/fa/home.json';
import faLog from '../../../messages/fa/log.json';
import faNav from '../../../messages/fa/nav.json';
import faOnboarding from '../../../messages/fa/onboarding.json';
import faProfile from '../../../messages/fa/profile.json';

const messages = {
  fa: {
    common: faCommon,
    home: faHome,
    auth: faAuth,
    onboarding: faOnboarding,
    nav: faNav,
    calendar: faCalendar,
    cycle: faCycle,
    profile: faProfile,
    log: faLog,
  },
  en: {
    common: enCommon,
    home: enHome,
    auth: enAuth,
    onboarding: enOnboarding,
    nav: enNav,
    calendar: enCalendar,
    cycle: enCycle,
    profile: enProfile,
    log: enLog,
  },
} as const;

export function getLocaleMessages(locale: Locale): AbstractIntlMessages {
  return messages[locale];
}
