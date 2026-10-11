'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { type JoinApplication, type JoinStatus, useJoinStatus } from '@/entities/city-service';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatTimeOfDay } from '@/shared/lib/date';
import { Icon, InfoNote, PrimaryButton, SecondaryButton, StatusPill, StepTimeline, type TimelineStep } from '@/shared/ui';

import { PhotoManager } from './PhotoManager';

function useTimeline(status: JoinStatus | undefined): TimelineStep[] {
  const t = useTranslations('cityJoin.done');
  const locale = useLocale() as Locale;
  const when = (at: string | null): string | undefined => {
    if (!at) return undefined;
    const d = new Date(at);
    return Number.isNaN(d.getTime()) ? undefined : `${formatLongDate(d, locale)} ${formatTimeOfDay(d, locale)}`;
  };
  return (status?.timeline ?? []).map((s) => {
    const hint = t(`timeline.${s.key}.hint`);
    return {
      id: s.key,
      title: t(`timeline.${s.key}.title`),
      meta: s.state === 'failed' ? t('statusRejected') : (when(s.at) ?? hint),
      state: s.state === 'done' ? 'done' : s.state === 'pending' ? 'todo' : 'current',
      tone: s.state === 'failed' ? 'bloom' : undefined,
    } satisfies TimelineStep;
  });
}

/** `/city-services/join` once the application is with the Ritme team (nbl_Dir_JoinDone): tracking code + timeline. */
export function DonePage({ app, onNew, newBusy }: { app: JoinApplication; onNew: () => void; newBusy: boolean }) {
  const t = useTranslations('cityJoin.done');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const status = useJoinStatus(app.id);
  const steps = useTimeline(status.data);
  const [copied, setCopied] = useState(false);
  const code = app.tracking_code;
  const name = app.business_name;

  const copy = () => {
    if (!code) return;
    void navigator.clipboard
      ?.writeText(code)
      .then(() => {
        setCopied(true);
        window.setTimeout(() => setCopied(false), 2000);
      })
      .catch(() => undefined);
  };

  const pill =
    app.status === 'approved' ? (
      <StatusPill tone="data" icon="check">
        {t('statusApproved')}
      </StatusPill>
    ) : app.status === 'rejected' ? (
      <StatusPill tone="bloom" icon="warning">
        {t('statusRejected')}
      </StatusPill>
    ) : null;

  return (
    <div className="cjn-stack">
      <section className="cjn-done-hero">
        <span className="cjn-done-ic">
          <Icon name="checkCircle" size={34} />
        </span>
        <h2 className="cjn-hero-title">{t('title')}</h2>
        <p className="cjn-hero-body">{name ? t('subtitle', { name }) : t('subtitleNoName')}</p>
        {pill}
        {code ? (
          <div className="cjn-code" dir="ltr">
            <span className="cjn-code-l">{t('codeLabel')}</span>
            <strong className="cjn-code-v">{code}</strong>
            <button type="button" className="cjn-link" onClick={copy}>
              <Icon name="copy" size={14} />
              {copied ? t('copied') : t('copy')}
            </button>
          </div>
        ) : null}
      </section>

      {app.review_note ? (
        <InfoNote icon="info">
          <strong>{t('reviewNote')}</strong> {app.review_note}
        </InfoNote>
      ) : null}

      {steps.length > 0 ? (
        <section className="card fld-card">
          <StepTimeline
            steps={steps}
            label={t('timelineLabel')}
            stateLabels={{ done: t('states.done'), current: t('states.current'), todo: t('states.todo') }}
            marker="dot"
            locale={locale}
          />
        </section>
      ) : null}

      {app.can_add_photos ? (
        <section className="card fld-card cjn-fields">
          <h3 className="cjn-label">{t('meanwhile')}</h3>
          <p className="cjn-hint">
            {t('morePhotos')} · {t('morePhotosHint')}
          </p>
          <PhotoManager app={app} />
        </section>
      ) : null}

      <Link href="/profile/support" className="cjn-link">
        {t('support')}
      </Link>
      <PrimaryButton onClick={onNew} loading={newBusy}>
        {t('newApp')}
      </PrimaryButton>
      <SecondaryButton onClick={() => router.push('/city-services')}>{t('toHome')}</SecondaryButton>
    </div>
  );
}
