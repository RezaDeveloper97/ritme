import type { UiMessages } from './ui-locales';

// Typed message keys for useTranslations/getTranslations.
declare module 'next-intl' {
  interface AppConfig {
    Messages: UiMessages;
  }
}
