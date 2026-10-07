import { getRequestConfig } from 'next-intl/server';

import { DEFAULT_LOCALE, MESSAGES } from './messages';

// next-intl without URL routing: one origin, no locale prefix, Persian.
export default getRequestConfig(async () => ({
  locale: DEFAULT_LOCALE,
  messages: MESSAGES[DEFAULT_LOCALE],
  timeZone: 'Asia/Tehran',
}));
