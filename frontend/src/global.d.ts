import type enAccount from '../messages/en/account.json';
import type enAuth from '../messages/en/auth.json';
import type enBanners from '../messages/en/banners.json';
import type enCalendar from '../messages/en/calendar.json';
import type enCommon from '../messages/en/common.json';
import type enCycle from '../messages/en/cycle.json';
import type enHome from '../messages/en/home.json';
import type enLog from '../messages/en/log.json';
import type enLogPeriod from '../messages/en/log-period.json';
import type enNav from '../messages/en/nav.json';
import type enNotifications from '../messages/en/notifications.json';
import type enOnboarding from '../messages/en/onboarding.json';
import type enPregnancy from '../messages/en/pregnancy.json';
import type enProfile from '../messages/en/profile.json';
import type enProfileEdit from '../messages/en/profile-edit.json';
import type enProfileInfo from '../messages/en/profile-info.json';
import type enReminders from '../messages/en/reminders.json';

// English is the reference locale for key completeness; next-intl uses this
// to type translation keys and ICU params (a wrong key becomes a compile
// error rather than a blank string). Keep namespaces in sync with
// `src/shared/i18n/messages.ts`.
type Messages = {
  common: typeof enCommon;
  home: typeof enHome;
  auth: typeof enAuth;
  banners: typeof enBanners;
  onboarding: typeof enOnboarding;
  pregnancy: typeof enPregnancy;
  nav: typeof enNav;
  calendar: typeof enCalendar;
  cycle: typeof enCycle;
  profile: typeof enProfile;
  profileEdit: typeof enProfileEdit;
  profileInfo: typeof enProfileInfo;
  reminders: typeof enReminders;
  notifications: typeof enNotifications;
  account: typeof enAccount;
  log: typeof enLog;
  logPeriod: typeof enLogPeriod;
};

declare global {
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  interface IntlMessages extends Messages {}
}
