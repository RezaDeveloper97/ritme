'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useActivatePregnancy, useCompleteOnboarding } from '@/entities/pregnancy';
import { OnbFrame } from '@/features/auth';
import { today } from '@/shared/lib/date';
import {
  EMPTY_PREGNANCY_ANSWERS,
  PregnancyDatingForm,
  pregnancyOnboardingBody,
  type PregnancyAnswers,
} from '@/widgets/pregnancy-dating';

import { StepScreen, type StepContext } from './StepScreen';

function PregnancyForm({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow');
  const activate = useActivatePregnancy();
  const onboard = useCompleteOnboarding();
  const [a, setA] = useState<PregnancyAnswers>(EMPTY_PREGNANCY_ANSWERS);
  const body = pregnancyOnboardingBody(a, today());
  const pending = activate.isPending || onboard.isPending;
  const failed = activate.isError || onboard.isError;

  const submit = async () => {
    if (!body || pending) return;
    try {
      // Same two calls the legacy setting-up screen made: switch the mode on,
      // then date the pregnancy.
      await activate.mutateAsync();
      await onboard.mutateAsync(body);
      ctx.next();
    } catch {
      /* the error line below shows; the user can retry */
    }
  };

  return (
    <OnbFrame
      progress={ctx.progress}
      onBack={ctx.back}
      onSkip={() => ctx.next()}
      title={t('pregnancy.title')}
      subtitle={t('pregnancy.subtitle')}
      primary={{ label: t('continue'), onClick: () => void submit(), disabled: !body, loading: pending }}
    >
      <PregnancyDatingForm value={a} onChange={setA} />
      {failed ? <p className="onb2-error" role="alert">{t('saveError')}</p> : null}
    </OnbFrame>
  );
}

/** nbl_Onb_Preg — dates the pregnancy (POST /pregnancy/activate + /pregnancy/onboarding). */
export function PregnancyStep() {
  return <StepScreen step="pregnancy">{(ctx) => <PregnancyForm ctx={ctx} />}</StepScreen>;
}
