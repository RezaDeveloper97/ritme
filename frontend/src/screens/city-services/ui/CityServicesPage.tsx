'use client';

import clsx from 'clsx';
import { useFormatter, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCityChoice, useCityServiceListings, useCityServiceOptions, type CityServiceCard } from '@/entities/city-service';
import { Link, useDirection, useRouter } from '@/shared/i18n';
import { Card, EmptyState, Icon, ScreenHeader, SecondaryButton, SkyLayer } from '@/shared/ui';

import { CsError, CsSkeleton } from './ListSkeleton';

/**
 * `/city-services` — «برای مادر و کودک» (entry from the services hub section): classes, pools and playhouses near the
 * user. The city is remembered on the device only; filters are a city chip row and a category chip row. A tap opens
 * the venue. The screen shows no map, no reviews and nothing from the user's health data.
 */
export function CityServicesPage() {
  const t = useTranslations('cityServices');
  const fmt = useFormatter();
  const router = useRouter();
  const rtl = useDirection() === 'rtl';
  const options = useCityServiceOptions();
  const [city, setCity, ready] = useCityChoice();
  const [category, setCategory] = useState<string | null>(null);
  const [page, setPage] = useState(1);

  // A remembered city that no longer has listings must not hide everything: fall back to «همه شهرها».
  const cities = options.data?.cities ?? [];
  const activeCity = city !== null && cities.some((c) => c.id === city) ? city : null;
  const list = useCityServiceListings(activeCity, category, page);

  const pick = (next: number | null) => {
    setCity(next);
    setPage(1);
  };

  const card = (l: CityServiceCard) => (
    <li key={l.id}>
      <Link href={`/city-services/${l.id}`} className="csv-card">
        <span className="csv-thumb" aria-hidden>
          {l.photo_url ? (
            // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
            <img src={l.photo_url} alt="" loading="lazy" />
          ) : (
            <Icon name={l.category.icon} size={28} strokeWidth={1.4} />
          )}
        </span>
        <span className="csv-card-main">
          <span className="csv-card-title">{l.name}</span>
          <span className="csv-card-meta">
            <span className="csv-badge">{l.category.name}</span>
            <span>{[l.city.name, l.district].filter(Boolean).join(' · ')}</span>
          </span>
          <span className="csv-card-meta">
            {l.price_text ? <span className="csv-price">{l.price_text}</span> : null}
            <span className="csv-mode">{t(`list.mode.${l.booking_mode}`)}</span>
          </span>
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="csv-go" />
      </Link>
    </li>
  );

  let body;
  if (options.isPending || list.isPending || !ready) {
    body = <CsSkeleton label={t('loading')} />;
  } else if (options.isError || list.isError) {
    body = (
      <CsError
        title={t('loadError')}
        retry={t('retry')}
        busy={options.isFetching || list.isFetching}
        onRetry={() => {
          void options.refetch();
          void list.refetch();
        }}
      />
    );
  } else if (list.data.listings.length === 0) {
    body = (
      <Card>
        <EmptyState icon="home" title={t('list.emptyTitle')} body={t('list.emptyBody')} />
      </Card>
    );
  } else {
    body = (
      <>
        <ul className="csv-list">{list.data.listings.map(card)}</ul>
        {list.data.meta.last_page > 1 ? (
          <nav className="csv-pager" aria-label={t('list.pages')}>
            <SecondaryButton variant="text" disabled={page <= 1} onClick={() => setPage((n) => n - 1)}>
              {t('list.prev')}
            </SecondaryButton>
            <span>{t('list.page', { page: fmt.number(page), last: fmt.number(list.data.meta.last_page) })}</span>
            <SecondaryButton variant="text" disabled={page >= list.data.meta.last_page} onClick={() => setPage((n) => n + 1)}>
              {t('list.next')}
            </SecondaryButton>
          </nav>
        ) : null}
      </>
    );
  }

  const chip = (key: string, label: string, on: boolean, onClick: () => void) => (
    <button key={key} type="button" role="radio" aria-checked={on} className={clsx('chip csv-chip', on && 'on')} onClick={onClick}>
      {label}
    </button>
  );

  return (
    <div className="view csv-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('title')} subtitle={t('tag')} onBack={() => router.push('/services')} backLabel={t('back')} />
        <div className="csv-body">
          {options.data ? (
            <>
              {cities.length > 0 ? (
                <div role="radiogroup" aria-label={t('list.cityLabel')} className="csv-chips">
                  {chip('all', t('list.allCities'), activeCity === null, () => pick(null))}
                  {cities.map((c) => chip(String(c.id), c.name, activeCity === c.id, () => pick(c.id)))}
                </div>
              ) : null}
              <div role="radiogroup" aria-label={t('list.categoryLabel')} className="csv-chips">
                {chip('all', t('list.allCategories'), category === null, () => {
                  setCategory(null);
                  setPage(1);
                })}
                {options.data.categories.map((c) =>
                  chip(c.slug, c.name, category === c.slug, () => {
                    setCategory(c.slug);
                    setPage(1);
                  }),
                )}
              </div>
            </>
          ) : null}
          {body}
          <Link href="/profile/bookings" className="csv-link">
            <Icon name="calendar" size={18} />
            {t('list.bookings')}
          </Link>
          <p className="csv-note">{t('note')}</p>
        </div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
