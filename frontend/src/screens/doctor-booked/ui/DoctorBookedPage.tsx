'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useState } from 'react';

import { useModuleEnabled } from '@/entities/app-module';
import {
  BodySkeleton,
  CONSENT_SCOPES,
  LoadError,
  SlotPicker,
  startsAtParam,
  useBooking,
  useDoctorTitle,
  useModeLabel,
  useTelemedError,
  whenLabel,
  type Booking,
  type ConsentScope,
  type PickedSlot,
} from '@/entities/doctor';
import { useCancelBooking, useRescheduleBooking, useUpdateConsent } from '@/features/book-visit';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import { Card, EmptyState, Icon, InfoNote, ListGroup, ListRow, PrimaryButton, ScreenHeader, SecondaryButton, SkyLayer, StatusPill, Switch } from '@/shared/ui';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/dr.css';

/**
 * `/services/bookings/[id]` — booked (nbl_/nbd_v17_Booked) and the booking's detail: confirmation hero, next steps
 * (reminders, chat), the video-link note, per-scope sharing toggles (self visits), free reschedule / cancel until
 * 2 h before (the API answers `cancel_window_closed` after that).
 */
export function DoctorBookedPage({ bookingId }: { bookingId: number }) {
  const t = useTranslations('doctors');
  const remindersOn = useModuleEnabled('reminders');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const fresh = useSearchParams().get('new') === '1';
  const title = useDoctorTitle();
  const modeLabel = useModeLabel();
  const q = useBooking(bookingId);
  const back = () => router.push('/services/bookings');

  const shell = (body: React.ReactNode) => (
    <div className="view dr-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('booked.header')} onBack={back} backLabel={t('common.back')} />
        {body}
        <div className="page-tail" />
      </div>
    </div>
  );

  if (q.isPending) return shell(<BodySkeleton rows={3} />);
  if (q.isError) {
    return shell(
      <div className="dr-body">
        {getApiErrorStatus(q.error) === 404 ? (
          <Card>
            <EmptyState icon="calendar" title={t('booked.notFound')} action={<PrimaryButton onClick={back}>{t('bookings.title')}</PrimaryButton>} />
          </Card>
        ) : (
          <LoadError onRetry={() => void q.refetch()} busy={q.isFetching} />
        )}
      </div>,
    );
  }
  const b = q.data;
  const active = b.status === 'confirmed';
  const heroTitle = fresh && active ? t('booked.created') : t(`status.${b.status}`);
  const who = b.for_whom === 'child' ? b.child?.name : b.for_whom === 'other' ? b.patient_name : null;

  return shell(
    <div className="dr-body">
      <section className="dr-hero">
        <span className={`dr-check ${active || b.status === 'completed' ? 'is-ok' : 'is-off'}`}>
          <Icon name={active || b.status === 'completed' ? 'check' : 'x'} size={44} strokeWidth={2.2} />
        </span>
        <h2 className="dr-hero-title">{heroTitle}</h2>
        <p className="dr-hero-sub">{t('booked.with', { mode: modeLabel(b.mode), name: title(b.doctor) })}</p>
        <p className="dr-hero-sub">{`${whenLabel(b.date, b.time, locale)} · ${t('profile.minutes', { count: formatNumber(b.duration_minutes, locale) })}`}</p>
        {who ? <StatusPill tone="brand">{t('booked.forWho', { name: who })}</StatusPill> : null}
      </section>

      {active || b.status === 'completed' ? (
        <>
          <ListGroup>
            {remindersOn ? (
              <ListRow icon="calendar" iconTone="brand" title={t('booked.reminders')} onClick={() => router.push('/reminders')} pilot={pilot('doctor-booked.reminders')} />
            ) : null}
            <ListRow icon="chat" iconTone="warm" title={t('booked.chat')} onClick={() => router.push(`/services/bookings/${b.id}/chat`)} pilot={pilot('doctor-booked.chat')} />
          </ListGroup>
          {active && b.mode === 'video' ? (
            <Card className="dr-note-card">
              <span className="dr-share-icon nb-tone-data"><Icon name="video" size={20} /></span>
              <p>{t('booked.videoNote')}</p>
            </Card>
          ) : null}
          {active && b.for_whom === 'self' ? <ConsentCard booking={b} /> : null}
          {active ? <Manage booking={b} /> : null}
        </>
      ) : (
        <InfoNote>{t(`booked.${b.status === 'held' ? 'heldNote' : 'closedNote'}`)}</InfoNote>
      )}
      <PrimaryButton onClick={() => router.push('/home')} {...pilot('doctor-booked.home')}>{t('booked.home')}</PrimaryButton>
    </div>,
  );
}

