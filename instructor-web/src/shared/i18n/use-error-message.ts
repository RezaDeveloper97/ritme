'use client';

import { useTranslations } from 'next-intl';
import { useCallback } from 'react';

import { isApiError } from '@/shared/api';

/**
 * One translated sentence for any failure. Codes first (instructor_*), then the
 * HTTP status; the server's English `message` is never shown.
 */
export function useErrorMessage(): (error: unknown) => string {
  const t = useTranslations('errors');
  return useCallback(
    (error: unknown) => {
      if (!isApiError(error)) return t('generic');
      switch (error.code) {
        case 'network_error':
          return t('network');
        case 'instructor_required':
          return t('instructorRequired');
        case 'instructor_pending':
          return t('instructorPending');
        case 'validation_failed':
          return t('validation');
      }
      if (error.status === 429) return t('tooMany');
      if (error.status >= 500) return t('server');
      return t('generic');
    },
    [t],
  );
}
