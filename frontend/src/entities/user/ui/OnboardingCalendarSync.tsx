'use client';

import { useLocale } from 'next-intl';
import { useEffect } from 'react';

import type { Locale } from '@/shared/i18n';

import { useOnboardingStore } from '../model/store';

/**
 * Keeps the persisted onboarding answers in the calendar the current locale
 * reads. Onboarding dates are stored as calendar *parts*, so switching the
 * language mid-flow would otherwise reinterpret a Jalali 1373 as a Gregorian
 * one. Mounted once in the app layout; the store action is a no-op when the
 * calendar already matches.
 */
export function OnboardingCalendarSync() {
  const locale = useLocale() as Locale;
  const syncCalendar = useOnboardingStore((s) => s.syncCalendar);

  useEffect(() => {
    syncCalendar(locale);
  }, [locale, syncCalendar]);

  return null;
}
