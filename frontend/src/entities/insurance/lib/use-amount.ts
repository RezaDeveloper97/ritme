'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { Locale } from '@/shared/i18n';
import { formatDecimal, formatNumber } from '@/shared/lib/date';

import { compactAmount } from '../model/display';

/** «۱٫۸ میلیون», «۹۵۰ هزار» for a toman amount, in the locale's digits (copy: `insurance.amount`). */
export function useAmountFormat(): (toman: number) => string {
  const t = useTranslations('insurance.amount');
  const locale = useLocale() as Locale;
  return (toman) => {
    const a = compactAmount(toman);
    return t(a.unit, { value: formatDecimal(a.value, locale) });
  };
}

/** «۱٬۲۶۰٬۰۰۰ تومان» — the exact toman amount with the locale's grouping and digits (claim breakdown). */
export function useExactAmount(): (toman: number) => string {
  const t = useTranslations('insurance.amount');
  const locale = useLocale() as Locale;
  return (toman) => {
    const sep = locale === 'fa' ? '٬' : ',';
    return t('toman', { value: String(Math.trunc(toman)).replace(/\B(?=(\d{3})+(?!\d))/g, ',').split(',').map((g) => formatNumber(g, locale)).join(sep) });
  };
}
