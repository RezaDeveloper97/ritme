'use client';

import { useTranslations } from 'next-intl';

import { cn, toPersianDigits } from '@/shared/lib';

import type { Grant } from '../model/schema';

/** Right-hand status of a person row: days left / unlimited / pending / expired. Never says «registered». */
export function GrantStatusPill({ grant }: { grant: Grant }) {
  const t = useTranslations('grant');
  let text: string;
  if (grant.status === 'pending') text = t('pending');
  else if (grant.status === 'expired') text = t('expired');
  else if (grant.days_left != null) text = t('daysLeft', { count: toPersianDigits(grant.days_left) });
  else text = grant.duration.type === 'unlimited' ? t('unlimited') : t('active');
  return <span className={cn('grant-status', `grant-status--${grant.status}`)}>{text}</span>;
}

/** «۷۲٪» progress, empty while hidden. */
export function progressText(grant: Grant): string | null {
  return grant.progress_percent == null ? null : `${toPersianDigits(grant.progress_percent)}٪`;
}
