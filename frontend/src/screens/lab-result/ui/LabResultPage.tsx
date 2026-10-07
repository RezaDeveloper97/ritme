'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState, type ReactNode } from 'react';

import { isBusy, type Lab, splitMarkers, useDeleteLab, useLab, useLabFeedback } from '@/entities/lab';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  IconCircle,
  InfoNote,
  PrimaryButton,
  ProgressRing,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  UrgentCard,
} from '@/shared/ui';

import { MarkerRow } from './MarkerRow';

const NORMAL_PREVIEW = 2;

function Shell({ header, children }: { header: ReactNode; children: ReactNode }) {
  return (
    <div className="view lab-screen">
      <SkyLayer />
      <div className="scroll lab-scroll">
        {header}
        <div className="lab-body">{children}</div>
      </div>
    </div>
  );
}

/** The «۲ پایین · ۱ پایین مرزی» pills of the hero. */
function countPills(lab: Lab, t: ReturnType<typeof useTranslations<'labs'>>, locale: Locale) {
  const c = lab.counts;
  return (
    [
      ['low', c.low],
      ['borderline_low', c.borderlineLow],
      ['high', c.high],
      ['borderline_high', c.borderlineHigh],
    ] as const
  )
    .filter(([, n]) => n > 0)
    .map(([k, n]) => (
      <StatusPill key={k} tone="warm">
        {t(`result.counts.${k}`, { n: formatNumber(n, locale) })}
      </StatusPill>
    ));
}

/**
 * `/labs/[id]` (nbl_Lab_Result): red flags first, then the hero (normal share
 * ring, attention count, the plain summary with its source badge), the markers
 * that need attention, the normal ones (two, then «نمایش همه»), questions for
 * the doctor, the record / doctor / assistant actions («به‌زودی» until B-N6-03,
 * B-N7), feedback and the disclaimer. Every AI text is rendered as plain text.
 */
