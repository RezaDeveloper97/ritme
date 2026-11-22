'use client';

import { useLocale, useTranslations } from 'next-intl';

import { ageRangeLabel, type CityServiceCard, type CityServiceCatalogOption, slotWhen } from '@/entities/city-service';
import { Link, type Locale, useDirection } from '@/shared/i18n';
import { formatNumber, formatTimeOfDay, formatWeekday } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import { Icon } from '@/shared/ui';
import '@/shared/ui/styles/csv.css';
import '@/shared/ui/styles/cdh.css';
import './dfc.css';

interface Props {
  place: CityServiceCard;
  catalog: CityServiceCatalogOption[];
  /** Compact = the map's bottom card (no thumbnail row chevron, fewer lines). */
  compact?: boolean;
  /** CB-DIR-11: where a card leads — the app's place page by default, `/places` on the public desktop pages. */
  hrefBase?: string;
}

/** Hooks for the card's facts — shared by the list row and the map's bottom card. */
export function usePlaceFacts(place: CityServiceCard) {
  const t = useTranslations('cityServices');
  const locale = useLocale() as Locale;
  const slot = slotWhen(place.next_slot_at, new Date());
  const slotText = slot
    ? `${slot.day === 'date' ? formatWeekday(slot.date, locale) : t(`card.${slot.day}`)} ${formatTimeOfDay(slot.date, locale)}`
    : null;
  const price =
    place.price_from_toman !== null
      ? t('card.priceFrom', { amount: formatNumber(Math.max(1, Math.round(place.price_from_toman / 1000)), locale) })
      : (place.price_text ?? null);
  const distance = place.distance_km !== null ? t('home.km', { km: formatNumber(place.distance_km, locale) }) : null;
  return { slotText, price, distance, locale };
}

/** One result row (nbl_Dir_List): verified badge, rating, price-from and the next free slot (or «open now»). */
export function DirectoryCard({ place: l, catalog, compact = false, hrefBase = '/city-services' }: Props) {
  const t = useTranslations('cityServices');
  const rtl = useDirection() === 'rtl';
  const { slotText, price, distance, locale } = usePlaceFacts(l);
  const ages = ageRangeLabel(l.age_ranges, catalog);
  const where = [l.district ?? l.city.name, distance].filter(Boolean).join(' · ');
  const status = l.open_now === true ? t('home.openNow') : slotText;

  return (
    <Link href={`${hrefBase}/${l.id}`} className="csv-card dfc" {...pilot(`city-list.listing:${l.id}`)}>
      {compact ? null : (
        <span className="csv-thumb" aria-hidden>
          {l.photo_url ? (
            // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
            <img src={l.photo_url} alt="" loading="lazy" />
          ) : (
            <Icon name={l.category.icon} size={28} strokeWidth={1.4} />
          )}
        </span>
      )}
      <span className="csv-card-main">
        {l.verified ? (
          <span className="dfc-verified">
            <Icon name="check" size={12} />
            {t('home.verified')}
          </span>
        ) : null}
        <span className="dfc-title-row">
          <span className="csv-card-title">{l.name}</span>
          {l.rating_avg !== null ? (
            <span className="cdh-rating">
              <Icon name="star" size={12} />
              {formatNumber(l.rating_avg, locale)}
              {l.rating_count > 0 ? <span className="dfc-count">({formatNumber(l.rating_count, locale)})</span> : null}
            </span>
          ) : null}
        </span>
        <span className="csv-card-meta">
          <span className="csv-badge">{l.category.name}</span>
          <span>{where}</span>
        </span>
        {compact ? null : ages ? <span className="cdh-ages">{t('home.suitable', { ages })}</span> : null}
        <span className="csv-card-meta dfc-foot">
          {price ? <span className="csv-price">{price}</span> : <span />}
          {status ? <span className={l.open_now === true ? 'cdh-open' : 'dfc-slot'}>{status}</span> : null}
        </span>
      </span>
      {compact ? null : <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="csv-go" />}
    </Link>
  );
}
