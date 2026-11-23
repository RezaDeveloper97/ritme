'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { Card, IconCircle, PrimaryButton, SecondaryButton } from '@/shared/ui';
import { PlusBadge } from '@/shared/ui/plus-gate';

import { usePlusStatus } from '../api/queries';
import { detailLocked, rangeLocked } from '../lib/gates';
import '@/shared/ui/styles/an.css';

/** Whether the 1-year / all range is locked now — skip the request when true. */
export function usePlusRangeLocked(range: string): boolean {
  return rangeLocked(usePlusStatus().data, range);
}

/** A-11: whether the detail surfaces (why-this-date, BBT coverline/confirmation, LH explanation) are locked. */
export function usePlusDetailLocked(): boolean {
  return detailLocked(usePlusStatus().data);
}

/** The paywall card shown in place of a 1-year / all analysis (not an error). */
export function PlusRangePaywall({ className }: { className?: string }) {
  const t = useTranslations('plus.limits.range');
  const router = useRouter();
  return (
    <Card className={className} role="status">
      <IconCircle icon="crown" tone="warm" size="md" />
      <h2 className="an-finding-title">{t('title')}</h2>
      <p className="an-card-note">{t('body')}</p>
      <PrimaryButton block onClick={() => router.push('/plus')}>
        {t('cta')}
      </PrimaryButton>
    </Card>
  );
}

/** A compact Plus lock for a detail surface (A-11); its button opens the paywall. */
export function PlusDetailLock({ text, className }: { text: string; className?: string }) {
  const t = useTranslations('plus');
  const router = useRouter();
  return (
    <Card className={className} role="status">
      <PlusBadge label={t('gate.label')} />
      <p className="an-card-note">{text}</p>
      <SecondaryButton icon="crown" block={false} onClick={() => router.push('/plus')}>
        {t('limits.detail.cta')}
      </SecondaryButton>
    </Card>
  );
}
