'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import type { CityServiceDetail, CityServiceService, CityServiceSlot } from '@/entities/city-service';
import { Link, type Locale } from '@/shared/i18n';
import { formatDecimal, formatNumber, formatWeekday, fromApiDate } from '@/shared/lib/date';
import { Card, Icon } from '@/shared/ui';

import { bookable, monthsLabelParts, neshanHref, priceParts, safeWebsite, slotDayOffset, slotParts } from '../model/place';

/* CB-DIR-08 — sections of the place page (nbl_Dir_Place). Prices are the venue's own declarations, display only. */

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

type T = ReturnType<typeof useTranslations>;

function useFmt() {
  const locale = useLocale() as Locale;
  const t = useTranslations('cityServices');
  return {
    t,
    locale,
    num: (v: number | string) => formatNumber(v, locale),
    dec: (v: number) => formatDecimal(v, locale),
  };
}

/** «۳۲۰ هزار تومان» / «۲٫۴ میلیون تومان». */
export function usePrice(): (toman: number) => string {
  const { t, dec } = useFmt();
  return (toman) => {
    const p = priceParts(toman);
    return t(p.unit === 'million' ? 'place.priceM' : 'place.priceK', { n: dec(p.value) });
  };
}

function ageRange(t: T, num: (v: number) => string, min: number | null, max: number | null): string | null {
  const part = (m: number) => {
    const p = monthsLabelParts(m);
    return t(p.unit === 'year' ? 'place.years' : 'place.months', { n: num(p.n) });
  };
  if (min !== null && max !== null) return t('place.ageBetween', { from: part(min), to: part(max) });
  if (min !== null) return t('place.ageFrom', { from: part(min) });
  if (max !== null) return t('place.ageTo', { to: part(max) });
  return null;
}

