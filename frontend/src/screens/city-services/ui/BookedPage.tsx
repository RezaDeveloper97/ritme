'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCancelCityBooking, useCityBooking } from '@/entities/city-service';
import { ReviewCityBookingForm } from '@/features/review-city-booking';
import { getApiErrorMessage } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, formatWeekday, fromApiDate } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import { Icon, InfoNote, PrimaryButton, ScreenHeader, SecondaryButton, SkyLayer, StatusPill } from '@/shared/ui';

import { bookingWhen } from '../model/book';
import { neshanHref, slotParts } from '../model/place';
import { CsError, CsSkeleton } from './ListSkeleton';
import { usePrice } from './PlaceSections';

/**
 * `/city-services/booked/[id]` (nbl_Dir_Booked). Honest about the status: a new booking is a REQUEST the Ritme team
 * confirms — the screen only says «confirmed» when the API status is `confirmed`. Price is «pay at the place».
 */

/** CB-DIR-13: «free cancellation until …» is only said while it is still true — not for a completed / cancelled booking or a window that already closed. */
function freeCancelOpen(status: string, until: string): boolean {
  if (status !== 'requested' && status !== 'confirmed') return false;
  const t = Date.parse(until.replace(' ', 'T'));
  return Number.isNaN(t) || t > Date.now();
}

