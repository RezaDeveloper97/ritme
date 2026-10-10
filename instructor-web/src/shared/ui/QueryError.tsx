'use client';

import { useTranslations } from 'next-intl';

import { useErrorMessage } from '@/shared/i18n';

import { Button } from './Button';
import { ErrorState } from './ErrorState';

/** A failed query: «اطلاعات بارگذاری نشد», the translated reason and «تلاش دوباره». */
export function QueryError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useTranslations('common');
  const message = useErrorMessage();
  return (
    <ErrorState
      title={t('loadFailed')}
      message={message(error)}
      action={
        <Button variant="ghost" onClick={onRetry}>
          {t('retry')}
        </Button>
      }
    />
  );
}
