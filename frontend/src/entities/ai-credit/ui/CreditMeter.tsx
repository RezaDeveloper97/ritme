'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { Locale } from '@/shared/i18n';
import { formatDayMonth, fromApiDate } from '@/shared/lib/date';
import { ProgressBar } from '@/shared/ui';

import { formatCredits } from '../lib/format';
import type { AiCredits } from '../model/types';

/**
 * The period's credits as a labelled bar: «۸۸ از ۱۰۰» + when they renew. Amber, the credits accent
 * (CLAUDE.md §10.2). Shared by the assistant hub, the profile card and the usage screen.
 */
export function CreditMeter({ credits, className }: { credits: AiCredits; className?: string }) {
  const t = useTranslations('aiCredit');
  const locale = useLocale() as Locale;
  const ends = credits.periodEndsAt ? formatDayMonth(fromApiDate(credits.periodEndsAt.slice(0, 10)), locale) : null;
  const remaining = formatCredits(credits.plan.remaining, locale);
  const total = formatCredits(credits.plan.total, locale);
  return (
    <ProgressBar
      className={className}
      tone="warm"
      value={credits.plan.remaining}
      max={credits.plan.total}
      label={t('meter.label')}
      valueLabel={t('meter.of', { remaining, total })}
      valueText={t('meter.spoken', { remaining, total })}
      hint={ends ? t('meter.renews', { date: ends }) : undefined}
    />
  );
}
