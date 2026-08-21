import { getRequestConfig } from 'next-intl/server';

import { getLocaleMessages } from './messages';
import { resolveLocale } from './registry';

/**
 * Resolves the active locale for each request and supplies its messages to
 * Server Components. Consumed by the next-intl plugin (see `next.config.ts`).
 *
 * The locale is validated against the live language registry, so a language an
 * admin added after the last deploy renders here without a rebuild.
 */
export default getRequestConfig(async ({ requestLocale }) => {
  const locale = await resolveLocale(await requestLocale);

  return {
    locale,
    messages: await getLocaleMessages(locale),
    // Iran's timezone keeps date/number formatting consistent server-side.
    timeZone: 'Asia/Tehran',
  };
});
