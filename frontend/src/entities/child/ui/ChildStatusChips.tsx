'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import { dueIn, isVisitUrgent } from '../model/format';
import type { Child } from '../model/types';

type T = ReturnType<typeof useTranslations<'children'>>;

/** «۱۸ روز دیگر» / «۲ ماه دیگر» / «امروز» / «عقب افتاده» for a visit's `daysLeft`. */
export function useDueText(): (daysLeft: number, overdueLabel?: string | null) => string {
  const t = useTranslations('children');
  const locale = useLocale() as Locale;
  return (daysLeft, overdueLabel) => dueText(t, locale, daysLeft, overdueLabel);
}

function dueText(t: T, locale: Locale, daysLeft: number, overdueLabel?: string | null): string {
  const d = dueIn(daysLeft);
  switch (d.kind) {
    case 'overdue':
      return overdueLabel || t('due.overdue');
    case 'today':
      return t('due.today');
    case 'days':
      return t('due.days', { n: formatNumber(d.n, locale) });
    case 'months':
      return t('due.months', { n: formatNumber(d.n, locale) });
  }
}

/**
 * The two status pills of a child card (v15_Children): next vaccine visit
 * (warm; danger when overdue) or «واکسن‌ها: کامل», and the growth verdict
 * (success when in band, warm when a value is outside P3–P97).
 */
export function ChildStatusChips({ child, className }: { child: Child; className?: string }) {
  const t = useTranslations('children');
  const locale = useLocale() as Locale;
  const next = child.vaccines.next;
  const growth = child.growth;
  return (
    <span className={clsx('chd-chips', className)}>
      {next ? (
        <span className={clsx('chd-chip', isVisitUrgent(next) ? 'nb-tone-danger' : 'nb-tone-warm')}>
          <Icon name="bellPlain" size={13} strokeWidth={2.4} />
          {t('chips.vaccine', { label: next.label, due: dueText(t, locale, next.daysLeft, next.statusLabel) })}
        </span>
      ) : child.vaccines.complete ? (
        <span className="chd-chip nb-tone-success">
          <Icon name="check" size={13} strokeWidth={2.4} />
          {t('chips.vaccinesComplete')}
        </span>
      ) : null}
      {growth.label ? (
        <span
          className={clsx(
            'chd-chip',
            growth.status === 'normal' ? 'nb-tone-success' : growth.status === 'check' ? 'nb-tone-warm' : 'nb-tone-neutral',
          )}
        >
          {growth.status === 'normal' ? <Icon name="check" size={13} strokeWidth={2.4} /> : null}
          {t('chips.growth', { label: growth.label })}
        </span>
      ) : null}
    </span>
  );
}
