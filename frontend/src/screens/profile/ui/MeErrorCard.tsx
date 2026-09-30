'use client';

import { useTranslations } from 'next-intl';

import { IconCircle, SecondaryButton } from '@/shared/ui';

/** Error state of the Me screens: the profile could not be fetched. */
export function MeErrorCard({ onRetry }: { onRetry: () => void }) {
  const t = useTranslations('me');
  return (
    <div className="me-error" role="alert">
      <IconCircle icon="warning" tone="danger" size="md" />
      <div className="me-error-text">
        <p className="me-error-title">{t('error.title')}</p>
        <p className="me-error-body">{t('error.body')}</p>
      </div>
      <SecondaryButton variant="outline" onClick={onRetry} className="me-error-retry">
        {t('error.retry')}
      </SecondaryButton>
    </div>
  );
}
