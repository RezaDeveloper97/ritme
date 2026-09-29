'use client';

import { useLocale } from 'next-intl';
import { useCallback } from 'react';

import { formatNumber } from './number';

/**
 * `n(value)` in the UI locale's digits. ICU simple arguments (`{count}`) print numbers as plain
 * `String()` — Latin digits even in fa — so every number passed to a message goes through this
 * (only `{x, plural, …}`'s `#` is localized by ICU itself).
 */
export function useNumber(): (value: number) => string {
  const locale = useLocale();
  return useCallback((value: number) => formatNumber(value, locale), [locale]);
}
