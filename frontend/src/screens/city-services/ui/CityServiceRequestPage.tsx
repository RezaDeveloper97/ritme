'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { AGE_RANGES, useCityServiceListing, useCreateCityBooking } from '@/entities/city-service';
import { getApiErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { addDays, formatLongDate, toApiDate, today } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { Icon, InfoNote, PrimaryButton, ScreenHeader, SkyLayer } from '@/shared/ui';

import {
  type BookingFormErrors,
  type BookingFormState,
  DAY_CHOICES,
  emptyBookingForm,
  NAME_MAX,
  NOTE_MAX,
  validateBookingForm,
} from '../model/form';
import { BookSlotFlow } from './BookSlotFlow';
import { CsError, CsSkeleton } from './ListSkeleton';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/csv.css';

/**
 * `/city-services/[id]/request` — the booking request for a `request`-mode venue: name, phone, a preferred day,
 * an optional child AGE RANGE and a short note, with a «don't write medical details» warning. Personal fields go to
 * the API only (never to storage or analytics); nothing about health or the cycle is part of the form.
 */
export function CityServiceRequestPage({
  id,
  service = null,
  slot = null,
  reschedule = null,
}: {
  id: number;
  /** CB-DIR-09: `?service=` / `?slot=` from the place page, `?reschedule=<booking id>` from «my bookings». */
  service?: number | null;
  slot?: number | null;
  reschedule?: number | null;
}) {
  const t = useTranslations('cityServices');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const listing = useCityServiceListing(id);
  const create = useCreateCityBooking();

  const days = useMemo(() => Array.from({ length: DAY_CHOICES }, (_, i) => addDays(today(), i)), []);
  const [state, setState] = useState<BookingFormState>(() => emptyBookingForm(toApiDate(today())));
  const [errors, setErrors] = useState<BookingFormErrors>({});
  const [formError, setFormError] = useState<string | null>(null);

  const set = <K extends keyof BookingFormState>(key: K, value: BookingFormState[K]) => setState((s) => ({ ...s, [key]: value }));
  const back = () => router.back(`/city-services/${id}`);

  const onSubmit = () => {
    setFormError(null);
    const found = validateBookingForm(state);
    setErrors(found);
    if (Object.keys(found).length > 0 || !listing.data) return;
    create.mutate(
      {
        listingId: listing.data.id,
        contactName: state.name.trim(),
        contactPhone: state.phone,
        preferredDate: state.day,
        childAgeRange: state.age,
        note: state.note,
      },
      {
        onSuccess: () => router.push('/profile/bookings'),
        onError: (e) => setFormError(getApiErrorMessage(e) ?? t('form.saveError')),
      },
    );
  };

  const err = (k: keyof BookingFormErrors) =>
    errors[k] ? (
      <p id={`csv-err-${k}`} role="alert" className="csv-err">
        {t(`form.${errors[k]}`)}
      </p>
    ) : null;

  let body;
  if (listing.isPending) {
    body = <CsSkeleton label={t('loading')} />;
  } else if (listing.isError) {
    body = <CsError title={t('loadError')} retry={t('retry')} busy={listing.isFetching} onRetry={() => void listing.refetch()} />;
  } else if (listing.data.booking_mode === 'online' || reschedule !== null) {
    // CB-DIR-09: an `online` place is booked by picking a slot (service, child, day, time) — a request the team confirms.
    body = <BookSlotFlow listing={listing.data} initialService={service} initialSlot={slot} rescheduleId={reschedule} />;
  } else if (listing.data.booking_mode !== 'request') {
    body = <InfoNote icon="info">{t('form.notRequestable')}</InfoNote>;
  } else {
    body = (
      <form
        className="csv-stack"
        noValidate
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit();
        }}
      >
        <p className="csv-for">{t('form.for', { name: listing.data.name })}</p>
        <InfoNote icon="shield">{t('form.warn')}</InfoNote>

        <section className="card fld-card flex flex-col gap-3.5">
          {(
            [
              ['name', 'user', 'text', NAME_MAX, 'name'],
              ['phone', 'phone', 'tel', 24, 'tel'],
            ] as const
          ).map(([key, icon, type, max, auto]) => (
            <label key={key} className="fld-label">
              <span className="fld-label-t">{t(`form.${key}`)}</span>
              <span className="field">
                <Icon name={icon} size={18} />
                <input
                  type={type}
                  inputMode={type === 'tel' ? 'tel' : undefined}
                  autoComplete={auto}
                  maxLength={max}
                  value={state[key]}
                  aria-invalid={Boolean(errors[key])}
                  aria-describedby={errors[key] ? `csv-err-${key}` : undefined}
                  onChange={(e) => set(key, e.target.value)}
                  {...pilot(`city-request.field:${key}`, 'text')}
                />
              </span>
              {err(key)}
            </label>
          ))}
        </section>

        <section className="card fld-card flex flex-col gap-3.5">
          <div className="flex flex-col gap-2">
            <span className="fld-label-t text-start">{t('form.day')}</span>
            <div role="radiogroup" aria-label={t('form.day')} className="fld-chips csv-days" {...pilot('city-request.day', 'choice')}>
              {days.map((d) => {
                const v = toApiDate(d);
                return (
                  <button key={v} type="button" role="radio" aria-checked={state.day === v} {...pilotOption(v)} className={clsx('chip', state.day === v && 'on')} onClick={() => set('day', v)}>
                    {formatLongDate(d, locale)}
                  </button>
                );
              })}
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <span className="fld-label-t text-start">{t('form.age')}</span>
            <div role="radiogroup" aria-label={t('form.age')} className="fld-chips" {...pilot('city-request.age', 'choice')}>
              <button type="button" role="radio" aria-checked={state.age === null} {...pilotOption('none')} className={clsx('chip', state.age === null && 'on')} onClick={() => set('age', null)}>
                {t('form.ageNone')}
              </button>
              {AGE_RANGES.map((a) => (
                <button key={a} type="button" role="radio" aria-checked={state.age === a} {...pilotOption(a)} className={clsx('chip', state.age === a && 'on')} onClick={() => set('age', a)}>
                  {t(`ages.${a}`)}
                </button>
              ))}
            </div>
          </div>
          <label className="fld-label">
            <span className="fld-label-t">{t('form.note')}</span>
            <span className="field csv-area">
              <textarea rows={2} maxLength={NOTE_MAX} placeholder={t('form.notePlaceholder')} value={state.note} onChange={(e) => set('note', e.target.value)} {...pilot('city-request.note', 'text')} />
            </span>
          </label>
        </section>

        {formError && (
          <p role="alert" className="csv-err">
            {formError}
          </p>
        )}
        <PrimaryButton type="submit" block loading={create.isPending} {...pilot('city-request.submit')}>
          {create.isPending ? t('form.submitting') : t('form.submit')}
        </PrimaryButton>
      </form>
    );
  }

  return (
    <div className="view csv-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={reschedule !== null ? t('book.rescheduleTitle') : listing.data?.booking_mode === 'online' ? t('book.title') : t('form.title')} onBack={back} backLabel={t('back')} />
        <div className="csv-body">{body}</div>
        <div className="page-tail" />
      </div>
    </div>
  );
}
