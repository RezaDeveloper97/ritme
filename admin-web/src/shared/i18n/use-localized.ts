'use client';

import { useLocale } from 'next-intl';
import { useCallback } from 'react';

import { pickTranslation } from '@/shared/lib';

import { useContentLanguages } from './content-languages';

/**
 * `localize(row.title)` → the text for lists and headings: the UI locale's
 * translation, else the default content language's, else any.
 */
export function useLocalized(): (value: Record<string, string> | null | undefined) => string {
  const locale = useLocale();
  const defaultCode = useContentLanguages().data?.default;
  return useCallback((value) => pickTranslation(value, [locale, defaultCode]), [locale, defaultCode]);
}
