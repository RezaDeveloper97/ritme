import { getRequestConfig } from 'next-intl/server';
import { cookies } from 'next/headers';

import { UI_BUNDLES, UI_LOCALE_COOKIE, resolveUiLocale } from './ui-locales';

// next-intl without URL routing: the admin is one origin with no locale prefix;
// the UI language is a per-browser cookie (default fa).
export default getRequestConfig(async () => {
  const store = await cookies();
  const locale = resolveUiLocale(store.get(UI_LOCALE_COOKIE)?.value);
  return {
    locale,
    messages: UI_BUNDLES[locale],
    timeZone: 'Asia/Tehran',
  };
});
