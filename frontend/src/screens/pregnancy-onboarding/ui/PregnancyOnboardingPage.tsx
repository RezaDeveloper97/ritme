'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  useActivatePregnancy,
  useCompleteOnboarding,
  useDatingPreview,
  useSetupCopy,
} from '@/entities/pregnancy';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  Card,
  IconCircle,
  PrimaryButton,
  ProgressSteps,
  ScreenHeader,
  SecondaryButton,
  SkyLayer,
} from '@/shared/ui';
import { WelcomePregnancy } from '@/shared/ui/illustrations';

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

/** welcome → dating basis → optional history → result: the four steps of PregFull_Setup. */
type Step = 1 | 2 | 3 | 4;
const TOTAL = 4;
/** Bundled welcome benefits (`setup.benefits.*`), same order as the seeded `pregnancy_setup/welcome` (the fallback). */
const BENEFITS = ['1', '2', '3'] as const;

/**
 * Pregnancy Setup v2 (`/pregnancy/setup`, PregFull_Setup): welcome → dating
 * basis → optional history → result from `dating-preview`. «تمومه» activates
 * pregnancy mode, submits the v1 `/pregnancy/onboarding` body, then lands on
 * `/pregnancy`. Sensitive data only leaves the form in those requests (§11).
 * Copy: the admin-edited `pregnancy_setup` texts (`GET /pregnancy/v2/setup-copy`)
 * win; the bundle is the fallback while they load or where a text is missing.
 */
export function PregnancyOnboardingPage() {
  const t = useTranslations('pregnancyV2.setup');
  const tv = useTranslations('pregnancyV2');
  const loc = useLocale() as Locale;
  const router = useRouter();

  const [step, setStep] = useState<Step>(1);
  const [dating, setDating] = useState<SetupDating>(EMPTY_DATING);
  const [history, setHistory] = useState<SetupHistory | null>(EMPTY_HISTORY);
  const [error, setError] = useState<string | null>(null);

  const copy = useSetupCopy().data;
  const preview = useDatingPreview(step === 4 ? toPreviewInput(dating, loc) : null);
  const activate = useActivatePregnancy();
  const onboard = useCompleteOnboarding();
  const submitting = activate.isPending || onboard.isPending;

  const back = () => {
    setError(null);
    if (step === 1) router.push('/profile');
    else setStep((step - 1) as Step);
  };

  const next = () => {
    setError(null);
    if (step === 2 && !isDatingComplete(dating)) return setError(t('fillRequired'));
    if (step < TOTAL) setStep((step + 1) as Step);
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

  const stepText = t('stepOf', { step: formatNumber(step, loc), total: formatNumber(TOTAL, loc) });
  const benefits = copy?.welcome.benefits.length ? copy.welcome.benefits : BENEFITS.map((k) => t(`benefits.${k}`));

  let content: React.ReactNode;
  let actions: React.ReactNode;
  if (step === 1) {
    content = (
      <>
        <div className="pgn-setup-illu">
          <WelcomePregnancy size={180} />
        </div>
        <Card as="section" className="pgn-sect" aria-labelledby="pgn-setup-welcome">
          <h2 id="pgn-setup-welcome" className="pgn-sect-title">
            {copy?.welcome.title ?? t('welcomeTitle')}
          </h2>
          <p className="pgn-body-text">{copy?.welcome.body ?? t('welcomeBody')}</p>
          <ul className="pgn-rows">
            {benefits.map((text, i) => (
              <li key={i} className="pgn-row">
                <IconCircle icon="check" tone="data" size="sm" />
                <b className="pgn-row-title">{text}</b>
              </li>
            ))}
          </ul>
        </Card>
      </>
    );
    actions = (
      <>
        <PrimaryButton onClick={() => setStep(2)}>{copy?.welcome.primary ?? t('turnOn')}</PrimaryButton>
        <SecondaryButton variant="text" block onClick={() => router.replace('/profile')}>
          {copy?.welcome.secondary ?? t('notNow')}
        </SecondaryButton>
      </>
    );
  } else if (step === 4) {
    content = <ResultStep preview={preview.data} loading={preview.isLoading} failed={preview.isError} />;
    actions = (
      <>
        <PrimaryButton loading={submitting} disabled={!preview.data} onClick={() => void finish()}>
          {submitting ? t('submitting') : (preview.data?.copy.primary ?? t('done'))}
        </PrimaryButton>
        <SecondaryButton variant="text" block disabled={submitting} onClick={() => setStep(2)}>
          {preview.data?.copy.secondary ?? t('changeBasis')}
        </SecondaryButton>
      </>
    );
  } else {
    content =
      step === 2 ? (
        <DatingStep value={dating} onChange={setDating} copy={copy} />
      ) : (
        <HistoryStep value={history ?? EMPTY_HISTORY} onChange={setHistory} copy={copy} />
      );
    actions = (
      <>
        <PrimaryButton onClick={next}>{t('continue')}</PrimaryButton>
        {step === 3 && (
          <SecondaryButton
            variant="text"
            block
            onClick={() => {
              setHistory(null);
              setStep(4);
            }}
          >
            {copy?.history.skip ?? t('skip')}
          </SecondaryButton>
        )}
      </>
    );
  }

  return (
    <div className="view pon-page pgn-screen">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={tv('today.setupCta')}
          subtitle={stepText}
          onBack={back}
          backLabel={tv('common.back')}
        />
        <div className="pgn-body is-form">
          <ProgressSteps total={TOTAL} current={step} label={stepText} className="pgn-steps" />
          {content}
          {error && (
            <p role="alert" className="pgn-error-text">
              {error}
            </p>
          )}
        </div>
      </div>
      <div className="pgn-actions">{actions}</div>
    </div>
  );
}

/** Same screen under its v2 route name. */
export { PregnancyOnboardingPage as PregnancySetupPage };
