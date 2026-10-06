'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef } from 'react';

import {
  labHref,
  nextPollDelay,
  PROCESS_STEPS,
  ringValue,
  stepStates,
  useLabStatus,
  useRefreshLab,
} from '@/entities/lab';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { Card, EmptyState, Icon, IconCircle, PrimaryButton, ProgressRing, SecondaryButton, SkyLayer } from '@/shared/ui';

/**
 * `/labs/[id]/processing` (nbl_Lab_Processing): polls `/labs/{id}/status`
 * (fast, then slower; paused while hidden; gives up after ~10 min) and shows the
 * ring + the four steps. On `needs_review` it hands over to verify, on `ready`
 * to the result; `failed` shows the server's reason with «بارگذاری دوباره».
 * The user may leave at any time — the lab keeps going on the server and the
 * history row on `/labs` shows where it is.
 */
export function LabProcessingPage({ id }: { id: number }) {
  const t = useTranslations('labs');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const valid = Number.isFinite(id) && id > 0;
  const status = useLabStatus(valid ? id : null);
  const refresh = useRefreshLab();
  const handed = useRef(false);
  const data = status.data;

  useEffect(() => {
    if (!data || handed.current) return;
    if (data.status === 'needs_review' || data.status === 'ready') {
      handed.current = true;
      void refresh(id)
        .catch(() => undefined)
        .then(() => router.replace(labHref(id, data.status)));
    }
  }, [data, id, refresh, router]);

  const leave = () => router.push('/labs');

  if (!valid || getApiErrorStatus(status.error) === 404) {
    return (
      <div className="view lab-screen lab-proc">
        <SkyLayer />
        <div className="scroll lab-scroll lab-proc-scroll">
          <EmptyState
            icon="flask"
            title={t('common.notFoundTitle')}
            body={t('common.notFoundBody')}
            action={<PrimaryButton onClick={leave}>{t('common.toList')}</PrimaryButton>}
          />
        </div>
      </div>
    );
  }

  if (data?.status === 'failed') {
    return (
      <div className="view lab-screen lab-proc">
        <SkyLayer />
        <div className="scroll lab-scroll lab-proc-scroll">
          <Card className="lab-state" role="alert">
            <IconCircle icon="warning" tone="danger" size="lg" />
            <p className="lab-state-title">{t('processing.failedTitle')}</p>
            <p className="lab-state-text">{data.errorMessage ?? t('processing.failedBody')}</p>
            <PrimaryButton icon="export" onClick={() => router.push('/labs/new')}>
              {t('processing.again')}
            </PrimaryButton>
            <SecondaryButton variant="text" onClick={leave}>
              {t('common.toList')}
            </SecondaryButton>
          </Card>
        </div>
      </div>
    );
  }

  const stage = data?.stage ?? 'queued';
  const steps = stepStates(stage);
  const value = ringValue(stage, data?.progress ?? 0);
  const percent = formatNumber(Math.round(value * 100), locale);
  const gaveUp = !!data && nextPollDelay(data.status, status.polls) === false && data.status !== 'needs_review' && data.status !== 'ready';
  const offline = status.isError && !data;

  return (
    <div className="view lab-screen lab-proc">
      <SkyLayer />
      <div className="scroll lab-scroll lab-proc-scroll">
        <div className="lab-proc-ring">
          <ProgressRing value={mounted ? value : 0} size={184} thickness={14} label={t('processing.ringLabel')} valueText={t('processing.percent', { p: percent })}>
            <span className="lab-proc-num">{t('processing.percent', { p: percent })}</span>
            <span className="lab-proc-cap">{t('processing.analyzing')}</span>
          </ProgressRing>
        </div>
        <h1 className="lab-proc-title">{t('processing.title')}</h1>
        <Card className="lab-proc-steps">
          <ol className="lab-steps" aria-label={t('processing.stepsLabel')}>
            {PROCESS_STEPS.map((step) => {
              const s = steps[step];
              return (
                <li key={step} className={clsx('lab-step', `is-${s}`)} aria-current={s === 'current' ? 'step' : undefined}>
                  <span className="lab-step-dot" aria-hidden>
                    {s === 'done' ? <Icon name="check" size={16} strokeWidth={3} /> : null}
                  </span>
                  <span className="lab-step-text">
                    {step === 'extract' && (data?.markerCount ?? 0) > 0
                      ? t('processing.steps.extractN', { n: formatNumber(data!.markerCount, locale) })
                      : t(`processing.steps.${step}`)}
                  </span>
                  <span className="sr-only">{t(`processing.state.${s}`)}</span>
                </li>
              );
            })}
          </ol>
        </Card>
        {offline || gaveUp ? (
          <div className="lab-proc-retry" role="status">
            <p className="lab-proc-note">{offline ? t('processing.offline') : t('processing.slow')}</p>
            <SecondaryButton icon="refresh" block={false} loading={status.isFetching} onClick={() => void status.refetch()}>
              {t('common.retry')}
            </SecondaryButton>
          </div>
        ) : null}
        <p className="lab-proc-note">{t('processing.leaveNote')}</p>
        <div className="lab-proc-leave">
          <SecondaryButton onClick={leave}>{t('processing.leave')}</SecondaryButton>
        </div>
      </div>
    </div>
  );
}
