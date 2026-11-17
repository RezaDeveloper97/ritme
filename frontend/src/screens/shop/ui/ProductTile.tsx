'use client';

import { useLocale, useTranslations } from 'next-intl';

import { formatToman, type ShopCatalogCard } from '@/entities/shop';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import { Icon } from '@/shared/ui';

/** «۴۸۵ هزار تومان» — compact toman, copy from the `shop.price.*` messages. */
export function usePrice() {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  return (toman: number) => {
    const { unit, n } = formatToman(toman, locale);
    return t(`price.${unit}`, { n });
  };
}

/** Rating number in locale digits: 4.7 → «۴٫۷». */
export function useRatingText() {
  const locale = useLocale() as Locale;
  return (v: number) => formatNumber(v.toFixed(1).replace('.', locale === 'fa' ? '٫' : '.'), locale);
}

const SKIN_KEYS = ['oily', 'combination', 'normal', 'dry', 'sensitive'] as const;

/** Skin-type keys are labelled by the client (`shop.skin.*`); unknown keys fall back to the server label. */
export function useSkinLabel() {
  const t = useTranslations('shop');
  return (key: string, fallback: string) => ((SKIN_KEYS as readonly string[]).includes(key) ? t(`skin.${key as (typeof SKIN_KEYS)[number]}`) : fallback);
}

/**
 * Product tile of the catalog (home sections, list grid, bought-together row): the whole tile opens the product page
 * (CB-SHOP-07). Moved here from ShopHomePage (CB-SHOP-06) so home and list share one tile.
 */
export function ProductTile({ p }: { p: ShopCatalogCard }) {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const price = usePrice();
  const ratingText = useRatingText();
  return (
    <li className="shp-tile">
      <Link href={`/shop/product/${p.id}`} className="shp-tile-link" {...pilot(`shop.product:${p.id}`)}>
        <span className="shp-tile-art" aria-hidden>
          {p.image_url ? (
            // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
            <img src={p.image_url} alt="" loading="lazy" />
          ) : (
            <Icon name="bag" size={28} strokeWidth={1.4} />
          )}
          {p.badges[0] ? <span className="shp-tile-badge">{p.badges[0]}</span> : null}
        </span>
        <span className="shp-tile-title">{p.title}</span>
        {p.seller ? <span className="shp-tile-seller">{p.seller.name}</span> : null}
        {p.rating_avg !== null && p.rating_count > 0 ? (
          <span className="shp-tile-rating">
            <Icon name="star" size={12} />
            <bdi>{ratingText(p.rating_avg)}</bdi>
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
