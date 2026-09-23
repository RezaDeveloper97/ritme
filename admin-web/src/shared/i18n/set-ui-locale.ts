'use client';

import { UI_LOCALE_COOKIE, resolveUiLocale } from './ui-locales';

/** Stores the admin UI language (1 year). Caller refreshes the router. */
export function setUiLocaleCookie(locale: string): void {
  const value = resolveUiLocale(locale);
  document.cookie = `${UI_LOCALE_COOKIE}=${value}; Path=/; Max-Age=31536000; SameSite=Lax`;
}
