import { getRequestConfig } from 'next-intl/server';

import { getLocaleMessages } from './messages';
import { isLocale, routing } from './routing';

/**
 * Resolves the active locale for each request and supplies its messages to
 * Server Components. Consumed by the next-intl plugin (see `next.config.ts`).
 */
export default getRequestConfig(async ({ requestLocale }) => {
  const requested = await requestLocale;
  const locale = isLocale(requested) ? requested : routing.defaultLocale;

  return {
    locale,
    messages: getLocaleMessages(locale),
    // Iran's timezone keeps date/number formatting consistent server-side.
    timeZone: 'Asia/Tehran',
  };
});
