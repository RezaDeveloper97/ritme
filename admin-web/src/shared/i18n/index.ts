// Public API of shared/i18n (client-safe). The next-intl request config lives in
// ./request.ts and is loaded by next.config only.
export {
  DEFAULT_UI_LOCALE,
  UI_LOCALE_COOKIE,
  resolveUiLocale,
  uiDirection,
  uiLocaleOptions,
} from './ui-locales';
export type { Direction, UiMessages } from './ui-locales';
export { useContentLanguages, contentLanguageKeys, contentLanguageSchema } from './content-languages';
export type { ContentLanguage } from './content-languages';
export { setUiLocaleCookie } from './set-ui-locale';
export { useErrorMessage } from './use-error-message';
export { useLocalized } from './use-localized';
