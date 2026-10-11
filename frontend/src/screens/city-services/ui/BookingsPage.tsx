'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  type BookingStatus,
  useCancelCityBooking,
  useCityBookings,
  type CityServiceBooking,
} from '@/entities/city-service';
import { getApiErrorMessage } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, fromApiDate } from '@/shared/lib/date';
import { Card, EmptyState, Icon, PrimaryButton, ScreenHeader, SecondaryButton, SkyLayer, StatusPill, type Tone } from '@/shared/ui';

import { CsError, CsSkeleton } from './ListSkeleton';

const TONE: Record<BookingStatus, Tone> = {
  requested: 'brand',
  confirmed: 'success',
  rejected: 'danger',
  cancelled: 'neutral',
  completed: 'data',
};

/**
 * `/profile/bookings` — «رزروها»: my city-service requests with their user-visible status and the team's note, and
 * cancel for an open one. A thin list over the same table the later directory bookings (CB-DIR-09) build on; the
 * doctor appointments live at /services/bookings and are linked from here.
 */
export function BookingsPage() {
  const t = useTranslations('cityServices');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const q = useCityBookings();
  const cancel = useCancelCityBooking();
  const [confirming, setConfirming] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const doCancel = (id: number) => {
    setError(null);
    cancel.mutate(id, {
      onSuccess: () => setConfirming(null),
      onError: (e) => setError(getApiErrorMessage(e) ?? t('bookings.cancelError')),
    });
  };

  const row = (b: CityServiceBooking) => (
    <li key={b.id} className="csv-bk">
      <Link href={`/city-services/${b.listing.id}`} className="csv-bk-head">
        <span className="csv-thumb" aria-hidden>
          {b.listing.photo_url ? (
            // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
            <img src={b.listing.photo_url} alt="" loading="lazy" />
          ) : (
            <Icon name={b.listing.category.icon} size={26} strokeWidth={1.4} />
          )}
        </span>
        <span className="csv-card-main">
          <span className="csv-card-title">{b.listing.name}</span>
          <span className="csv-card-meta">{[b.listing.category.name, b.listing.city.name, b.listing.district].filter(Boolean).join(' · ')}</span>
        </span>
        <StatusPill tone={TONE[b.status]}>{t(`bookings.status.${b.status}`)}</StatusPill>
      </Link>
      <p className="csv-bk-line">{t('bookings.day', { day: formatLongDate(fromApiDate(b.preferred_date), locale) })}</p>
      {b.child_age_range ? <p className="csv-bk-line">{t('bookings.age', { age: t(`ages.${b.child_age_range as '0-1'}`) })}</p> : null}
      {b.status_note ? <p className="csv-bk-note">{t('bookings.teamNote', { note: b.status_note })}</p> : null}
      {b.can_cancel ? (
        confirming === b.id ? (
          <div className="csv-bk-confirm" role="group" aria-label={t('bookings.cancelConfirm')}>
            <span>{t('bookings.cancelConfirm')}</span>
            <PrimaryButton onClick={() => doCancel(b.id)} loading={cancel.isPending}>
              {cancel.isPending ? t('bookings.cancelling') : t('bookings.cancelYes')}
            </PrimaryButton>
            <SecondaryButton variant="text" onClick={() => setConfirming(null)}>
              {t('bookings.cancelNo')}
            </SecondaryButton>
          </div>
        ) : (
          <SecondaryButton variant="text" onClick={() => setConfirming(b.id)}>
            {t('bookings.cancel')}
          </SecondaryButton>
        )
      ) : null}
    </li>
  );

  let body;
  if (q.isPending) {
    body = <CsSkeleton label={t('bookings.loading')} />;
  } else if (q.isError) {
    body = <CsError title={t('bookings.loadError')} retry={t('retry')} busy={q.isFetching} onRetry={() => void q.refetch()} />;
  } else if (q.data.length === 0) {
    body = (
      <Card>
        <EmptyState
          icon="calendar"
          title={t('bookings.emptyTitle')}
          body={t('bookings.emptyBody')}
          action={<PrimaryButton onClick={() => router.push('/city-services')}>{t('bookings.browse')}</PrimaryButton>}
        />
      </Card>
    );
  } else {
    body = <ul className="csv-list">{q.data.map(row)}</ul>;
  }

  return (
    <div className="view csv-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('bookings.title')} subtitle={t('bookings.tag')} onBack={() => router.push('/profile')} backLabel={t('back')} />
        <div className="csv-body">
          {body}
          {error ? (
            <p role="alert" className="csv-err">
              {error}
            </p>
          ) : null}
          <Link href="/services/bookings" className="csv-link">
            <Icon name="doctor" size={18} />
            {t('bookings.doctorBookings')}
          </Link>
        </div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
