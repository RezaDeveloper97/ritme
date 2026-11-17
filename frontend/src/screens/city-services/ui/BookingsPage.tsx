'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useModuleEnabled } from '@/entities/app-module';
import {
  type BookingStatus,
  type CityServiceBooking,
  useCancelCityBooking,
  useCityBookingScope,
} from '@/entities/city-service';
import { ReviewCityBookingForm } from '@/features/review-city-booking';
import { getApiErrorMessage } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, formatWeekday, fromApiDate } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import { Card, EmptyState, Icon, PrimaryButton, ScreenHeader, SecondaryButton, SkyLayer, StatusPill, type Tone } from '@/shared/ui';

import { bookingWhen } from '../model/book';
import { neshanHref } from '../model/place';
import { CsError, CsSkeleton } from './ListSkeleton';

const TONE: Record<BookingStatus, Tone> = {
  requested: 'brand',
  confirmed: 'success',
  rejected: 'danger',
  cancelled: 'neutral',
  completed: 'data',
};

type Scope = 'upcoming' | 'past';

/**
 * `/profile/bookings` — «رزروهای من» (nbl_Dir_MyBookings), upcoming / past tabs over `city_service_bookings` (it
 * supersedes bloom B-N10-05's flat list). Each row: user-visible status + the team's note, directions, cancel,
 * reschedule (the request flow with `?reschedule=`), review (pending until moderated) and book again. Doctor
 * appointments live at /services/bookings and stay linked from here.
 */
