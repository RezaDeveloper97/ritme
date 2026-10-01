'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';
import { clsx } from 'clsx';

import { Icon } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { useSaveOnboardingStep } from '../api/onboarding';
import type { OnboardingGender } from '../model/flow';
import { StepScreen, type StepContext } from './StepScreen';

const OPTIONS: readonly { value: OnboardingGender; icon: 'female' | 'male' }[] = [
  { value: 'female', icon: 'female' },
  { value: 'male', icon: 'male' },
];

function GenderForm({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow');
  const te = useTranslations('onboarding.enums.gender');
  const save = useSaveOnboardingStep();
  const [gender, setGender] = useState<OnboardingGender | null>(ctx.state.gender);

  const submit = () => {
    if (!gender || save.isPending) return;
    save.mutate({ step: 'gender', body: { gender } }, { onSuccess: () => ctx.next({ gender }) });
  };

  return (
    <OnbFrame
      progress={ctx.progress}
      onBack={ctx.back}
      title={t('gender.title')}
      subtitle={t('gender.subtitle')}
      primary={{ label: t('continue'), onClick: submit, disabled: !gender, loading: save.isPending }}
    >
      <div className="onb2-gender" role="radiogroup" aria-label={t('gender.title')}>
        {OPTIONS.map((o) => (
          <button
            key={o.value}
            type="button"
            role="radio"
            aria-checked={gender === o.value}
            className={clsx('onb2-gender-card', `is-${o.value}`)}
            onClick={() => setGender(o.value)}
          >
            <span className="onb2-gender-disc" aria-hidden>
              <Icon name={o.icon} size={30} strokeWidth={2} />
            </span>
            <span className="onb2-gender-label">{te(o.value)}</span>
          </button>
        ))}
      </div>
      <p className="onb2-note is-center">{t('gender.maleNote')}</p>
      {save.isError ? <p className="onb2-error" role="alert">{t('saveError')}</p> : null}
    </OnbFrame>
  );
}

/** nbl_Onb_Gender — a man continues to the partner-code screen. */
export function GenderStep() {
  return <StepScreen step="gender">{(ctx) => <GenderForm ctx={ctx} />}</StepScreen>;
}
