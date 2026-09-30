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
import { type Locale, useRouter } from '@/shared/i18n';
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
import {
  HeaderButton,
  Icon,
  type IconName,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  Switch,
} from '@/shared/ui';

import {
  appointmentIcsEvent,
  daysUntil,
  mapsHref,
  pregnancyWeekAt,
  reminderMoment,
  splitScheduled,
  weekdayOf,
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
        <div className="scroll rmd-screen">
          <SkyLayer />
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
              <SkeletonGroup label={t('appointmentDetail.loading')} className="rmd-form-skel">
                <Skeleton shape="card" />
                <Skeleton shape="card" />
              </SkeletonGroup>
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
  const router = useRouter();
  return (
    <ScreenHeader
      title={t('appointmentDetail.title')}
      subtitle={sub}
      onBack={() => router.push('/reminders?tab=appointments')}
      backLabel={t('back')}
      action={
        editable ? (
          <HeaderButton
            icon="pencil"
            label={t('appointmentDetail.edit')}
            onClick={() => router.push(`/reminders/appointment/${id}/edit`)}
          />
        ) : undefined
      }
    />
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
  const weekday = when ? weekdayOf(when.date) : null;
  // «در خصوص» is the appointment's own text: title, then the note (as in the artboard).
  const title = appt.title?.trim() ?? '';
  const note = appt.notes?.trim() ?? '';
  const about = title && note ? td('aboutWithNote', { title, note }) : title || note || topic;

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
      <div className="scroll rmd-screen">
        <SkyLayer />
        <DetailHeader
          id={appt.id}
          editable={!cancelled}
          sub={cancelled ? td('cancelled') : days !== null && days >= 0 ? td('daysUntil', { days }) : undefined}
        />

        <div className={clsx('rmd-body', cancelled && 'opacity-60')}>
          <section className="apd-hero">
            <div className="apd-hero-top">
              <span className="apd-pill">
                <Icon name={KIND_ICON[appt.kind]} size={14} strokeWidth={1.8} />
                {tf(`kinds.${appt.kind}`)}
              </span>
              {week !== null && (
                <span className="apd-pill">{td('pregnancyWeek', { week: formatNumber(week, locale) })}</span>
              )}
              {cancelled && <span className="apd-pill">{td('cancelled')}</span>}
            </div>
            <div className="apd-hero-main">
              <span className="apd-date" aria-hidden>
                {parts && (
                  <>
                    <span className="apd-date-day">{formatNumber(parts.day, locale)}</span>
                    <span className="apd-date-month">{monthName(parts.month, locale)}</span>
                  </>
                )}
              </span>
              <div className="apd-hero-body">
                {date && weekday && (
                  <span className="apd-eyebrow">
                    <span className="sr-only">{formatWeekdayDayMonth(date, locale)} · </span>
                    <span aria-hidden>{td('weekdayAt', { weekday: td(`weekdays.${weekday}`) })}</span>
                  </span>
                )}
                {time && (
                  <span className="apd-time" dir="ltr">
                    {time}
                  </span>
                )}
                <h2 className="apd-title">{appt.title || topic}</h2>
              </div>
            </div>
          </section>

          <div className="rmd-list">
            <InfoRow icon="user" label={td('with')} value={[appt.withWhom, appt.specialty].filter(Boolean).join(' · ')} />
            <InfoRow icon="note" label={td('about')} value={about} />
            {appt.location && (
              <div className="rmd-row">
                <span className="rmd-tile is-info" aria-hidden>
                  <Icon name={appt.kind === 'online' ? 'video' : appt.kind === 'phone' ? 'phone' : 'mapPin'} size={18} />
                </span>
                <span className="rmd-row-body">
                  <span className="rmd-row-meta">{placeLabel}</span>
                  <span className="rmd-row-title" dir="auto">{appt.location}</span>
                </span>
                {maps && (
                  <a className="rmd-sec-add rmd-pill-link" href={maps} target="_blank" rel="noopener noreferrer">
                    <span>{td('directions')}</span>
                  </a>
                )}
              </div>
            )}
            <div className="rmd-row">
              <span className="rmd-tile is-info" aria-hidden>
                <Icon name="bellRing" size={18} />
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
              <Switch
                compact
                className="rmd-hit"
                checked={appt.isActive}
                label={td('reminderToggle')}
                disabled={cancelled || update.isPending}
                onCheckedChange={(next) => update.mutate({ id: appt.id, patch: { isActive: next } })}
              />
            </div>
          </div>

          {appt.prep.length > 0 && (
            <section className="apd-prep" aria-labelledby="apd-prep">
              <h2 className="apd-prep-title" id="apd-prep">{td('prepTitle')}</h2>
              {appt.prep.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  role="checkbox"
                  className="apd-item"
                  aria-checked={item.done}
                  disabled={cancelled}
                  onClick={() => togglePrep.mutate({ appointment: appt, itemId: item.id, done: !item.done })}
                >
                  <span className="apd-box" aria-hidden>
                    {item.done && <Icon name="check" size={14} strokeWidth={3} />}
                  </span>
                  <span>{item.text}</span>
                </button>
              ))}
            </section>
          )}

          {!cancelled && (
            <div className="apd-actions">
              {when && (
                <button type="button" className="apd-cal" onClick={addToCalendar}>
                  <Icon name="calendar" size={18} strokeWidth={1.8} />
                  {td('addToCalendar')}
                </button>
              )}
              <button
                type="button"
                className={clsx('apd-cancel', !when && 'is-wide')}
                onClick={() => setConfirming(true)}
              >
                {td('cancel')}
              </button>
            </div>
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
          <button
            type="button"
            className="apd-cancel is-wide is-filled"
            disabled={cancel.isPending}
            onClick={doCancel}
          >
            {td('confirmCancelYes')}
          </button>
          <SecondaryButton onClick={() => setConfirming(false)}>{td('confirmCancelNo')}</SecondaryButton>
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
      <span className="rmd-tile is-info" aria-hidden>
        <Icon name={icon} size={18} />
      </span>
      <span className="rmd-row-body">
        <span className="rmd-row-meta">{label}</span>
        <span className="rmd-row-title">{value}</span>
      </span>
    </div>
  );
}
