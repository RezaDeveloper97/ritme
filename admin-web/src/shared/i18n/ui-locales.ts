import en from '../../../messages/en.json';
import fa from '../../../messages/fa.json';

/**
 * Admin UI (chrome) languages. These are the translations shipped WITH the app
 * — adding one means adding `messages/<code>.json` and one line here, like the
 * user app's bundled floor. They are NOT the content languages: those are data
 * (GET /api/v1/languages, see content-languages.ts) and must never be listed in
 * code. Direction and name come from each bundle's `meta`, never from the code.
 */
export type UiMessages = typeof fa;

export const UI_BUNDLES: Readonly<Record<string, UiMessages>> = { fa, en };

export const DEFAULT_UI_LOCALE = 'fa';
export const UI_LOCALE_COOKIE = 'ritme_admin_locale';

export type Direction = 'rtl' | 'ltr';

export function resolveUiLocale(value: string | null | undefined): string {
  return value && Object.hasOwn(UI_BUNDLES, value) ? value : DEFAULT_UI_LOCALE;
}

export function uiDirection(locale: string): Direction {
  return UI_BUNDLES[resolveUiLocale(locale)]?.meta.direction === 'ltr' ? 'ltr' : 'rtl';
}

export function uiLocaleOptions(): { code: string; name: string }[] {
  return Object.entries(UI_BUNDLES).map(([code, bundle]) => ({ code, name: bundle.meta.name }));
}
