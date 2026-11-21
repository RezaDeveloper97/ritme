'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import {
  type OnboardingDating,
  type OnboardingResult,
  useActivatePregnancy,
  useCompleteOnboarding,
  usePregnancyProfile,
  useSetupCopy,
} from '@/entities/pregnancy';
import { useUpdateLifeStage, writeLifeModeHint } from '@/entities/user';
import { OnbFrame } from '@/features/auth';
import { DatingResultLine, datingResultKey } from '@/features/track-pregnancy';
import { useRouter } from '@/shared/i18n';
import { today } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { pilot } from '@/shared/pilot';
import {
  EMPTY_PREGNANCY_ANSWERS,
  PregnancyDatingForm,
  pregnancyOnboardingBody,
  type PregnancyAnswers,
} from '@/widgets/pregnancy-dating';

import { EMPTY_HISTORY, finishPregnancySetup, withHistory, type SetupHistory } from '../model/setup';

import { HistoryStep } from './HistoryStep';
import '@/shared/ui/styles/onb.css';

/** dating → optional history; the same frame and form as the sign-up pregnancy step. */
type Step = 'dating' | 'history';
const PROGRESS: Record<Step, number> = { dating: 1, history: 2 };
const TOTAL = 2;

/**
 * In-app pregnancy setup (`/pregnancy/setup`, reached from the mode switch).
 * It wears the sign-up screens (OnbFrame + the nbl_Onb_Preg dating form) so
 * turning pregnancy on later looks exactly like choosing it at sign-up, then
 * offers the optional history. «تمومه» activates pregnancy mode, submits the
 * v1 `/pregnancy/onboarding` body, stores the life-stage mode (only here —
 * stage B-3), then lands on `/pregnancy`. Sensitive data only leaves the form
 * in those requests (§11).
 */
export function PregnancyOnboardingPage() {
  const t = useTranslations('pregnancyV2.setup');
  const tv = useTranslations('pregnancyV2');
  const tf = useTranslations('onboarding.flow');
  const router = useRouter();

  const [step, setStep] = useState<Step>('dating');
  const [answers, setAnswers] = useState<PregnancyAnswers>(EMPTY_PREGNANCY_ANSWERS);
  const [history, setHistory] = useState<SetupHistory>(EMPTY_HISTORY);
  const [failed, setFailed] = useState(false);
  // The ACOG re-dating outcome of the saved scan (D-169): shown before `/pregnancy`.
  const [saved, setSaved] = useState<OnboardingDating | null>(null);

  const copy = useSetupCopy().data;
  const activate = useActivatePregnancy();
  const onboard = useCompleteOnboarding();
  const storeMode = useUpdateLifeStage();
  const queryClient = useQueryClient();
  const submitting = activate.isPending || onboard.isPending || storeMode.isPending;
  const body = pregnancyOnboardingBody(answers, today());

  // Already pregnant (an active, onboarded pregnancy profile): this wizard would
  // re-date a running pregnancy, so it hands over to `/pregnancy` (stage smoke
  // B-3). Decided once, on the first answer — never mid-wizard, where «تمومه»
  // itself activates the profile before submitting the new dates.
  const existing = usePregnancyProfile();
  // The profile query is token-gated (off on the server): decide only after
  // mount so the server and first client render agree (no hydration mismatch).
  const mounted = useMounted();
  const guard = useRef<'pending' | 'stay' | 'leave'>('pending');
  if (mounted && guard.current === 'pending' && (existing.data !== undefined || existing.isError || existing.fetchStatus === 'idle')) {
    const p = existing.data?.profile;
    guard.current = p?.pregnancyMode && p.onboardingCompleted ? 'leave' : 'stay';
  }
  const leaving = guard.current === 'leave';
  useEffect(() => {
    if (leaving) router.replace('/pregnancy');
  }, [leaving, router]);

  const finish = async (withHistoryAnswers: boolean) => {
    if (!body || submitting) return;
    setFailed(false);
    try {
      let result: OnboardingResult | null = null;
      await finishPregnancySetup({
        activate: () => activate.mutateAsync(),
        onboard: async () => {
          result = await onboard.mutateAsync(withHistory(body, withHistoryAnswers ? history : null));
        },
        storeMode: () => storeMode.mutateAsync({ mode: 'pregnancy' }),
      });
      // A mode reshapes the whole app (nav, home, messages): refetch everything.
      writeLifeModeHint('pregnancy');
      void queryClient.invalidateQueries();
      // A scan saved next to an LMP was compared by the server: say what it decided before leaving.
      const dating = (result as OnboardingResult | null)?.dating ?? null;
      if (datingResultKey(dating)) setSaved(dating);
      else router.replace('/pregnancy');
    } catch {
      setFailed(true);
    }
  };

  if (guard.current === 'pending' || leaving) return <OnbFrame privacyNote={false} />;

  if (saved) {
    return (
      <OnbFrame
        title={tv('today.setupCta')}
        privacyNote={false}
        primary={{ label: t('continue'), onClick: () => router.replace('/pregnancy'), pilot: pilot('preg.setup-continue') }}
      >
        <DatingResultLine dating={saved} />
      </OnbFrame>
    );
  }

  const error = failed ? (
    <p className="onb2-error" role="alert">
      {t('submitError')}
    </p>
  ) : null;

  if (step === 'dating') {
    return (
      <OnbFrame
        progress={{ current: PROGRESS.dating, total: TOTAL }}
        onBack={() => router.replace('/profile')}
        title={tf('pregnancy.title')}
        subtitle={tf('pregnancy.subtitle')}
        primary={{ label: tf('continue'), onClick: () => setStep('history'), disabled: !body, pilot: pilot('preg.setup-next') }}
      >
        <PregnancyDatingForm value={answers} onChange={setAnswers} pilotArea="preg.setup" />
        {error}
      </OnbFrame>
    );
  }

  return (
    <OnbFrame
      progress={{ current: PROGRESS.history, total: TOTAL }}
      onBack={() => {
        setFailed(false);
        setStep('dating');
      }}
      onSkip={submitting ? undefined : () => void finish(false)}
      skipPilot={pilot('preg.setup-skip')}
      title={copy?.history.title ?? t('historyTitle')}
      subtitle={copy?.history.body ?? t('historyBody')}
      primary={{
        label: submitting ? t('submitting') : t('done'),
        onClick: () => void finish(true),
        disabled: !body,
        loading: submitting,
        pilot: pilot('preg.setup-finish'),
      }}
    >
      <HistoryStep value={history} onChange={setHistory} copy={copy} />
      {error}
    </OnbFrame>
  );
}

/** Same screen under its v2 route name. */
export { PregnancyOnboardingPage as PregnancySetupPage };