/** Rating · open state · action row (call / directions / website). */
export function PlaceHead({ l }: { l: CityServiceDetail }) {
  const { t, num, dec } = useFmt();
  const site = safeWebsite(l.website_url);
  const actions: { key: string; icon: 'phone' | 'mapPin' | 'globe'; label: string; href: string; external: boolean }[] = [];
  if (l.phone) actions.push({ key: 'call', icon: 'phone', label: t('place.call'), href: `tel:${l.phone.replace(/[^\d+]/g, '')}`, external: false });
  if (l.lat !== null && l.lng !== null) actions.push({ key: 'dir', icon: 'mapPin', label: t('place.directions'), href: neshanHref(l.lat, l.lng), external: true });
  if (site) actions.push({ key: 'web', icon: 'globe', label: t('place.website'), href: site, external: true });
  return (
    <>
      <p className="csv-card-meta cdp-meta">
        {l.verified ? (
          <span className="csv-badge cdp-verified">
            <Icon name="shield" size={12} />
            {t('home.verified')}
          </span>
        ) : null}
        {l.rating_avg !== null ? (
          <span className="cdh-rating">
            <Icon name="star" size={12} />
            {dec(l.rating_avg)} <span className="cdp-dim">({t('place.reviewCount', { n: num(l.rating_count) })})</span>
          </span>
        ) : null}
        {l.open_now !== null ? (
          <span className={l.open_now ? 'cdh-open' : 'cdp-closed'}>{l.open_now ? t('home.openNow') : t('place.closedNow')}</span>
        ) : null}
        {l.distance_km !== null ? <span className="cdp-dim">{t('home.km', { km: dec(l.distance_km) })}</span> : null}
      </p>
      {actions.length > 0 ? (
        <ul className="cdp-actions">
          {actions.map((a) => (
            <li key={a.key}>
              {/* External links open with noopener noreferrer; only http(s) reaches here (safeWebsite). */}
              <a className="cdp-action" href={a.href} {...(a.external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}>
                <Icon name={a.icon} size={18} />
                {a.label}
              </a>
            </li>
          ))}
        </ul>
      ) : null}
    </>
  );
}

/** About text, age range and amenity chips. */
export function PlaceAbout({ l }: { l: CityServiceDetail }) {
  const { t } = useFmt();
  const about = l.about ?? l.description;
  const amenityName = (code: string) => (code in AMENITY_KEYS ? t(AMENITY_KEYS[code as keyof typeof AMENITY_KEYS]) : code.replace(/_/g, ' '));
  if (!about && l.amenities.length === 0) return null;
  return (
    <Card>
      <h3 className="csv-sub">{t('detail.about')}</h3>
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
  );
}

/** Services & prices; «رزرو» picks the service (and loads its slots). */
export function PlaceServices({
  services,
  serviceId,
  onPick,
}: {
  services: CityServiceService[];
  serviceId: number | null;
  onPick: (id: number) => void;
}) {
  const { t, num } = useFmt();
  const price = usePrice();
  if (services.length === 0) return null;
  return (
    <Card>
      <h3 className="csv-sub">{t('place.servicesTitle')}</h3>
      <p className="cdp-hint">{t('place.servicesNote')}</p>
      <ul className="cdp-svc-list">
        {services.map((s) => {
          const meta = [
            s.kind === 'package' && s.package_sessions ? t('place.packageSessions', { n: num(s.package_sessions) }) : null,
            s.duration_minutes ? t('place.minutes', { n: num(s.duration_minutes) }) : null,
            ageRange(t, num, s.age_min_months, s.age_max_months),
            s.kind === 'package' && s.package_validity_days ? t('place.validity', { n: num(s.package_validity_days) }) : null,
            s.format ? t(`place.format.${s.format}`) : null,
          ].filter(Boolean);
          return (
            <li key={s.id} className="cdp-svc">
              <div className="cdp-svc-main">
                <strong>{s.name}</strong>
                {meta.length > 0 ? <span className="cdp-dim">{meta.join(' · ')}</span> : null}
              </div>
              <div className="cdp-svc-side">
                {s.price_toman !== null ? <strong>{price(s.price_toman)}</strong> : null}
                <button type="button" className={clsx('cdp-pick', serviceId === s.id && 'on')} aria-pressed={serviceId === s.id} onClick={() => onPick(s.id)}>
                  {t('place.book')}
                </button>
              </div>
            </li>
          );
        })}
      </ul>
    </Card>
  );
}

/** Nearest slots (a service's slots once one is picked). */
export function PlaceSlots({
  slots,
  slotId,
  loading,
  onPick,
}: {
  slots: CityServiceSlot[];
  slotId: number | null;
  loading: boolean;
  onPick: (id: number) => void;
}) {
  const { t, num, locale } = useFmt();
  const label = (s: CityServiceSlot) => {
    const p = slotParts(s.starts_at);
    if (!p) return s.starts_at;
    const off = slotDayOffset(p.day);
    const day = off === 0 ? t('place.today') : off === 1 ? t('place.tomorrow') : formatWeekday(fromApiDate(p.day), locale);
    return `${day} ${num(p.time)}`;
  };
  return (
    <Card>
      <h3 className="csv-sub">{t('place.slotsTitle')}</h3>
      {loading ? (
        <p className="cdp-hint">{t('loading')}</p>
      ) : slots.length === 0 ? (
        <p className="cdp-hint">{t('place.slotsEmpty')}</p>
      ) : (
        <ul className="cdp-slots">
          {slots.map((s) => (
            <li key={s.id}>
              <button
                type="button"
                className={clsx('cdp-slot', slotId === s.id && 'on')}
                aria-pressed={slotId === s.id}
                disabled={!bookable(s)}
                onClick={() => onPick(s.id)}
              >
                {label(s)}
                {!bookable(s) ? <small>{t('place.full')}</small> : s.seats_left <= 3 ? <small>{t('place.seatsLeft', { n: num(s.seats_left) })}</small> : null}
              </button>
            </li>
          ))}
        </ul>
      )}
      <p className="cdp-health">
        <Icon name="info" size={14} />
        {t('place.healthNote')}
      </p>
    </Card>
  );
}

/** Reviews summary with sub-scores and the latest reviews (first name only). */
export function PlaceReviews({ l }: { l: CityServiceDetail }) {
  const { t, num, dec } = useFmt();
  const r = l.reviews;
  if (r.rating_count === 0 && r.latest.length === 0) return null;
  const scores = [
    ['cleanliness', r.scores.cleanliness],
    ['instructor', r.scores.instructor],
    ['amenities', r.scores.amenities],
  ] as const;
  return (
    <Card>
      <h3 className="csv-sub">{t('place.reviewsTitle')}</h3>
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
  );
}

/** Address, weekly hours, rules, cancel policy, owner/report links. */
export function PlaceInfo({ l }: { l: CityServiceDetail }) {
  const { t, num } = useFmt();
  const address = [l.district, l.address].filter(Boolean).join(' · ');
  const hoursByDay = new Map(l.weekly_hours.map((h) => [h.day, h]));
  const cancel =
    l.cancel_policy ?? (l.cancel_free_hours !== null ? t('place.cancelFree', { n: num(l.cancel_free_hours) }) : null);
  return (
    <Card>
      <dl className="cdp-info">
        {address ? (
          <div>
            <dt>{t('detail.address')}</dt>
            <dd>{address}</dd>
          </div>
        ) : null}
        {l.weekly_hours.length > 0 ? (
          <div>
            <dt>{t('detail.hours')}</dt>
            <dd>
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
            </dd>
          </div>
        ) : l.hours_text ? (
          <div>
            <dt>{t('detail.hours')}</dt>
            <dd>{l.hours_text}</dd>
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
      <p className="cdp-links">
        <Link href="/city-services/join" className="csv-link">
          {t('place.owner')}
        </Link>
        {/* TODO(CB-DIR-08): «گزارش اطلاعات نادرست» — wire to the report endpoint once CB-DIR-03 ships it. */}
      </p>
    </Card>
  );
}
