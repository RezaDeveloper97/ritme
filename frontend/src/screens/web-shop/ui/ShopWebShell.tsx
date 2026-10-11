'use client';

import { useLocale as useIntlLocale, useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { useShopPrice } from '@/entities/shop';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';
import { WebShell } from '@/widgets/web-shell';

import { STORE_HOME } from '../model/store';

const useLocale = () => useIntlLocale() as Locale;

/** The shop's configuration of the generic web shell: shop + directory nav, the app fallback, no CTA. */
export function ShopWebShell({ appHref, children }: { appHref: string; children: ReactNode }) {
  const t = useTranslations('webShell');
  return (
    <WebShell
      nav={[
        { href: STORE_HOME, label: t('nav.shop') },
        { href: '/places', label: t('nav.services') },
      ]}
      appHref={appHref}
      homeHref={STORE_HOME}
    >
      {children}
    </WebShell>
  );
}

/** Rating number in locale digits: 4.7 → «۴٫۷». */
export function useRating() {
  const locale = useLocale();
  return (v: number) => formatNumber(v.toFixed(1).replace('.', locale === 'fa' ? '٫' : '.'), locale);
}

/** Product tile of the web catalog: the whole tile opens `/store/product/[id]`. Shares the app tile's look. */
export function WebTile({
  p,
}: {
  p: { id: number; title: string; image_url: string | null; price: number; compare_at_price: number | null; rating_avg: number | null; rating_count: number; badges: string[]; in_stock: boolean; seller: { name: string } | null };
}) {
  const t = useTranslations('shop');
  const locale = useLocale();
  const price = useShopPrice();
  const rating = useRating();
  return (
    <li className="shp-tile wsh-tile">
      <Link href={`/store/product/${p.id}`} className="shp-tile-link">
        <span className="shp-tile-art wsh-tile-art" aria-hidden>
          {p.image_url ? (
            // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
            <img src={p.image_url} alt="" loading="lazy" />
          ) : (
            <Icon name="bag" size={36} strokeWidth={1.4} />
          )}
          {p.badges[0] ? <span className="shp-tile-badge">{p.badges[0]}</span> : null}
        </span>
        <span className="shp-tile-title">{p.title}</span>
        {p.seller ? <span className="shp-tile-seller">{p.seller.name}</span> : null}
        {p.rating_avg !== null && p.rating_count > 0 ? (
          <span className="shp-tile-rating">
            <Icon name="star" size={12} />
            <bdi>{rating(p.rating_avg)}</bdi>
            <span className="shp-tile-count">({formatNumber(p.rating_count, locale)})</span>
          </span>
        ) : null}
        <span className="shp-tile-prices">
          <span className="shp-price">{price(p.price)}</span>
          {p.compare_at_price && p.compare_at_price > p.price ? <s className="shp-was">{price(p.compare_at_price)}</s> : null}
        </span>
        {!p.in_stock ? <span className="shp-tile-out">{t('home.outOfStock')}</span> : null}
      </Link>
    </li>
  );
}