export function LabResultPage({ id }: { id: number }) {
  const t = useTranslations('labs');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const valid = Number.isFinite(id) && id > 0;
  const lab = useLab(valid ? id : null);
  const feedback = useLabFeedback(id);
  const del = useDeleteLab(id);
  const [allNormal, setAllNormal] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const status = lab.data?.status;
  useEffect(() => {
    if (!status) return;
    if (status === 'needs_review') router.replace(`/labs/${id}/verify`);
    else if (isBusy(status) || status === 'failed') router.replace(`/labs/${id}/processing`);
  }, [status, id, router]);

  const header = (
    <ScreenHeader
      title={t('result.title')}
      subtitle={
        lab.data
          ? t('result.subtitle', { title: lab.data.title, date: formatLongDate(fromApiDate(lab.data.date), locale) })
          : undefined
      }
      onBack={() => router.push('/labs')}
      backLabel={t('common.back')}
      action={
        lab.data?.status === 'ready' ? (
          <HeaderButton icon="export" label={t('result.shareDoctor')} onClick={() => router.push('/record/export?section=checkups')} />
        ) : undefined
      }
    />
  );

  if (!valid || getApiErrorStatus(lab.error) === 404) {
    return (
      <Shell header={header}>
        <EmptyState
          icon="flask"
          title={t('common.notFoundTitle')}
          body={t('common.notFoundBody')}
          action={<PrimaryButton onClick={() => router.push('/labs')}>{t('common.toList')}</PrimaryButton>}
        />
      </Shell>
    );
  }
  if (!mounted || lab.isPending || (lab.data && lab.data.status !== 'ready')) {
    return (
      <Shell header={header}>
        <SkeletonGroup label={t('common.loading')}>
          <Skeleton shape="block" />
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }
  if (lab.isError) {
    return (
      <Shell header={header}>
        <Card className="lab-state" role="alert">
          <IconCircle icon="warning" tone="danger" size="lg" />
          <p className="lab-state-text">{t('common.loadError')}</p>
          <SecondaryButton icon="refresh" block={false} loading={lab.isFetching} onClick={() => void lab.refetch()}>
            {t('common.retry')}
          </SecondaryButton>
        </Card>
      </Shell>
    );
  }

  const data = lab.data;
  const groups = splitMarkers(data.markers);
  const rest = [...groups.normal, ...groups.unknown];
  const shownRest = allNormal ? rest : rest.slice(0, NORMAL_PREVIEW);
  const interp = data.interpretation;
  const ringShare = data.counts.total ? data.counts.normal / data.counts.total : 0;
  const open = (markerId: number) => router.push(`/labs/${id}/markers/${markerId}`);

  return (
    <Shell header={header}>
      {interp?.redFlags.map((f) => (
        <UrgentCard
          key={`${f.markerId}-${f.severity}`}
          variant={f.severity === 'urgent' ? 'card' : 'note'}
          icon="warning"
          title={f.name}
          action={
            <SecondaryButton block={false} onClick={() => open(f.markerId)}>
              {t('result.flagMore')}
            </SecondaryButton>
          }
        >
          {f.message}
        </UrgentCard>
      ))}

      <Card variant="hero" className="lab-res-hero">
        <div className="lab-res-hero-top">
          <ProgressRing
            value={ringShare}
            size={92}
            thickness={10}
            tone="data"
            label={t('result.ringLabel')}
            valueText={t('result.ringValue', { normal: formatNumber(data.counts.normal, locale), total: formatNumber(data.counts.total, locale) })}
          >
            <span className="lab-res-ring-num">
              {formatNumber(data.counts.normal, locale)}/{formatNumber(data.counts.total, locale)}
            </span>
            <span className="lab-res-ring-cap">{t('result.normal')}</span>
          </ProgressRing>
          <div className="lab-res-hero-text">
            <h2 className="lab-res-hero-title">
              {data.counts.attention > 0
                ? t('result.attentionTitle', { count: data.counts.attention, n: formatNumber(data.counts.attention, locale) })
                : t('result.allNormalTitle')}
            </h2>
            <div className="lab-res-pills">{countPills(data, t, locale)}</div>
          </div>
        </div>
        {interp?.summary ? <p className="lab-res-summary">{interp.summary}</p> : null}
        {interp ? (
          <p className="lab-res-source">
            <StatusPill tone={interp.source === 'ai' ? 'bloom' : 'neutral'} icon={interp.source === 'ai' ? 'sparkle' : 'book'}>
              {interp.source === 'ai' ? t('result.sourceAi') : t('result.sourceRules')}
            </StatusPill>
            {interp.stale ? <span className="lab-res-stale">{t('result.stale')}</span> : null}
          </p>
        ) : null}
      </Card>

      {groups.attention.length ? (
        <Card as="section" className="lab-res-card" aria-labelledby="lab-att">
          <h2 id="lab-att" className="lab-card-title">
            {t('result.attention')}
          </h2>
          <ul className="lab-mrows">
            {groups.attention.map((m) => (
              <MarkerRow key={m.id} marker={m} onOpen={() => open(m.id)} />
            ))}
          </ul>
        </Card>
      ) : null}

      {rest.length ? (
        <Card as="section" className="lab-res-card" aria-labelledby="lab-normal">
          <div className="lab-card-head">
            <h2 id="lab-normal" className="lab-card-title">
              {t('result.inRange')}
            </h2>
            <span className="lab-card-meta">{t('result.markerCount', { n: formatNumber(rest.length, locale) })}</span>
          </div>
          <ul className="lab-mrows">
            {shownRest.map((m) => (
              <MarkerRow key={m.id} marker={m} onOpen={() => open(m.id)} />
            ))}
          </ul>
          {rest.length > NORMAL_PREVIEW ? (
            <button type="button" className="lab-link is-center" aria-expanded={allNormal} onClick={() => setAllNormal((v) => !v)}>
              {allNormal ? t('result.showLess') : t('result.showAll', { n: formatNumber(rest.length, locale) })}
            </button>
          ) : null}
        </Card>
      ) : null}

      {interp?.doctorQuestions.length ? (
        <Card as="section" className="lab-res-card" aria-labelledby="lab-q">
          <h2 id="lab-q" className="lab-card-title">
            {t('result.questions')}
          </h2>
          <ul className="lab-questions">
            {interp.doctorQuestions.map((q, i) => (
              <li key={i} className="lab-question">
                <Icon name="help" size={17} strokeWidth={1.8} className="lab-question-icon" />
                <span>{q}</span>
              </li>
            ))}
          </ul>
        </Card>
      ) : null}

      <div className="lab-res-actions">
        <button type="button" className="lab-soon-btn is-primary" aria-disabled="true">
          <Icon name="stetho" size={18} />
          <span>{t('result.doctor')}</span>
          <StatusPill tone="neutral">{t('common.soon')}</StatusPill>
        </button>
        <button type="button" className="lab-soon-btn" aria-disabled="true">
          <Icon name="plus" size={18} />
          <span>{t('result.addRecord')}</span>
          <StatusPill tone="neutral">{t('common.soon')}</StatusPill>
        </button>
      </div>

      <div className="lab-ask" aria-disabled="true">
        <IconCircle icon="sparkle" tone="bloom" />
        <span className="lab-ask-text">
          <span className="lab-ask-title">{t('result.askTitle')}</span>
          <span className="lab-ask-sub">{t('result.askSub')}</span>
        </span>
        <StatusPill tone="neutral">{t('common.soon')}</StatusPill>
      </div>

      {data.source === 'upload' ? (
        <div className="lab-feedback">
          <span className="lab-feedback-q">{data.feedback ? t('result.feedbackThanks') : t('result.feedback')}</span>
          <div className="lab-feedback-btns" role="group" aria-label={t('result.feedback')}>
            {([true, false] as const).map((helpful) => (
              <button
                key={String(helpful)}
                type="button"
                className="lab-feedback-btn"
                aria-pressed={data.feedback?.helpful === helpful}
                aria-label={helpful ? t('result.helpful') : t('result.notHelpful')}
                disabled={feedback.isPending}
                onClick={() => feedback.mutate(helpful)}
              >
                <span aria-hidden>{helpful ? '👍' : '👎'}</span>
              </button>
            ))}
          </div>
        </div>
      ) : null}

      <InfoNote>{interp?.disclaimer || t('common.disclaimer')}</InfoNote>

      <div className="lab-res-manage">
        {data.editable ? (
          <SecondaryButton variant="text" icon="pencil" onClick={() => router.push(`/labs/${id}/verify`)}>
            {t('result.editValues')}
          </SecondaryButton>
        ) : null}
        {confirmDelete ? (
          <div className="lab-confirm" role="alertdialog" aria-label={t('result.deleteConfirm')}>
            <p className="lab-confirm-text">{t('result.deleteConfirm')}</p>
            <div className="lab-sheet-btns">
              <SecondaryButton block={false} onClick={() => setConfirmDelete(false)} disabled={del.isPending}>
                {t('result.keep')}
              </SecondaryButton>
              <PrimaryButton
                block={false}
                className="lab-danger"
                loading={del.isPending}
                onClick={() => del.mutate(undefined, { onSuccess: () => router.replace('/labs') })}
              >
                {t('result.delete')}
              </PrimaryButton>
            </div>
            {del.isError ? (
              <p className="lab-error" role="alert">
                {t('result.deleteError')}
              </p>
            ) : null}
          </div>
        ) : (
          <SecondaryButton variant="text" icon="trash" className="lab-delete" onClick={() => setConfirmDelete(true)}>
            {t('result.delete')}
          </SecondaryButton>
        )}
      </div>
    </Shell>
  );
}