export function BookingsPage() {
  const t = useTranslations('cityServices');
  const doctorsOn = useModuleEnabled('doctors');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const [scope, setScope] = useState<Scope>('upcoming');
  const q = useCityBookingScope(scope);
  const cancel = useCancelCityBooking();
  const [confirming, setConfirming] = useState<number | null>(null);
  const [reviewing, setReviewing] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const num = (v: number | string) => formatNumber(v, locale);

  const doCancel = (id: number) => {
    setError(null);
    cancel.mutate(id, {
      onSuccess: () => setConfirming(null),
      // 422 cancel_window_passed → the API's own message (after the free window she contacts the place).
      onError: (e) => setError(getApiErrorMessage(e) ?? t('bookings.cancelError')),
    });
  };

  const row = (b: CityServiceBooking) => {
    const when = bookingWhen(b);
    const date = fromApiDate(when.day);
    const l = b.listing;
    const title = b.service?.name ?? l.name;
    return (
      <li key={b.id} className="csv-bk">
        <Link href={`/city-services/booked/${b.id}`} className="csv-bk-head" {...pilot(`city-bookings.open:${b.id}`)}>
          <span className="csv-thumb" aria-hidden>
            {l.photo_url ? (
              // eslint-disable-next-line @next/next/no-img-element -- admin-uploaded public-disk image, sized by CSS
              <img src={l.photo_url} alt="" loading="lazy" />
            ) : (
              <Icon name={l.category.icon} size={26} strokeWidth={1.4} />
            )}
          </span>
          <span className="csv-card-main">
            <span className="csv-card-title">{title}</span>
            <span className="csv-card-meta">{[b.service ? l.name : l.category.name, l.district ?? l.city.name].filter(Boolean).join(' · ')}</span>
            <span className="csv-card-meta">
              {[`${formatWeekday(date, locale)} ${formatDayMonth(date, locale)}`, when.time ? num(when.time) : null, b.child?.name].filter(Boolean).join(' · ')}
            </span>
          </span>
          <StatusPill tone={TONE[b.status]}>{t(`bookings.status.${b.status}`)}</StatusPill>
        </Link>
        {b.status_note ? <p className="csv-bk-note">{t('bookings.teamNote', { note: b.status_note })}</p> : null}
        {b.review ? (
          <p className="csv-bk-line">{b.review.status === 'published' ? t('bookings.reviewPublished') : t('bookings.reviewPending')}</p>
        ) : null}

        {confirming === b.id ? (
          <div className="csv-bk-confirm" role="group" aria-label={t('bookings.cancelConfirm')}>
            <span>{t('bookings.cancelConfirm')}</span>
            <PrimaryButton block={false} onClick={() => doCancel(b.id)} loading={cancel.isPending}>
              {cancel.isPending ? t('bookings.cancelling') : t('bookings.cancelYes')}
            </PrimaryButton>
            <SecondaryButton block={false} variant="text" onClick={() => setConfirming(null)}>
              {t('bookings.cancelNo')}
            </SecondaryButton>
          </div>
        ) : reviewing === b.id ? (
          <ReviewCityBookingForm bookingId={b.id} onDone={() => setReviewing(null)} onCancel={() => setReviewing(null)} />
        ) : (
          <div className="cdb-btns">
            {scope === 'upcoming' && l.lat !== null && l.lng !== null ? (
              <a className="csv-go-btn" href={neshanHref(l.lat, l.lng)} target="_blank" rel="noopener noreferrer">
                <Icon name="mapPin" size={16} />
                {t('place.directions')}
              </a>
            ) : null}
            {b.can_reschedule ? (
              <SecondaryButton block={false} variant="text" onClick={() => router.push(`/city-services/${l.id}/request?reschedule=${b.id}`)} {...pilot(`city-bookings.reschedule:${b.id}`)}>
                {t('bookings.reschedule')}
              </SecondaryButton>
            ) : null}
            {b.can_cancel ? (
              <SecondaryButton block={false} variant="text" onClick={() => setConfirming(b.id)}>
                {t('bookings.cancel')}
              </SecondaryButton>
            ) : null}
            {b.can_review ? (
              <SecondaryButton block={false} variant="text" onClick={() => setReviewing(b.id)} {...pilot(`city-bookings.review:${b.id}`)}>
                {t('bookings.review')}
              </SecondaryButton>
            ) : null}
            {scope === 'past' ? (
              <SecondaryButton block={false} variant="text" onClick={() => router.push(`/city-services/${l.id}`)} {...pilot(`city-bookings.again:${b.id}`)}>
                {t('bookings.again')}
              </SecondaryButton>
            ) : null}
          </div>
        )}
      </li>
    );
  };

  let content;
  if (q.isPending) {
    content = <CsSkeleton label={t('bookings.loading')} />;
  } else if (q.isError) {
    content = <CsError title={t('bookings.loadError')} retry={t('retry')} busy={q.isFetching} onRetry={() => void q.refetch()} />;
  } else if (q.data.bookings.length === 0) {
    content = (
      <Card>
        <EmptyState
          icon="calendar"
          title={t('bookings.emptyTitle')}
          body={t('bookings.emptyBody')}
          action={<PrimaryButton onClick={() => router.push('/city-services')} {...pilot('city-bookings.browse')}>{t('bookings.browse')}</PrimaryButton>}
        />
      </Card>
    );
  } else {
    content = <ul className="csv-list">{q.data.bookings.map(row)}</ul>;
  }

  const counts = q.data?.counts;
  return (
    <div className="view csv-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('bookings.title')} subtitle={t('bookings.tag')} onBack={() => router.push('/profile')} backLabel={t('back')} />
        <div className="csv-body">
          <div role="tablist" aria-label={t('bookings.title')} className="cdb-tabs">
            {(['upcoming', 'past'] as const).map((s) => (
              <button key={s} type="button" role="tab" aria-selected={scope === s} {...pilot(`city-bookings.tab:${s}`)} className={clsx('cdb-tab', scope === s && 'on')} onClick={() => setScope(s)}>
                {t(`bookings.tab.${s}`)}
                {counts && counts[s] > 0 ? ` · ${num(counts[s])}` : ''}
              </button>
            ))}
          </div>
          {content}
          {error ? (
            <p role="alert" className="csv-err">
              {error}
            </p>
          ) : null}
          {doctorsOn ? (
            <Link href="/services/bookings" className="csv-link" {...pilot('city-bookings.doctors')}>
              <Icon name="doctor" size={18} />
              {t('bookings.doctorBookings')}
            </Link>
          ) : null}
        </div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
