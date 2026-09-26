'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { type Appointment, type AppointmentKind, useAppointment } from '@/entities/care-reminder';
import { useUserMode } from '@/entities/message';
import { usePregnancyStatus } from '@/entities/pregnancy';
import {
  useCancelAppointment,
  useTogglePrepItem,
  useUpdateAppointment,
} from '@/features/manage-appointment';
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import {
  formatDayMonth,
  formatNumber,
  formatWeekdayDayMonth,
  fromApiDate,
  monthName,
  toApiDate,
  today,
  toParts,
} from '@/shared/lib/date';
import { buildIcs, downloadIcs } from '@/shared/lib/ics';
import { AppSheet } from '@/shared/sheet';
import { Icon, type IconName } from '@/shared/ui';

import {
  appointmentIcsEvent,
  daysUntil,
  mapsHref,
  pregnancyWeekAt,
  reminderMoment,
  splitScheduled,
} from '../model/detail';

const KIND_ICON: Record<AppointmentKind, IconName> = {
  in_person: 'doctor',
  phone: 'phone',
  online: 'video',
};

/**
 * Appointment detail (`v13_AppointmentDetail`): countdown header, hero card,
 * who / about / place / reminder rows, the prep checklist, add-to-calendar
 * (`.ics`) and cancel. A cancelled appointment renders muted without cancel.
 */
