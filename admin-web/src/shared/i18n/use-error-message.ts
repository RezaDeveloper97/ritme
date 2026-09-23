'use client';

import { useTranslations } from 'next-intl';
import { useCallback } from 'react';

import { isApiError } from '@/shared/api';

/**
 * Turns any thrown value into a translated sentence. The API's English
 * `message` is for logs; the UI speaks `error_code` (admin-api.md §2).
 */
export function useErrorMessage(): (error: unknown) => string {
  const t = useTranslations('errors');
  return useCallback(
    (error: unknown) => {
      if (!isApiError(error)) return t('unknown');
      if (error.code === 'too_many_attempts') {
        return t('too_many_attempts', { seconds: error.retryAfter ?? 0 });
      }
      const code = error.code as 'unknown';
      return t.has(code) ? t(code) : t('unknown');
    },
    [t],
  );
}
