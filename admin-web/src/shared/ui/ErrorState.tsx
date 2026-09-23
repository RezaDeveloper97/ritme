'use client';

import { useTranslations } from 'next-intl';

import { useErrorMessage } from '@/shared/i18n';

import { Button } from './Button';
import { Icon } from './Icon';

/** Inline failure with a retry, for a panel or list that didn't load. */
export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const t = useTranslations('common');
  const describe = useErrorMessage();
  return (
    <div role="alert" className="flex flex-col items-center gap-3 px-4 py-10 text-center text-ink-3">
      <Icon name="alert" size={22} className="text-danger" />
      <p className="m-0">{describe(error)}</p>
      {onRetry ? (
        <Button size="sm" onClick={onRetry}>
          {t('retry')}
        </Button>
      ) : null}
    </div>
  );
}
