'use client';

import { useTranslations } from 'next-intl';

/**
 * Labels for the profile enums in the list (the API sends raw values there;
 * the detail endpoint carries its own `options`). Unknown values show as-is.
 */
export function useUserLabels() {
  const t = useTranslations('users');
  return {
    subscription: (value: string) => {
      const key = `subscriptionTypes.${value}` as 'subscriptionTypes.free';
      return t.has(key) ? t(key) : value;
    },
    goal: (value: string) => {
      const key = `goals.${value}` as 'goals.ttc';
      return t.has(key) ? t(key) : value;
    },
  };
}
