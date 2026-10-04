'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { deriveCycleSchedule, hasFertileWindow, useCycleToday } from '@/entities/cycle';
import { userKeys } from '@/entities/user';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth } from '@/shared/lib/date';
import { clearOnboardingPending, getAuthToken, setAuthToken } from '@/shared/session';
import { Icon, IconCircle, Skeleton, Switch } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { useBeforePeriodReminder, useFinishOnboarding, useSetBeforePeriodReminder } from '../api/onboarding';
import { FLOW_ROUTES, isGoal, landingRoute } from '../model/flow';
import type { OnboardingState } from '../model/state';
import { StepScreen, type StepContext } from './StepScreen';

const LIFE_MODES = ['cycle', 'ttc', 'pregnancy', 'postpartum', 'menopause', 'teen'] as const;

/** The 422 of POST /onboarding/complete keys its errors by the missing step. */
function missingStepRoute(error: unknown): string | null {
  const errors = (error as { response?: { data?: { errors?: Record<string, unknown> } } })?.response?.data?.errors;
  if (!errors) return null;
  if ('name' in errors) return FLOW_ROUTES.name;
  if ('gender' in errors) return FLOW_ROUTES.gender;
  if ('goal' in errors) return FLOW_ROUTES.goal;
  return null;
}

function Row({ dot, label, value }: { dot: 'period' | 'warm' | 'brand'; label: string; value: React.ReactNode }) {
  return (
    <div className="onb2-ready-row">
      <span className={`onb2-dot is-${dot}`} aria-hidden />
      <span className="onb2-ready-label">{label}</span>
      <b className="onb2-ready-value">{value}</b>
    </div>
  );
}

/** Next period + fertile window from the engine (cycle and TTC paths). */
function CycleRows() {
  const t = useTranslations('onboarding.flow.ready');
  const locale = useLocale() as Locale;
  const today = useCycleToday();
  if (today.isPending) return <Skeleton width="medium" />;
  const schedule = today.data ? deriveCycleSchedule(today.data.cycleView ?? null, today.data.calculation ?? null) : null;
  if (!schedule) return null;
  return (
    <>
      <Row dot="period" label={t('nextPeriod')} value={t('about', { date: formatDayMonth(schedule.nextPeriodStart, locale) })} />
      {hasFertileWindow(schedule) ? (
        <Row
          dot="warm"
          label={t('fertile')}
          value={t('range', {
            from: formatDayMonth(schedule.fertileStart, locale),
            to: formatDayMonth(schedule.fertileEnd, locale),
          })}
        />
      ) : null}
    </>
  );
}

function ReminderCard() {
  const t = useTranslations('onboarding.flow.ready');
  const reminder = useBeforePeriodReminder(true);
  const setReminder = useSetBeforePeriodReminder();
  if (reminder.isPending || !reminder.data) return null;
  const on = setReminder.isPending ? (setReminder.variables ?? reminder.data.enabled) : reminder.data.enabled;
  return (
    <div className="nb-card onb2-remind">
      <IconCircle icon="bell" tone="period" size="lg" />
      <div className="onb2-remind-text">
        <div id="onb2-remind-t" className="onb2-remind-title">{t('reminder')}</div>
        <div id="onb2-remind-d" className="onb2-remind-desc">{t('reminderDesc', { days: reminder.data.daysBefore })}</div>
      </div>
      <Switch
        checked={on}
        onCheckedChange={(next) => setReminder.mutate(next)}
        labelledBy="onb2-remind-t"
        describedBy="onb2-remind-d"
      />
    </div>
  );
}

function ReadyView({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow');
  const tm = useTranslations('onboarding.enums.lifeMode');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const finish = useFinishOnboarding();
  const queryClient = useQueryClient();
  const started = useRef(false);
  const [leaving, setLeaving] = useState(false);
  // Tracked locally rather than through the mutation's observer: under
  // StrictMode's effect replay a mutate() fired from a mount effect never
  // reports back to the re-subscribed observer and stays "pending".
  const [result, setResult] = useState<{ status: 'pending' | 'done' | 'error'; state?: OnboardingState; error?: unknown }>({
    status: 'pending',
  });

  // Mark registration finished as soon as the summary shows (idempotent); a
  // 422 sends the user to the step that is still missing.
  const run = () => {
    setResult({ status: 'pending' });
    finish
      .mutateAsync()
      .then((state) => {
        // Registration is complete on the server: drop the resume marker now, not only on «ورود», so
        // opening another app route from here isn't sent back to the flow by the middleware (stage
        // B-6), and refetch the cached user (name / gender / completed) the app guards read.
        clearOnboardingPending();
        void queryClient.invalidateQueries({ queryKey: userKeys.all });
        setResult({ status: 'done', state });
      })
      .catch((error: unknown) => {
        const route = getApiErrorStatus(error) === 422 ? missingStepRoute(error) : null;
        if (route) router.replace(route);
        else setResult({ status: 'error', error });
      });
  };
  useEffect(() => {
    if (started.current) return;
    started.current = true;
    run();
    // Run once on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const state = result.state ?? ctx.state;
  const isMale = state.gender === 'male';
  const goal = isGoal(state.goal) ? state.goal : null;
  const cycleLike = !isMale && (goal === 'cycle' || goal === 'ttc' || goal === null);
  const mode = state.life_stage.mode;
  const modeLabel = (LIFE_MODES as readonly string[]).includes(mode) ? tm(mode as (typeof LIFE_MODES)[number]) : tm('cycle');

  const enter = () => {
    if (result.status !== 'done') return;
    setLeaving(true);
    // Registration is done: drop the resume marker, re-assert the session
    // cookie and cross into the app with a full document load (an App Router
    // transition could replay a redirect cached while signed out).
    clearOnboardingPending();
    const token = getAuthToken();
    if (token) setAuthToken(token);
    window.location.replace(`/${locale}${landingRoute(isMale ? null : goal, mode === 'pregnancy', isMale ? 'male' : null)}`);
  };

  return (
    <OnbFrame
      className="onb2-ready"
      privacyNote={false}
      primary={{
        label: t('ready.enter'),
        onClick: result.status === 'error' ? run : enter,
        disabled: result.status === 'pending',
        loading: result.status === 'pending' || leaving,
      }}
    >
      <div className="onb2-ready-ring" aria-hidden>
        <Icon name="check" size={56} strokeWidth={2.6} />
      </div>
      <h1 className="onb2-title is-center">{t(isMale ? 'ready.titleMale' : 'ready.title')}</h1>
      <p className="onb2-sub is-center">{t(isMale ? 'ready.subtitleMale' : 'ready.subtitle')}</p>

      <div className="nb-card onb2-ready-card">
        {cycleLike ? <CycleRows /> : null}
        <Row dot="brand" label={t('ready.mode')} value={isMale ? t('ready.partnerMode') : modeLabel} />
      </div>
      {cycleLike && result.status === 'done' ? <ReminderCard /> : null}
      {result.status === 'error' ? (
        <p className="onb2-error is-center" role="alert">{t('saveError')}</p>
      ) : null}
    </OnbFrame>
  );
}

/** nbl_Onb_Ready — finishes onboarding and enters the app on the mode's home. */
export function ReadyStep() {
  return <StepScreen step="ready">{(ctx) => <ReadyView ctx={ctx} />}</StepScreen>;
}
