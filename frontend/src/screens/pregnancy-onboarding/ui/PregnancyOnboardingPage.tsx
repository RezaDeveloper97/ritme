'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  useActivatePregnancy,
  useCompleteOnboarding,
  useDatingPreview,
} from '@/entities/pregnancy';
import { useDirection, useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import {
  EMPTY_DATING,
  EMPTY_HISTORY,
  isDatingComplete,
  toOnboardingInput,
  toPreviewInput,
  type SetupDating,
  type SetupHistory,
} from '../model/setup';

import { DatingStep, HistoryStep, ResultStep } from './SetupSteps';

type Phase = 'welcome' | 1 | 2 | 3;
const TOTAL = 3;

/**
 * Pregnancy Setup v2 (`/pregnancy/setup`): welcome → dating basis → optional
 * history → result from `dating-preview`. «تمومه» activates pregnancy mode,
 * submits the v1 `/pregnancy/onboarding` body, then lands on `/pregnancy`.
 * Sensitive data only leaves the form in those requests (CLAUDE.md §11).
 */
export function PregnancyOnboardingPage() {
  const t = useTranslations('pregnancyV2.setup');
  const tc = useTranslations('pregnancyV2.common');
  const loc = useLocale() as Locale;
  const isRtl = useDirection() === 'rtl';
  const router = useRouter();

  const [phase, setPhase] = useState<Phase>('welcome');
  const [dating, setDating] = useState<SetupDating>(EMPTY_DATING);
  const [history, setHistory] = useState<SetupHistory | null>(EMPTY_HISTORY);
  const [error, setError] = useState<string | null>(null);

  const preview = useDatingPreview(phase === 3 ? toPreviewInput(dating, loc) : null);
  const activate = useActivatePregnancy();
  const onboard = useCompleteOnboarding();
  const submitting = activate.isPending || onboard.isPending;

  const back = () => {
    setError(null);
    if (phase === 'welcome' || phase === 1) setPhase('welcome');
    else setPhase((phase - 1) as Phase);
  };

  const next = () => {
    setError(null);
    if (phase === 1 && !isDatingComplete(dating)) return setError(t('fillRequired'));
    if (phase === 1 || phase === 2) setPhase((phase + 1) as Phase);
  };

  const finish = async () => {
    const body = toOnboardingInput(dating, history, loc);
    if (!body || submitting) return;
    setError(null);
    try {
      await activate.mutateAsync();
      await onboard.mutateAsync(body);
      router.replace('/pregnancy');
    } catch {
      setError(t('submitError'));
    }
  };

  if (phase === 'welcome') {
    return (
      <div className="view onb-page">
        <div className="scroll onb-body">
          <div className="onb-center">
            <Icon name="heart" size={48} />
            <div className="titr onb-titr">{t('welcomeTitle')}</div>
            <p className="sub onb-center-text">{t('welcomeBody')}</p>
          </div>
        </div>
        <div className="onb-actions">
          <button className="btn btn-primary" onClick={() => setPhase(1)}>
            {t('turnOn')}
          </button>
          <button className="btn btn-ghost" onClick={() => router.replace('/profile')}>
            {t('notNow')}
          </button>
        </div>
      </div>
    );
  }

  const title = phase === 1 ? t('datingTitle') : phase === 2 ? t('historyTitle') : null;
  const body = phase === 1 ? t('datingBody') : phase === 2 ? t('historyBody') : null;

  return (
    <div className="view onb-page">
      <div className="hdr">
        <button className="iconbtn" onClick={back} aria-label={tc('back')}>
          <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
        </button>
        <span className="stepcount" aria-label={t('progressLabel')}>
          {t('step', { step: phase, total: TOTAL })}
        </span>
      </div>
      <div
        className="seg"
        role="progressbar"
        aria-label={t('progressLabel')}
        aria-valuemin={1}
        aria-valuemax={TOTAL}
        aria-valuenow={phase}
        aria-valuetext={`${formatNumber(phase, loc)} / ${formatNumber(TOTAL, loc)}`}
      >
        {Array.from({ length: TOTAL }, (_, i) => (
          <button key={i} type="button" tabIndex={-1} className={i < phase ? 'on' : undefined} aria-hidden />
        ))}
      </div>

      <div className="scroll onb-body">
        {title && (
          <div className="onb-intro">
            <div className="titr">{title}</div>
            <p className="sub onb-intro-sub">{body}</p>
          </div>
        )}

        {phase === 1 && <DatingStep value={dating} onChange={setDating} />}
        {phase === 2 && <HistoryStep value={history ?? EMPTY_HISTORY} onChange={setHistory} />}
        {phase === 3 && (
          <ResultStep preview={preview.data} loading={preview.isLoading} failed={preview.isError} />
        )}

        {error && <p className="onb-error">{error}</p>}
        <div className="onb-tail" />
      </div>

      <div className="onb-actions">
        {phase === 3 ? (
          <>
            <button
              className="btn btn-primary"
              onClick={() => void finish()}
              disabled={submitting || !preview.data}
            >
              {submitting ? t('submitting') : t('done')}
            </button>
            <button className="btn btn-ghost" onClick={() => setPhase(1)} disabled={submitting}>
              {t('changeBasis')}
            </button>
          </>
        ) : (
          <>
            <button className="btn btn-primary" onClick={next}>
              {t('continue')}
            </button>
            {phase === 2 && (
              <button
                className="btn btn-ghost"
                onClick={() => {
                  setHistory(null);
                  setPhase(3);
                }}
              >
                {t('skip')}
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}

/** Same screen under its v2 route name. */
export { PregnancyOnboardingPage as PregnancySetupPage };