export function BookedPage({ id }: { id: number }) {
  const t = useTranslations('cityServices');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const price = usePrice();
  const q = useCityBooking(id);
  const cancel = useCancelCityBooking();
  const [confirming, setConfirming] = useState(false);
  const [reviewing, setReviewing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const num = (v: number | string) => formatNumber(v, locale);
  /** «سه‌شنبه ۲۱ مهر ۱۷:۰۰» from the API's Tehran wall-clock stamp. */
  const untilLabel = (stamp: string) => {
    const p = slotParts(stamp);
    if (!p) return stamp;
    const d = fromApiDate(p.day);
    return `${formatWeekday(d, locale)} ${formatDayMonth(d, locale)} ${num(p.time)}`;
  };

  const doCancel = () => {
    setError(null);
    cancel.mutate(id, {
      onSuccess: () => {
        setConfirming(false);
        void q.refetch();
      },
      // 422 cancel_window_passed → the API's message (she contacts the place).
      onError: (e) => setError(getApiErrorMessage(e) ?? t('bookings.cancelError')),
    });
  };

  let body;
  if (q.isPending) {
    body = <CsSkeleton label={t('loading')} />;
  } else if (q.isError) {
    body = <CsError title={t('bookings.loadError')} retry={t('retry')} busy={q.isFetching} onRetry={() => void q.refetch()} />;
  } else {
    const b = q.data;
    const when = bookingWhen(b);
    const date = fromApiDate(when.day);
    const l = b.listing;
    const tone = b.status === 'confirmed' ? 'success' : b.status === 'requested' ? 'brand' : b.status === 'completed' ? 'data' : b.status === 'rejected' ? 'danger' : 'neutral';
    const place = [l.name, l.district].filter(Boolean).join(' · ');
    const rows: { icon: 'calendar' | 'user' | 'note'; title: string; sub: string | null }[] = [
      {
        icon: 'calendar',
        title: `${formatWeekday(date, locale)} ${formatDayMonth(date, locale)}`,
        sub: when.time ? num(when.time) : t('booked.dayOnly'),
      },
    ];
    if (b.child) rows.push({ icon: 'user', title: b.child.name, sub: t('booked.childAge', { n: num(b.child.age_months) }) });
    if (b.note) rows.push({ icon: 'note', title: t('booked.yourNote'), sub: b.note });
    const hasPrice = b.price_toman !== null;

    body = (
      <>
        <div className="cdb-hero">
          <span className={`cdb-hero-ic is-${b.status}`} aria-hidden>
            <Icon name={b.status === 'rejected' || b.status === 'cancelled' ? 'x' : b.status === 'requested' ? 'clock' : 'check'} size={34} strokeWidth={2.2} />
          </span>
          <h2 className="cdb-hero-t">{t(`booked.title.${b.status}`)}</h2>
          <p className="cdb-hero-s">{t(`booked.lead.${b.status}`, { name: l.name })}</p>
          <StatusPill tone={tone}>{t(`bookings.status.${b.status}`)}</StatusPill>
          {b.code ? <p className="cdb-code">{t('booked.code', { code: b.code })}</p> : null}
        </div>

        <section className="card cdb-rows">
          <div className="cdb-row">
            <span className="cdb-row-ic" aria-hidden>
              <Icon name="mapPin" size={18} />
            </span>
            <span className="cdb-opt-main">
              <strong>{b.service?.name ?? l.name}</strong>
              <small>{b.service ? place : l.category.name}</small>
            </span>
          </div>
          {rows.map((r) => (
            <div key={r.icon} className="cdb-row">
              <span className="cdb-row-ic" aria-hidden>
                <Icon name={r.icon} size={18} />
              </span>
              <span className="cdb-opt-main">
                <strong>{r.title}</strong>
                {r.sub ? <small>{r.sub}</small> : null}
              </span>
            </div>
          ))}
          <div className="cdb-row">
            <span className="cdb-row-ic" aria-hidden>
              <Icon name="info" size={18} />
            </span>
            <span className="cdb-opt-main">
              <strong>{hasPrice && b.price_toman !== null ? price(b.price_toman) : t('book.payAtPlace')}</strong>
              <small>{hasPrice ? t('book.payAtPlace') : t('booked.noPrice')}</small>
            </span>
          </div>
        </section>

        {b.status_note ? <p className="csv-bk-note">{t('bookings.teamNote', { note: b.status_note })}</p> : null}
        {b.status === 'requested' ? <InfoNote icon="info">{t('booked.waiting')}</InfoNote> : null}

        <div className="cdb-actions">
          {l.lat !== null && l.lng !== null ? (
            <a className="csv-go-btn" href={neshanHref(l.lat, l.lng)} target="_blank" rel="noopener noreferrer">
              <Icon name="mapPin" size={16} />
              {t('place.directions')}
            </a>
          ) : null}
          {l.phone ? (
            <a className="csv-go-btn" href={`tel:${l.phone.replace(/[^\d+]/g, '')}`}>
              <Icon name="phone" size={16} />
              {t('booked.callPlace')}
            </a>
          ) : null}
        </div>

        {b.can_cancel ? (
          confirming ? (
            <div className="csv-bk-confirm" role="group" aria-label={t('bookings.cancelConfirm')}>
              <span>{t('bookings.cancelConfirm')}</span>
              <PrimaryButton block={false} onClick={doCancel} loading={cancel.isPending}>
                {cancel.isPending ? t('bookings.cancelling') : t('bookings.cancelYes')}
              </PrimaryButton>
              <SecondaryButton block={false} variant="text" onClick={() => setConfirming(false)}>
                {t('bookings.cancelNo')}
              </SecondaryButton>
            </div>
          ) : (
            <SecondaryButton block={false} variant="text" onClick={() => setConfirming(true)}>
              {t('bookings.cancel')}
            </SecondaryButton>
          )
        ) : null}
        {/* CB-DIR-13 F14: a completed booking can be rated here too (same form as «رزروهای من»). */}
        {b.review ? (
          <p className="csv-bk-line">{b.review.status === 'published' ? t('bookings.reviewPublished') : t('bookings.reviewPending')}</p>
        ) : b.can_review ? (
          reviewing ? (
            <ReviewCityBookingForm bookingId={b.id} onDone={() => setReviewing(false)} onCancel={() => setReviewing(false)} />
          ) : (
            <SecondaryButton block={false} variant="text" onClick={() => setReviewing(true)} {...pilot('city-booked.review')}>
              {t('bookings.review')}
            </SecondaryButton>
          )
        ) : null}
        {error ? (
          <p role="alert" className="csv-err">
            {error}
          </p>
        ) : null}
        {b.cancel_free_until && freeCancelOpen(b.status, b.cancel_free_until) ? (
          <p className="csv-note">{t('booked.cancelUntil', { when: untilLabel(b.cancel_free_until) })}</p>
        ) : b.cancel_policy ? (
          <p className="csv-note">{b.cancel_policy}</p>
        ) : null}
        <Link href="/profile/bookings" className="csv-link">
          {t('booked.all')}
        </Link>
      </>
    );
  }

  return (
    <div className="view csv-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('booked.header')} onBack={() => router.push('/profile/bookings')} backLabel={t('back')} />
        <div className="csv-body">{body}</div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
