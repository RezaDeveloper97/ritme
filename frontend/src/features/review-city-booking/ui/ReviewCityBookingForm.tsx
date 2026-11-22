'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useReviewCityBooking } from '@/entities/city-service';
import { getApiErrorMessage } from '@/shared/api';
import { type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { Icon, PrimaryButton, SecondaryButton } from '@/shared/ui';
import '@/shared/ui/styles/csv.css';
import '@/shared/ui/styles/cdb.css';

/**
 * CB-DIR-13 (F14) — the review form of a completed place booking: 1-5 stars + an optional text. It is stored
 * pending until the team moderates it. Shared by «رزروهای من» and the booked page (one source of the logic).
 */
export function ReviewCityBookingForm({ bookingId, onDone, onCancel }: { bookingId: number; onDone?: () => void; onCancel: () => void }) {
  const t = useTranslations('cityServices');
  const locale = useLocale() as Locale;
  const review = useReviewCityBooking();
  const [rating, setRating] = useState(0);
  const [body, setBody] = useState('');
  const [error, setError] = useState<string | null>(null);

  const send = () => {
    if (rating < 1) return;
    setError(null);
    review.mutate(
      { id: bookingId, rating, body },
      {
        onSuccess: () => onDone?.(),
        onError: (e) => setError(getApiErrorMessage(e) ?? t('bookings.reviewError')),
      },
    );
  };

  return (
    <div className="cdb-review">
      <div role="radiogroup" aria-label={t('bookings.rating')} className="cdb-stars" {...pilot('city-review.rating', 'choice')}>
        {[1, 2, 3, 4, 5].map((n) => (
          <button
            key={n}
            type="button"
            role="radio"
            aria-checked={rating === n}
            {...pilotOption(String(n))}
            aria-label={formatNumber(n, locale)}
            className={clsx('cdb-star', n <= rating && 'on')}
            onClick={() => setRating(n)}
          >
            <Icon name="star" size={22} />
          </button>
        ))}
      </div>
      <span className="field csv-area">
        <textarea rows={2} maxLength={1000} placeholder={t('bookings.reviewPlaceholder')} value={body} onChange={(e) => setBody(e.target.value)} {...pilot('city-review.body', 'text')} />
      </span>
      <p className="csv-note">{t('bookings.reviewNote')}</p>
      {error ? (
        <p role="alert" className="csv-err">
          {error}
        </p>
      ) : null}
      <div className="cdb-btns">
        <PrimaryButton block={false} disabled={rating < 1} loading={review.isPending} onClick={send} {...pilot('city-review.send')}>
          {t('bookings.reviewSend')}
        </PrimaryButton>
        <SecondaryButton block={false} variant="text" onClick={onCancel}>
          {t('bookings.cancelNo')}
        </SecondaryButton>
      </div>
    </div>
  );
}
