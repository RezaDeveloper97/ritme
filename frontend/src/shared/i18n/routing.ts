import { defineRouting } from 'next-intl/routing';

/**
 * The locales Ritme ships. `fa` (Persian) is the default and renders RTL;
 * `en` is the secondary LTR locale. The active locale lives in the URL path
 * (`/fa/...`, `/en/...`) for SEO and shareable links — see CLAUDE.md §6.
 */
export const routing = defineRouting({
  locales: ['fa', 'en'],
  defaultLocale: 'fa',
  // Always keep the locale prefix so URLs are explicit and shareable.
  localePrefix: 'always',
  // Persian is the product default (CLAUDE.md §1). Don't auto-switch to the
  // browser's Accept-Language (which is often `en`) — unprefixed routes like
  // `/` always resolve to `fa`. Users still pick `en` explicitly via the URL
  // or the locale switcher (which sets the NEXT_LOCALE cookie).
  localeDetection: false,
});

export type Locale = (typeof routing.locales)[number];

/** Text direction per locale. Persian is RTL by default (CLAUDE.md §1). */
export const localeDirection = {
  fa: 'rtl',
  en: 'ltr',
} as const satisfies Record<Locale, 'rtl' | 'ltr'>;

export function getDirection(locale: Locale): 'rtl' | 'ltr' {
  return localeDirection[locale];
}

/** Type guard: is `value` one of the supported locales? */
export function isLocale(value: unknown): value is Locale {
  return (
    typeof value === 'string' &&
    (routing.locales as readonly string[]).includes(value)
  );
}
