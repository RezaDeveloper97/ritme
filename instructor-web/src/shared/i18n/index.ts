// Public API of shared/i18n (client-safe). The next-intl request config lives in
// ./request.ts and is loaded by next.config only.
export { DEFAULT_LOCALE, DIRECTION } from './messages';
export type { Messages } from './messages';
export { useErrorMessage } from './use-error-message';
