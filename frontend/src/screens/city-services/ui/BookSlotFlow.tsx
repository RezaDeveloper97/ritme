'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { useChildren } from '@/entities/child';
import {
  type CityServiceDetail,
  useCityBooking,
  useCityServiceSlots,
  useCreateCityBooking,
  useRescheduleCityBooking,
} from '@/entities/city-service';
import { useCurrentUser } from '@/entities/user';
import { getApiErrorMessage } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, formatWeekday, fromApiDate } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { Icon, InfoNote, PrimaryButton } from '@/shared/ui';

import { ageFits, groupSlotsByDay, totalMonths } from '../model/book';
import { NAME_MAX, NOTE_MAX } from '../model/form';
import { bookable } from '../model/place';
import { CsSkeleton } from './ListSkeleton';
import { usePrice } from './PlaceSections';

/**
 * CB-DIR-09 — the slot booking request (nbl_Dir_Book): service, her own child, day strip, time grid, note, summary.
 * It is a REQUEST the Ritme team confirms; the price is shown as «pay at the place» (no payment step). With
 * `rescheduleId` it only asks for another time of an existing booking. Children show a first name + age only.
 */
export function BookSlotFlow({
  listing,
  initialService,
  initialSlot,
  rescheduleId,
}: {
  listing: CityServiceDetail;
  initialService: number | null;
  initialSlot: number | null;
  rescheduleId: number | null;
}) {
  const t = useTranslations('cityServices');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const price = usePrice();
  const num = (v: number | string) => formatNumber(v, locale);
  const create = useCreateCityBooking();
  const resched = useRescheduleCityBooking();
  const children = useChildren({ enabled: rescheduleId === null });
  const existing = useCityBooking(rescheduleId ?? 0);

  const services = listing.services;
  const [serviceId, setServiceId] = useState<number | null>(initialService ?? services[0]?.id ?? null);
  const effectiveService = rescheduleId !== null ? (existing.data?.service?.id ?? null) : serviceId;
  const service = services.find((s) => s.id === effectiveService);
  const slotsQ = useCityServiceSlots(listing.id, effectiveService);
  const days = useMemo(() => groupSlotsByDay(slotsQ.data ?? []), [slotsQ.data]);

  const [slotId, setSlotId] = useState<number | null>(initialSlot);
  const [dayPick, setDayPick] = useState<string | null>(null);
  const [childId, setChildId] = useState<number | null>(null);
  const [note, setNote] = useState('');
  const [error, setError] = useState<string | null>(null);
  // CB-DIR-13 F12: the API defaults the contact name to the account's name; an account without one has to type it.
  const me = useCurrentUser();
  const needsName = rescheduleId === null && me.isSuccess && !(me.data.name ?? '').trim();
  const [contactName, setContactName] = useState('');
  const [nameMissing, setNameMissing] = useState(false);

  // The picked slot decides the day; otherwise the chosen day, otherwise the soonest day with a seat.
  const slotDay = days.find((d) => d.slots.some((s) => s.slot.id === slotId))?.day ?? null;
  const day = dayPick ?? slotDay ?? days.find((d) => d.slots.some((s) => bookable(s.slot)))?.day ?? days[0]?.day ?? null;
  const dayEntry = days.find((d) => d.day === day);
  const picked = days.flatMap((d) => d.slots).find((s) => s.slot.id === slotId);

  const owned = (children.data?.children ?? []).filter((c) => c.role === 'owner');
  const child = owned.find((c) => c.id === childId) ?? (owned.length === 1 ? owned[0] : null);
  const busy = create.isPending || resched.isPending;

  const dayLabel = (d: string) => formatWeekday(fromApiDate(d), locale);
  const whenLabel = (d: string, time: string) => `${dayLabel(d)} ${formatDayMonth(fromApiDate(d), locale)} · ${num(time)}`;

  const submit = () => {
    if (slotId === null || busy) return;
    setError(null);
    if (needsName && contactName.trim() === '') {
      setNameMissing(true);
      return;
    }
    const fail = (e: unknown) => {
      setError(getApiErrorMessage(e) ?? t('book.error'));
      void slotsQ.refetch(); // the seat may be gone — show the fresh grid
    };
    if (rescheduleId !== null) {
      resched.mutate({ id: rescheduleId, slotId }, { onSuccess: (b) => router.push(`/city-services/booked/${b.id}`), onError: fail });
      return;
    }
    create.mutate(
      { listingId: listing.id, serviceId: effectiveService, slotId, childId: child?.id ?? null, note, ...(needsName ? { contactName: contactName.trim() } : {}) },
      { onSuccess: (b) => router.push(`/city-services/booked/${b.id}`), onError: fail },
    );
  };

  if (rescheduleId !== null && existing.isPending) return <CsSkeleton label={t('loading')} />;
  const total = service?.price_toman ?? null;

  return (
    <div className="csv-stack">
      <p className="csv-for">{rescheduleId !== null ? t('book.rescheduleFor', { name: listing.name }) : t('form.for', { name: listing.name })}</p>

      {rescheduleId === null && services.length > 0 ? (
        <section className="cdb-sec">
          <h3 className="csv-sub">{t('book.service')}</h3>
          <div role="radiogroup" aria-label={t('book.service')} className="cdb-list" {...pilot('city-book.service', 'choice')}>
            {services.map((s) => {
              const meta = [s.duration_minutes ? t('place.minutes', { n: num(s.duration_minutes) }) : null, s.price_toman !== null ? price(s.price_toman) : null].filter(Boolean);
              return (
                <button
                  key={s.id}
                  type="button"
                  role="radio"
                  aria-checked={serviceId === s.id}
                  {...pilotOption(String(s.id))}
                  className={clsx('cdb-opt', serviceId === s.id && 'on')}
                  onClick={() => {
                    setServiceId(s.id);
                    setSlotId(null);
                    setDayPick(null);
                  }}
                >
                  <span className="cdb-radio" aria-hidden />
                  <span className="cdb-opt-main">
                    <strong>{s.name}</strong>
                    {meta.length ? <small>{meta.join(' · ')}</small> : null}
                  </span>
                </button>
              );
            })}
          </div>
        </section>
      ) : null}

      {rescheduleId === null ? (
        <section className="cdb-sec">
          <h3 className="csv-sub">{t('book.child')}</h3>
          {children.isPending ? null : (
            <div className="cdb-chips" role="radiogroup" aria-label={t('book.child')} {...pilot('city-book.child', 'choice')}>
              {owned.map((c) => {
                const fits = ageFits(totalMonths(c.age), service);
                return (
                  <button key={c.id} type="button" role="radio" aria-checked={child?.id === c.id} {...pilotOption(String(c.id))} className={clsx('cdb-child', child?.id === c.id && 'on')} onClick={() => setChildId(c.id)}>
                    <span className="cdb-av" aria-hidden>
                      {c.initial}
                    </span>
                    <span className="cdb-opt-main">
                      <strong>{c.name}</strong>
                      <small>{[c.age.label, fits ? null : t('book.ageMismatch')].filter(Boolean).join(' · ')}</small>
                    </span>
                  </button>
                );
              })}
              <Link href="/children/new" className="cdb-child cdb-add">
                <Icon name="plus" size={18} />
                <span>{t('book.addChild')}</span>
              </Link>
            </div>
          )}
        </section>
      ) : null}

      <section className="cdb-sec">
        <h3 className="csv-sub">{t('book.day')}</h3>
        {slotsQ.isPending && effectiveService !== null ? (
          <CsSkeleton label={t('loading')} />
        ) : days.length === 0 ? (
          <InfoNote icon="info">{t('book.noSlots')}</InfoNote>
        ) : (
          <>
            <div role="radiogroup" aria-label={t('book.day')} className="cdb-days" {...pilot('city-book.day', 'choice')}>
              {days.slice(0, 14).map((d) => (
                <button key={d.day} type="button" role="radio" aria-checked={day === d.day} {...pilotOption(d.day)} className={clsx('cdb-day', day === d.day && 'on')} onClick={() => setDayPick(d.day)}>
                  <small>{dayLabel(d.day)}</small>
                  <strong>{formatDayMonth(fromApiDate(d.day), locale)}</strong>
                </button>
              ))}
            </div>
            <h3 className="csv-sub">{t('book.time')}</h3>
            <ul className="cdp-slots" {...pilot('city-book.slot', 'choice')}>
              {dayEntry?.slots.map(({ slot, time }) => (
                <li key={slot.id}>
                  <button type="button" className={clsx('cdp-slot', slotId === slot.id && 'on')} aria-pressed={slotId === slot.id} {...pilotOption(String(slot.id))} disabled={!bookable(slot)} onClick={() => setSlotId(slot.id)}>
                    {num(time)}
                    {!bookable(slot) ? <small>{t('place.full')}</small> : slot.seats_left <= 3 ? <small>{t('place.seatsLeft', { n: num(slot.seats_left) })}</small> : null}
                  </button>
                </li>
              ))}
            </ul>
          </>
        )}
      </section>

      {needsName ? (
        <label className="fld-label">
          <span className="fld-label-t">{t('form.name')}</span>
          <span className="field">
            <Icon name="user" size={18} />
            <input
              type="text"
              autoComplete="name"
              maxLength={NAME_MAX}
              value={contactName}
              aria-invalid={nameMissing && contactName.trim() === ''}
              onChange={(e) => setContactName(e.target.value)}
              {...pilot('city-book.name', 'text')}
            />
          </span>
          {nameMissing && contactName.trim() === '' ? (
            <p role="alert" className="csv-err">
              {t('form.required')}
            </p>
          ) : null}
        </label>
      ) : null}

      {rescheduleId === null ? (
        <label className="fld-label">
          <span className="fld-label-t">{t('book.note')}</span>
          <span className="field csv-area">
            <textarea rows={3} maxLength={NOTE_MAX} placeholder={t('book.notePlaceholder')} value={note} onChange={(e) => setNote(e.target.value)} {...pilot('city-book.note', 'text')} />
          </span>
        </label>
      ) : null}

      {picked && day ? (
        <section className="card cdb-sum" aria-live="polite">
          <p className="cdb-sum-t">{[service?.name, whenLabel(day, picked.time)].filter(Boolean).join(' · ')}</p>
          <dl>
            {total !== null ? (
              <div>
                <dt>{t('book.amount')}</dt>
                <dd>{price(total)}</dd>
              </div>
            ) : null}
            <div>
              <dt>{t('book.pay')}</dt>
              <dd>{t('book.payAtPlace')}</dd>
            </div>
            {listing.cancel_free_hours !== null ? (
              <div>
                <dt>{t('place.cancel')}</dt>
                <dd>{t('place.cancelFree', { n: num(listing.cancel_free_hours) })}</dd>
              </div>
            ) : null}
          </dl>
        </section>
      ) : null}

      <InfoNote icon="shield">{t('book.dataNote')}</InfoNote>
      <p className="csv-note">{t('book.requestNote')}</p>
      {error ? (
        <p role="alert" className="csv-err">
          {error}
        </p>
      ) : null}

      <div className="cdp-bar-sticky">
        <div className="cdp-bar-price">
          {/* CB-DIR-13: «Payment 320k» next to «Send request» read like an online charge; it is the venue's price (paid there). */}
          <small>{total !== null ? t('book.amount') : t('book.pay')}</small>
          <strong>{total !== null ? price(total) : t('book.payAtPlace')}</strong>
        </div>
        <PrimaryButton block={false} disabled={slotId === null} loading={busy} onClick={submit} {...pilot('city-book.submit')}>
          {busy ? t('form.submitting') : rescheduleId !== null ? t('book.sendReschedule') : t('form.submit')}
        </PrimaryButton>
      </div>
    </div>
  );
}
