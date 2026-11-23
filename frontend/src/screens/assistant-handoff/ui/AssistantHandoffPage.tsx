'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { BodySkeleton, LoadError, TriagePill, useAssistantError, useConversation, useHandoff } from '@/entities/assistant';
import { DoctorAvatar, ModeTag, useBookings, useDoctorTitle, whenLabel } from '@/entities/doctor';
import { pilot } from '@/shared/pilot';
import { getApiErrorCode } from '@/shared/api';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { Card, CheckTick, EmptyState, Icon, InfoNote, PrimaryButton, ScreenHeader, SecondaryButton, SkyLayer } from '@/shared/ui';
import '@/shared/ui/styles/as.css';

/**
 * `/assistant/chat/[id]/handoff` — «ارسال به پزشک» (nbl_/nbd_v18_Handoff): pick one of HER OWN upcoming bookings; the
 * summary goes to that visit's doctor only, and only with the visit's `assistant_summaries` sharing switched on
 * (otherwise the API answers `share_consent_required` and we point her to the booking).
 */
export function AssistantHandoffPage({ id }: { id: number }) {
  const t = useTranslations('assistant');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const title = useDoctorTitle();
  const errText = useAssistantError();
  const conv = useConversation(id);
  const bookings = useBookings('upcoming');
  const handoff = useHandoff(id);
  const [picked, setPicked] = useState<number | null>(null);
  const [agree, setAgree] = useState(false);

  const usable = (bookings.data?.items ?? []).filter((b) => b.status === 'confirmed' || b.status === 'held');
  const sentTo = conv.data?.handoffs.map((h) => h.booking_id) ?? [];
  const noConsent = getApiErrorCode(handoff.error) === 'share_consent_required';
  const noSummary = conv.data && !conv.data.summary;

  let body;
  if (conv.isPending || bookings.isPending) body = <BodySkeleton rows={3} />;
  else if (conv.isError || bookings.isError || !conv.data)
    body = <div className="as-body"><LoadError onRetry={() => { void conv.refetch(); void bookings.refetch(); }} busy={conv.isFetching || bookings.isFetching} /></div>;
  else if (handoff.isSuccess)
    body = (
      <div className="as-body">
        <Card>
          <EmptyState icon="checkCircle" title={t('handoff.sentTitle')} body={t('handoff.sentBody')} action={<PrimaryButton onClick={() => router.push('/assistant')} {...pilot('assistant.open:assistant')}>{t('hub.title')}</PrimaryButton>} />
        </Card>
      </div>
    );
  else if (noSummary)
    body = (
      <div className="as-body">
        <Card><EmptyState icon="note" title={t('handoff.noSummary')} action={<PrimaryButton onClick={() => router.push(`/assistant/chat/${id}/summary`)}>{t('chat.summary')}</PrimaryButton>} /></Card>
      </div>
    );
  else
    body = (
      <div className="as-body">
        <Card className="as-preview">
          <Icon name="note" size={20} />
          <span className="as-row-main">
            <strong>{t('handoff.preview')}</strong>
            <span>{conv.data.conversation.department.name}</span>
          </span>
          {conv.data.conversation.triage_level ? <TriagePill level={conv.data.conversation.triage_level} /> : null}
        </Card>

        <h2 className="as-h2">{t('handoff.pick')}</h2>
        {usable.length === 0 ? (
          <Card>
            <EmptyState
              icon="stetho"
              title={t('handoff.noBookingTitle')}
              body={t('handoff.noBookingBody')}
              action={<PrimaryButton onClick={() => router.push('/services/doctors')} {...pilot('assistant.open:services-doctors')}>{t('handoff.findDoctor')}</PrimaryButton>}
            />
          </Card>
        ) : (
          <div className="as-list" role="radiogroup" aria-label={t('handoff.pick')}>
            {usable.map((b) => (
              <button
                key={b.id}
                type="button"
                role="radio"
                aria-checked={picked === b.id}
                className={clsx('as-pick', picked === b.id && 'is-on')}
                onClick={() => { setPicked(b.id); handoff.reset(); }}
              >
                <DoctorAvatar name={b.doctor.name} photoUrl={b.doctor.photo_url} size="md" />
                <span className="as-row-main">
                  <strong>{title(b.doctor)}</strong>
                  <span>{[b.doctor.specialty.title, whenLabel(b.date, b.time, locale)].filter(Boolean).join(' · ')}</span>
                  <ModeTag mode={b.mode} />
                </span>
                {sentTo.includes(b.id) ? <span className="as-sent">{t('handoff.alreadySent')}</span> : null}
              </button>
            ))}
          </div>
        )}

        {usable.length > 0 ? (
          <>
            <label className="as-check">
              <input type="checkbox" className="nb-check-native" checked={agree} onChange={(e) => setAgree(e.target.checked)} />
              <span className="nb-check-box" aria-hidden><CheckTick on={agree} /></span>
              <span>{t('handoff.consent')}</span>
            </label>
            <p className="as-fine">{t('handoff.revoke')}</p>
            {handoff.isError ? (
              <p className="as-error" role="alert">
                {errText(handoff.error)}
                {noConsent && picked ? <> <Link href={`/services/bookings/${picked}`} className="as-link">{t('handoff.openBooking')}</Link></> : null}
              </p>
            ) : null}
            <PrimaryButton block icon="send" loading={handoff.isPending} disabled={!picked || !agree} onClick={() => picked && handoff.mutate(picked)}>
              {t('handoff.send')}
            </PrimaryButton>
          </>
        ) : null}
        <SecondaryButton block onClick={() => router.push('/services/doctors')} {...pilot('assistant.open:services-doctors-all')}>{t('handoff.allDoctors')}</SecondaryButton>
        <InfoNote icon="shield">{t('summary.footnote')}</InfoNote>
      </div>
    );

  return (
    <div className="view as-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader title={t('handoff.title')} subtitle={t('handoff.subtitle')} onBack={() => router.push(`/assistant/chat/${id}/summary`)} backLabel={t('common.back')} />
        {body}
      </div>
    </div>
  );
}
