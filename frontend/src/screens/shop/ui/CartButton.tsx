'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useShopCart } from '@/entities/shop';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import { Icon } from '@/shared/ui';

/**
 * Header cart button with the item count (CB-SHOP-08). Hidden for a teen-mode account (403) and while the cart is
 * unknown; the count is in the accessible name, not only the badge.
 */
export function CartButton() {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const q = useShopCart();
  if (q.isError && getApiErrorStatus(q.error) === 403) return <span className="nb-hdr-spacer" aria-hidden />;
  const n = q.data?.items_count ?? 0;
  return (
    <Link
      href="/shop/cart"
      className="nb-hbtn shp-cartbtn"
      {...pilot('shop.cart')}
      aria-label={n > 0 ? `${t('cartLink')} · ${formatNumber(n, locale)}` : t('cartLink')}
    >
      <Icon name="bag" size={20} strokeWidth={1.8} />
      {n > 0 ? (
        <span className="shp-cartbtn-n" aria-hidden>
          {formatNumber(n, locale)}
        </span>
      ) : null}
    </Link>
  );
}
