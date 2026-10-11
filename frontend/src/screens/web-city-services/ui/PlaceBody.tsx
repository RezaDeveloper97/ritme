'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { CityServiceDetail } from '@/entities/city-service';
import type { Locale } from '@/shared/i18n';
import { formatDecimal, formatNumber } from '@/shared/lib/date';
import { Card, Icon } from '@/shared/ui';

import { monthsLabelParts, priceParts } from '../model/place';

/* CB-DIR-11 — the read-only sections of the desktop place page (the booking panel is in WebPlacePage). Prices are the
   venue's own declarations, display only. */

const AMENITY_KEYS = {
  nursing_room: 'place.amenity.nursing_room',
  changing_table: 'place.amenity.changing_table',
  female_instructor: 'place.amenity.female_instructor',
  stroller_parking: 'place.amenity.stroller_parking',
  parking: 'place.amenity.parking',
  elevator: 'place.amenity.elevator',
  wheelchair_access: 'place.amenity.wheelchair_access',
} as const;
const DAY_KEYS = ['place.days.d0', 'place.days.d1', 'place.days.d2', 'place.days.d3', 'place.days.d4', 'place.days.d5', 'place.days.d6'] as const;

/** «۳۲۰ هزار تومان» / «۲٫۴ میلیون تومان». */
export function usePrice(): (toman: number) => string {
  const t = useTranslations('cityServices');
  const locale = useLocale() as Locale;
  return (toman) => {
    const p = priceParts(toman);
    return t(p.unit === 'million' ? 'place.priceM' : 'place.priceK', { n: formatDecimal(p.value, locale) });
  };
}