export function AppointmentDetailPage({ id }: { id: number }) {
  const t = useTranslations('care');
  const query = useAppointment(Number.isFinite(id) ? id : null);

  if (!query.data) {
    return (
      <div className="view rmd-page">
        <div className="scroll">
          <DetailHeader id={id} />
          <div className="rmd-body">
            {query.isError || !Number.isFinite(id) ? (
              <div className="rmd-state">
                <p className="rmd-empty">{t('loadError')}</p>
                <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
                  {t('retry')}
                </button>
              </div>
            ) : (
              <div className="rmd-state" aria-busy="true" aria-label={t('appointmentDetail.loading')}>
                <span className="skeleton-line rmd-row-skel" />
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }
  return <Detail appt={query.data} />;
}

function DetailHeader({ id, sub, editable = false }: { id: number; sub?: string; editable?: boolean }) {
  const t = useTranslations('care');
  const dir = useDirection();
  return (
    <header className="rmd-hdr">
      <Link href="/reminders?tab=appointments" className="rmd-hdr-btn" aria-label={t('back')}>
        <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={20} strokeWidth={1.8} />
      </Link>
      <div className="rmd-hdr-text">
        <h1 className="rmd-hdr-title">{t('appointmentDetail.title')}</h1>
        {sub && <p className="rmd-hdr-sub">{sub}</p>}
      </div>
      {editable ? (
        <Link
          href={`/reminders/appointment/${id}/edit`}
          className="rmd-hdr-btn"
          aria-label={t('appointmentDetail.edit')}
        >
          <Icon name="pencil" size={18} strokeWidth={1.8} />
        </Link>
      ) : (
        <span className="rmd-hdr-btn invisible" aria-hidden />
      )}
    </header>
  );
}

function Detail({ appt }: { appt: Appointment }) {
  const t = useTranslations('care');
  const td = useTranslations('care.appointmentDetail');
  const tf = useTranslations('care.appointmentForm');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mode = useUserMode();
  const pregnancy = usePregnancyStatus();
  const update = useUpdateAppointment();
  const togglePrep = useTogglePrepItem();
  const cancel = useCancelAppointment();
  const [confirming, setConfirming] = useState(false);

  const cancelled = appt.status === 'cancelled';
  const when = splitScheduled(appt.scheduledAt);
  const date = when ? fromApiDate(when.date) : null;
  const parts = date ? toParts(date, locale) : null;
  const days = when ? daysUntil(when.date, toApiDate(today())) : null;
  const time = when ? formatNumber(when.time, locale) : '';
  const week =
    mode.data?.mode === 'pregnancy' && days !== null
      ? pregnancyWeekAt(pregnancy.data?.gestationalAge?.totalDays, days)
      : null;
  const maps = mapsHref(appt);
  const topic = tf(`topics.${appt.topic}`);
  const placeLabel =
    appt.kind === 'online' ? tf('placeOnline') : appt.kind === 'phone' ? tf('placePhone') : td('place');
  const fires = when ? reminderMoment(when, appt.remindBefore) : null;

  const addToCalendar = () => {
    const description = appt.withWhom ? td('descriptionWith', { with: appt.withWhom }) : null;
    const event = appointmentIcsEvent(appt, appt.title || topic, description);
    if (event) downloadIcs(`${td('calendarFile')}-${appt.id}.ics`, buildIcs(event));
  };

  const doCancel = () =>
    cancel.mutate(appt.id, {
      onSuccess: () => {
        setConfirming(false);
        router.push('/reminders?tab=appointments');
      },
    });

  return (
    <div className="view rmd-page">
      <div className="scroll">
        <DetailHeader
          id={appt.id}
          editable={!cancelled}
          sub={cancelled ? td('cancelled') : days !== null && days >= 0 ? td('daysUntil', { days }) : undefined}
        />

        <div className={clsx('rmd-body', cancelled && 'opacity-60')}>
          <section className="rmd-today">
            <div className="fld-chips">
              <span className="chip">
                <Icon name={KIND_ICON[appt.kind]} size={14} strokeWidth={1.8} />
                {tf(`kinds.${appt.kind}`)}
              </span>
              {week !== null && (
                <span className="chip on">{td('pregnancyWeek', { week: formatNumber(week, locale) })}</span>
              )}
              {cancelled && <span className="chip">{td('cancelled')}</span>}
            </div>
            <div className="rmd-row">
              <span className="rmd-date" aria-hidden>
                {parts && (
                  <>
                    <span className="rmd-date-day">{formatNumber(parts.day, locale)}</span>
                    <span className="rmd-date-month">{monthName(parts.month, locale)}</span>
                  </>
                )}
              </span>
              <span className="rmd-row-body">
                <span className="rmd-row-title">{appt.title || topic}</span>
                {date && (
                  <span className="rmd-row-meta">
                    {formatWeekdayDayMonth(date, locale)} · {td('timeAt', { time })}
                  </span>
                )}
                <span className="rmd-row-meta">{topic}</span>
              </span>
            </div>
          </section>

          <div className="rmd-list">
            <InfoRow icon="user" label={td('with')} value={[appt.withWhom, appt.specialty].filter(Boolean).join(' · ')} />
            <InfoRow icon="note" label={td('about')} value={topic} />
            {appt.location && (
              <div className="rmd-row">
                <span className="rmd-tile" aria-hidden>
                  <Icon name={appt.kind === 'online' ? 'video' : appt.kind === 'phone' ? 'phone' : 'mapPin'} size={20} />
                </span>
                <span className="rmd-row-body">
                  <span className="rmd-row-meta">{placeLabel}</span>
                  <span className="rmd-row-title" dir="auto">{appt.location}</span>
                </span>
                {maps && (
                  <a className="rmd-sec-add" href={maps} target="_blank" rel="noopener noreferrer">
                    {td('directions')}
                  </a>
                )}
              </div>
            )}
            <div className="rmd-row">
              <span className="rmd-tile" aria-hidden>
                <Icon name="bellRing" size={20} />
              </span>
              <span className="rmd-row-body">
                <span className="rmd-row-meta">{td('reminder')}</span>
                <span className="rmd-row-title">
                  {appt.isActive
                    ? fires
                      ? td('reminderAt', {
                          before: tf(`remindOptions.${appt.remindBefore}`),
                          date: formatDayMonth(fromApiDate(fires.date), locale),
                          time: formatNumber(fires.time, locale),
                        })
                      : tf(`remindOptions.${appt.remindBefore}`)
                    : td('reminderOff')}
                </span>
              </span>
              <button
                type="button"
                role="switch"
                aria-checked={appt.isActive}
                aria-label={td('reminderToggle')}
                className="rmd-switch"
                disabled={cancelled || update.isPending}
                onClick={() => update.mutate({ id: appt.id, patch: { isActive: !appt.isActive } })}
              >
                <span className="rmd-switch-knob" />
              </button>
            </div>
          </div>

          {appt.prep.length > 0 && (
            <section className="rmd-sec" aria-labelledby="apd-prep">
              <div className="rmd-sec-head">
                <h2 className="rmd-sec-title" id="apd-prep">{td('prepTitle')}</h2>
              </div>
              <div className="rmd-list">
                {appt.prep.map((item) => (
                  <div key={item.id} className="rmd-row">
                    <button
                      type="button"
                      className="rmd-check"
                      aria-pressed={item.done}
                      aria-label={item.text}
                      disabled={cancelled}
                      onClick={() => togglePrep.mutate({ appointment: appt, itemId: item.id, done: !item.done })}
                    >
                      <span className="rmd-check-dot">{item.done && <Icon name="check" size={14} strokeWidth={3} />}</span>
                    </button>
                    <span className="rmd-row-body">
                      <span className={clsx('rmd-row-title', item.done && 'line-through')}>{item.text}</span>
                    </span>
                  </div>
                ))}
              </div>
            </section>
          )}

          {!cancelled && when && (
            <button type="button" className="rmd-cta" onClick={addToCalendar}>
              <Icon name="calendar" size={18} strokeWidth={2} />
              {td('addToCalendar')}
            </button>
          )}
          {!cancelled && (
            <button type="button" className="btn btn-ghost" onClick={() => setConfirming(true)}>
              {td('cancel')}
            </button>
          )}
        </div>
      </div>

      <AppSheet
        open={confirming}
        onClose={() => setConfirming(false)}
        size="half"
        title={td('confirmCancel')}
      >
        <div className="del-btns">
          <button type="button" className="btn btn-primary" disabled={cancel.isPending} onClick={doCancel}>
            {td('confirmCancelYes')}
          </button>
          <button type="button" className="btn btn-soft" onClick={() => setConfirming(false)}>
            {td('confirmCancelNo')}
          </button>
          {cancel.isError && <p className="rem-form-error" role="alert">{t('saveError')}</p>}
        </div>
      </AppSheet>
    </div>
  );
}

function InfoRow({ icon, label, value }: { icon: IconName; label: string; value: string }) {
  if (!value) return null;
  return (
    <div className="rmd-row">
      <span className="rmd-tile" aria-hidden>
        <Icon name={icon} size={20} />
      </span>
      <span className="rmd-row-body">
        <span className="rmd-row-meta">{label}</span>
        <span className="rmd-row-title">{value}</span>
      </span>
    </div>
  );
}
