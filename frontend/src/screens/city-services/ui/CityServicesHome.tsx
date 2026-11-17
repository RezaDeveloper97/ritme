'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type FormEvent, useState } from 'react';

import { ChildAvatar, useChildren } from '@/entities/child';
import {
  ageRangeLabel,
  nextBooking,
  useAgeSuggest,
  useCityBookings,
  useCityChoice,
  useCityServiceOptions,
  useDirectoryListings,
  type CityServiceCard,
  type CityServiceCatalogOption,
} from '@/entities/city-service';
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { Card, Icon, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { CsError } from './ListSkeleton';

/** What the home hands to the list view: a category, a search text, or «suitable for the child's age» (months). */
export interface BrowseQuery {
  category: string | null;
  q: string;
  ageMonths: number | null;
}

interface Props {
  onBrowse: (query: BrowseQuery) => void;
}

/**
 * CB-DIR-06 — `nbl_Dir_Home`: «مادر و کودک» home. City + child header, search, category chips, the next booking,
 * places that suit the child's age (switchable), places near the user, the «join» CTA and the ranking/age notes.
 * Only the child's age in whole months leaves the device (as an `age_months` search param); no name, no birth date.
 * Ranking is distance + age + rating — nothing is paid for. The map (DIR-07) and join flow (DIR-10) are linked by
 * route; they ship in their own tasks.
 */
export function CityServicesHome({ onBrowse }: Props) {
  const t = useTranslations('cityServices');
  const locale = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const router = useRouter();
  const options = useCityServiceOptions();
  const [city, setCity, cityReady] = useCityChoice();
  const [ageOn, setAgeOn, ageReady] = useAgeSuggest();
  const children = useChildren();
  const bookings = useCityBookings();
  const [q, setQ] = useState('');
  const [pickCity, setPickCity] = useState(false);

  const cities = options.data?.cities ?? [];
  const activeCity = city !== null && cities.some((c) => c.id === city) ? city : null;
  const cityName = cities.find((c) => c.id === activeCity)?.name ?? t('list.allCities');
  const child = children.data?.children[0] ?? null;
  const childMonths = child ? child.age.months : null;
  const ageMonths = ageOn && childMonths !== null ? Math.min(216, childMonths) : null;
  const ready = cityReady && ageReady;

  // «مناسب سن …»: ranked by age fit; only places whose age data fits are shown.
  const suited = useDirectoryListings({ city: activeCity, ageMonths, sort: 'rank', perPage: 8 }, ready && ageMonths !== null);
  const near = useDirectoryListings({ city: activeCity, sort: 'rank', perPage: 4 }, ready);
  const catalog = options.data?.place_age_ranges ?? [];
  const upcoming = bookings.data ? nextBooking(bookings.data, toApiDate(today())) : null;
  const suitedList = (suited.data?.listings ?? []).filter((l) => l.age_fit === true);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    onBrowse({ category: null, q, ageMonths: null });
  };

  const meta = (l: CityServiceCard) => (
    <span className="cdh-meta">
      {l.rating_avg !== null ? (
        <span className="cdh-rating">
          <Icon name="star" size={12} />
          {formatNumber(l.rating_avg, locale)}
        </span>
      ) : null}
      <span>
        {[l.district ?? l.city.name, l.distance_km !== null ? t('home.km', { km: formatNumber(l.distance_km, locale) }) : null]
          .filter(Boolean)
          .join(' · ')}
      </span>
    </span>
  );

  const thumb = (l: CityServiceCard) =>
    l.photo_url ? (
      // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
      <img src={l.photo_url} alt="" loading="lazy" />
    ) : (
      <Icon name={l.category.icon} size={26} strokeWidth={1.4} />
    );

  const rail = (l: CityServiceCard, cat: CityServiceCatalogOption[]) => {
    const ages = ageRangeLabel(l.age_ranges, cat);
    return (
      <li key={l.id} className="cdh-rail-item">
        <Link href={`/city-services/${l.id}`} className="cdh-rail-card" {...pilot(`city-home.rail:${l.id}`)}>
          <span className="cdh-rail-art" aria-hidden>
            {thumb(l)}
          </span>
          <span className="cdh-card-title">{l.name}</span>
          {meta(l)}
          {ages ? <span className="cdh-ages">{t('home.suitable', { ages })}</span> : null}
        </Link>
      </li>
    );
  };

  const row = (l: CityServiceCard) => (
    <li key={l.id}>
      <Link href={`/city-services/${l.id}`} className="csv-card" {...pilot(`city-home.listing:${l.id}`)}>
        <span className="csv-thumb" aria-hidden>
          {thumb(l)}
        </span>
        <span className="csv-card-main">
          <span className="csv-card-title">{l.name}</span>
          {meta(l)}
          <span className="csv-card-meta">
            {l.open_now === true ? <span className="cdh-open">{t('home.openNow')}</span> : null}
            {l.verified ? <span className="csv-badge">{t('home.verified')}</span> : null}
          </span>
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="csv-go" />
      </Link>
    </li>
  );

  if (options.isPending || !ready) {
    return (
      <div className="view csv-view">
        <SkyLayer />
        <div className="scroll">
          <ScreenHeader title={t('title')} onBack={() => router.push('/services')} backLabel={t('back')} />
          <div className="csv-body">
            <SkeletonGroup label={t('loading')} className="csv-list">
              <Skeleton shape="block" />
              <Skeleton shape="block" />
              <Skeleton shape="block" />
            </SkeletonGroup>
          </div>
        </div>
        <BottomNav />
      </div>
    );
  }

  return (
    <div className="view csv-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('title')} subtitle={t('tag')} onBack={() => router.push('/services')} backLabel={t('back')} />
        <div className="csv-body">
          {options.isError ? (
            <CsError title={t('loadError')} retry={t('retry')} busy={options.isFetching} onRetry={() => void options.refetch()} />
          ) : (
            <>
              <div className="cdh-top">
                <button type="button" className="cdh-city" aria-expanded={pickCity} {...pilot('city-home.city')} onClick={() => setPickCity((v) => !v)}>
                  <Icon name="mapPin" size={16} />
                  <span>{cityName}</span>
                  <Icon name="chevronDown" size={14} />
                </button>
                <Link href="/profile/bookings" className="cdh-bk" {...pilot('city-home.bookings')} aria-label={t('list.bookings')}>
                  <Icon name="calendar" size={20} />
                </Link>
              </div>
              {child ? (
                <p className="cdh-for">
                  <ChildAvatar id={child.id} name={child.name} initial={child.initial} sex={child.sex} size={36} />
                  <span>{t('home.forYouAnd', { name: child.name })}</span>
                </p>
              ) : null}
              {pickCity ? (
                <div role="radiogroup" aria-label={t('list.cityLabel')} className="csv-chips" {...pilot('city-home.city-pick', 'choice')}>
                  {[{ id: null as number | null, name: t('list.allCities') }, ...cities].map((c) => (
                    <button
                      key={String(c.id)}
                      type="button"
                      role="radio"
                      aria-checked={activeCity === c.id}
                      {...pilotOption(c.id === null ? 'all' : String(c.id))}
                      className={activeCity === c.id ? 'chip csv-chip on' : 'chip csv-chip'}
                      onClick={() => {
                        setCity(c.id);
                        setPickCity(false);
                      }}
                    >
                      {c.name}
                    </button>
                  ))}
                </div>
              ) : null}

              <div className="cdh-find">
                <form className="cdh-search" role="search" onSubmit={submit}>
                  <Icon name="search" size={18} />
                  <input
                    type="search"
                    value={q}
                    maxLength={80}
                    placeholder={t('home.searchPlaceholder')}
                    aria-label={t('home.searchLabel')}
                    onChange={(e) => setQ(e.target.value)}
                    {...pilot('city-home.search', 'text')}
                  />
                </form>
                <Link href="/city-services/map" className="cdh-map-btn" {...pilot('city-home.map')} aria-label={t('home.map')}>
                  <Icon name="mapPin" size={20} />
                </Link>
              </div>

              <ul className="cdh-cats" aria-label={t('list.categoryLabel')}>
                {options.data?.categories.map((c) => (
                  <li key={c.slug}>
                    <button type="button" className="cdh-cat" {...pilot(`city-home.category:${c.slug}`)} onClick={() => onBrowse({ category: c.slug, q: '', ageMonths: null })}>
                      <span className="cdh-cat-ic" aria-hidden>
                        <Icon name={c.icon ?? 'home'} size={22} strokeWidth={1.6} />
                      </span>
                      <span className="cdh-cat-name">{c.name}</span>
                    </button>
                  </li>
                ))}
              </ul>

              {upcoming ? (
                <Link href="/profile/bookings" className="cdh-next" {...pilot('city-home.next-booking')}>
                  <span className="cdh-next-k">{t('home.nextBooking')}</span>
                  <span className="cdh-next-title">{upcoming.listing.name}</span>
                  <span className="cdh-next-day">{formatLongDate(fromApiDate(upcoming.preferred_date), locale)}</span>
                </Link>
              ) : null}

              {child && ageMonths !== null && suitedList.length > 0 ? (
                <section className="cdh-sec" aria-labelledby="cdh-age">
                  <div className="cdh-sec-head">
                    <div>
                      <h2 id="cdh-age" className="cdh-sec-title">
                        {t('home.suitedTitle', { name: child.name })}
                      </h2>
                      <p className="cdh-sec-sub">{t('home.suitedSub', { age: child.age.label })}</p>
                    </div>
                    <button type="button" className="cdh-sec-all" {...pilot('city-home.suited-all')} onClick={() => onBrowse({ category: null, q: '', ageMonths })}>
                      {t('home.all')}
                    </button>
                  </div>
                  <ul className="cdh-rail">{suitedList.map((l) => rail(l, catalog))}</ul>
                </section>
              ) : null}

              <section className="cdh-sec" aria-labelledby="cdh-near">
                <div className="cdh-sec-head">
                  <h2 id="cdh-near" className="cdh-sec-title">
                    {t('home.nearTitle')}
                  </h2>
                  <span className="cdh-sec-links">
                    <button type="button" className="cdh-sec-all" {...pilot('city-home.list')} onClick={() => onBrowse({ category: null, q: '', ageMonths: null })}>
                      {t('home.list')}
                    </button>
                    <Link href="/city-services/map" className="cdh-sec-all" {...pilot('city-home.map-all')}>
                      {t('home.map')}
                    </Link>
                  </span>
                </div>
                {near.isPending ? (
                  <SkeletonGroup label={t('loading')} className="csv-list">
                    <Skeleton shape="block" />
                    <Skeleton shape="block" />
                  </SkeletonGroup>
                ) : near.isError ? (
                  <CsError title={t('loadError')} retry={t('retry')} busy={near.isFetching} onRetry={() => void near.refetch()} />
                ) : near.data.listings.length === 0 ? (
                  <Card>
                    <p className="cdh-empty">{t('list.emptyBody')}</p>
                  </Card>
                ) : (
                  <ul className="csv-list">{near.data.listings.map(row)}</ul>
                )}
              </section>

              <Link href="/city-services/join" className="cdh-join" {...pilot('city-home.join')}>
                <span className="cdh-join-title">{t('home.joinTitle')}</span>
                <span className="cdh-join-sub">{t('home.joinSub')}</span>
              </Link>

              <p className="csv-note">{t('home.rankNote')}</p>
              <p className="csv-note">
                {t('home.ageNote')}{' '}
                <button type="button" className="cdh-toggle" {...pilot('city-home.age-toggle')} onClick={() => setAgeOn(!ageOn)}>
                  {ageOn ? t('home.ageOff') : t('home.ageOn')}
                </button>
              </p>
              <p className="csv-note">{t('note')}</p>
            </>
          )}
        </div>
        <div className="page-tail" />
      </div>
      <BottomNav />
    </div>
  );
}