export function PlaceBody({ l }: { l: CityServiceDetail }) {
  const t = useTranslations('cityServices');
  const locale = useLocale() as Locale;
  const price = usePrice();
  const num = (v: number | string) => formatNumber(v, locale);
  const dec = (v: number) => formatDecimal(v, locale);
  const about = l.about ?? l.description;
  const amenityName = (code: string) => (code in AMENITY_KEYS ? t(AMENITY_KEYS[code as keyof typeof AMENITY_KEYS]) : code.replace(/_/g, ' '));
  const ageLabel = (min: number | null, max: number | null): string | null => {
    const part = (m: number) => {
      const p = monthsLabelParts(m);
      return t(p.unit === 'year' ? 'place.years' : 'place.months', { n: num(p.n) });
    };
    if (min !== null && max !== null) return t('place.ageBetween', { from: part(min), to: part(max) });
    if (min !== null) return t('place.ageFrom', { from: part(min) });
    if (max !== null) return t('place.ageTo', { to: part(max) });
    return null;
  };
  const hoursByDay = new Map(l.weekly_hours.map((h) => [h.day, h]));
  const cancel = l.cancel_policy ?? (l.cancel_free_hours !== null ? t('place.cancelFree', { n: num(l.cancel_free_hours) }) : null);
  const address = [l.city.name, l.district, l.address].filter(Boolean).join('، ');
  const r = l.reviews;
  const scores = [
    ['cleanliness', r.scores.cleanliness],
    ['instructor', r.scores.instructor],
    ['amenities', r.scores.amenities],
  ] as const;

  return (
    <>
      {about || l.amenities.length > 0 ? (
        <Card>
          <h2 className="csv-sub">{t('detail.about')}</h2>
          {about ? <p className="csv-text">{about}</p> : null}
          {l.amenities.length > 0 ? (
            <ul className="cdp-chips">
              {l.amenities.map((a) => (
                <li key={a} className="cdp-chip">
                  {amenityName(a)}
                </li>
              ))}
            </ul>
          ) : null}
        </Card>
      ) : null}

      {l.services.length > 0 ? (
        <Card>
          <h2 className="csv-sub">{t('place.servicesTitle')}</h2>
          <p className="cdp-hint">{t('place.servicesNote')}</p>
          <ul className="cdp-svc-list">
            {l.services.map((s) => {
              const meta = [
                s.kind === 'package' && s.package_sessions ? t('place.packageSessions', { n: num(s.package_sessions) }) : null,
                s.duration_minutes ? t('place.minutes', { n: num(s.duration_minutes) }) : null,
                ageLabel(s.age_min_months, s.age_max_months),
                s.kind === 'package' && s.package_validity_days ? t('place.validity', { n: num(s.package_validity_days) }) : null,
                s.format ? t(`place.format.${s.format}`) : null,
              ].filter(Boolean);
              return (
                <li key={s.id} className="cdp-svc">
                  <div className="cdp-svc-main">
                    <strong>{s.name}</strong>
                    {meta.length > 0 ? <span className="cdp-dim">{meta.join(' · ')}</span> : null}
                  </div>
                  {s.price_toman !== null ? <strong>{price(s.price_toman)}</strong> : null}
                </li>
              );
            })}
          </ul>
        </Card>
      ) : null}

      <Card>
        <h2 className="csv-sub">{t('detail.hours')}</h2>
        {l.weekly_hours.length > 0 ? (
          <ul className="cdp-hours">
            {[0, 1, 2, 3, 4, 5, 6].map((d) => {
              const h = hoursByDay.get(d);
              return (
                <li key={d}>
                  <span>{t(DAY_KEYS[d])}</span>
                  <span dir="ltr">{h ? `${num(h.open)} – ${num(h.close)}` : t('place.closedDay')}</span>
                </li>
              );
            })}
          </ul>
        ) : l.hours_text ? (
          <p className="csv-text">{l.hours_text}</p>
        ) : null}
        <p className="cdp-health">
          <Icon name="info" size={14} />
          {t('place.healthNote')}
        </p>
      </Card>

      {r.rating_count > 0 || r.latest.length > 0 ? (
        <Card>
          <h2 className="csv-sub">{t('place.reviewsTitle')}</h2>
          <p className="cdp-hint">{t('place.reviewsNote')}</p>
          <div className="cdp-rate">
            <strong className="cdp-rate-n">{r.rating_avg !== null ? dec(r.rating_avg) : '—'}</strong>
            <ul className="cdp-scores">
              {scores.map(([k, v]) =>
                v !== null ? (
                  <li key={k}>
                    <span>{t(`place.score.${k}`)}</span>
                    <span className="cdp-bar" aria-hidden>
                      <span className="cdp-bar-fill" style={{ inlineSize: `${Math.max(0, Math.min(100, (v / 5) * 100))}%` }} />
                    </span>
                    <span>{dec(v)}</span>
                  </li>
                ) : null,
              )}
            </ul>
          </div>
          <ul className="cdp-reviews">
            {r.latest.map((rv) => (
              <li key={rv.id} className="cdp-review">
                <div className="cdp-review-head">
                  <strong>{rv.author_first_name ? t('place.mother', { name: rv.author_first_name }) : t('place.motherAnon')}</strong>
                  <span className="cdh-rating">
                    <Icon name="star" size={12} />
                    {num(rv.rating)}
                  </span>
                </div>
                {rv.body ? <p className="csv-text">{rv.body}</p> : null}
              </li>
            ))}
          </ul>
        </Card>
      ) : null}

      {address || l.rules || cancel ? (
        <Card>
          <dl className="cdp-info">
            {address ? (
              <div>
                <dt>{t('detail.address')}</dt>
                <dd>{address}</dd>
              </div>
            ) : null}
            {l.rules ? (
              <div>
                <dt>{t('place.rules')}</dt>
                <dd>{l.rules}</dd>
              </div>
            ) : null}
            {cancel ? (
              <div>
                <dt>{t('place.cancel')}</dt>
                <dd>{cancel}</dd>
              </div>
            ) : null}
          </dl>
        </Card>
      ) : null}
    </>
  );
}