function ConsentCard({ booking }: { booking: Booking }) {
  const t = useTranslations('doctors');
  const errorText = useTelemedError();
  const update = useUpdateConsent(booking.id);
  return (
    <Card as="section" className="dr-form" aria-label={t('book.share')}>
      <h2 className="dr-h2">{t('book.share')}</h2>
      {CONSENT_SCOPES.map((s: ConsentScope) => (
        <div key={s} className="dr-share-row">
          <span className="dr-share-text">
            <strong id={`bk-${s}`}>{t(`consent.${s}.title`)}</strong>
            <span>{t(`consent.${s}.hint`)}</span>
          </span>
          <Switch checked={booking.consent[s] === true} disabled={update.isPending} onCheckedChange={(v) => update.mutate({ [s]: v })} labelledBy={`bk-${s}`} pilot={pilot(`doctor-booked.consent:${s}`)} />
        </div>
      ))}
      {update.isError ? <p role="alert" className="dr-error">{errorText(update.error)}</p> : null}
      <p className="dr-fine">{t('book.shareNote')}</p>
    </Card>
  );
}

function Manage({ booking }: { booking: Booking }) {
  const t = useTranslations('doctors');
  const router = useRouter();
  const errorText = useTelemedError();
  const cancel = useCancelBooking(booking.id);
  const reschedule = useRescheduleBooking(booking.id);
  const [panel, setPanel] = useState<'none' | 'cancel' | 'move'>('none');
  const [slot, setSlot] = useState<PickedSlot | null>(null);
  const err = cancel.error ?? reschedule.error;

  return (
    <Card as="section" className="dr-form" aria-label={t('manage.title')}>
      <h2 className="dr-h2">{t('manage.title')}</h2>
      {panel === 'none' ? (
        <div className="dr-actions">
          {booking.can_reschedule ? <SecondaryButton icon="calendar" onClick={() => setPanel('move')} {...pilot('doctor-booked.reschedule')}>{t('manage.reschedule')}</SecondaryButton> : null}
          {booking.can_cancel ? <SecondaryButton icon="x" onClick={() => setPanel('cancel')}>{t('manage.cancel')}</SecondaryButton> : null}
          {!booking.can_cancel && !booking.can_reschedule ? <p className="dr-fine">{t('errors.cancel_window_closed')}</p> : null}
        </div>
      ) : null}
      {panel === 'cancel' ? (
        <div className="dr-actions">
          <p className="dr-fine">{t('manage.cancelConfirm')}</p>
          <PrimaryButton loading={cancel.isPending} onClick={() => cancel.mutate(undefined, { onSuccess: () => router.replace('/services/bookings') })}>
            {t('manage.cancelYes')}
          </PrimaryButton>
          <SecondaryButton onClick={() => setPanel('none')}>{t('manage.keep')}</SecondaryButton>
        </div>
      ) : null}
      {panel === 'move' ? (
        <div className="dr-actions">
          <SlotPicker doctorId={booking.doctor.id} mode={booking.mode} value={slot} onChange={setSlot} />
          <PrimaryButton
            disabled={!slot}
            {...pilot('doctor-booked.move-confirm')}
            loading={reschedule.isPending}
            onClick={() => slot && reschedule.mutate(startsAtParam(slot.date, slot.time), { onSuccess: () => { setPanel('none'); setSlot(null); } })}
          >
            {t('manage.moveConfirm')}
          </PrimaryButton>
          <SecondaryButton onClick={() => setPanel('none')}>{t('manage.keep')}</SecondaryButton>
        </div>
      ) : null}
      {err ? <p role="alert" className="dr-error">{errorText(err)}</p> : null}
    </Card>
  );
}
