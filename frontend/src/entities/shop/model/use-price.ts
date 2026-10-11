'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { Locale } from '@/shared/i18n';

import { formatToman } from './price';

/** «۴۸۵ هزار تومان» — compact toman, copy from the `shop.price.*` messages (CB-SHOP-08 screens). */
export function useShopPrice() {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  return (toman: number) => {
    const { unit, n } = formatToman(toman, locale);
    return t(`price.${unit}`, { n });
  };
}
