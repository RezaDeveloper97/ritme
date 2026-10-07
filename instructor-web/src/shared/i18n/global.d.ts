import type { Messages } from './messages';

// Typed message keys for useTranslations/getTranslations.
declare module 'next-intl' {
  interface AppConfig {
    Messages: Messages;
  }
}
