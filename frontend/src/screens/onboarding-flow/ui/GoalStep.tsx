'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { modeAvailable } from '@/entities/app-module';
import { RadioCardGroup, type RadioCardOption } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { useSaveOnboardingStep } from '../api/onboarding';
import type { OnboardingGoal } from '../model/flow';
import { StepScreen, type StepContext } from './StepScreen';

const META: Record<OnboardingGoal, Pick<RadioCardOption<OnboardingGoal>, 'icon' | 'iconTone'>> = {
  cycle: { icon: 'drop', iconTone: 'period' },
  ttc: { icon: 'target', iconTone: 'warm' },
  pregnancy: { icon: 'user', iconTone: 'bloom' },
  menopause: { icon: 'moon', iconTone: 'brand' },
};

function GoalForm({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow');
  const te = useTranslations('onboarding.enums.goal');
  const save = useSaveOnboardingStep();
  // Goals whose module the admin switched off (ttc, menopause) are not offered.
  const goals = (Object.keys(META) as OnboardingGoal[]).filter((g) => modeAvailable(ctx.modules, g));
  const [goal, setGoal] = useState<OnboardingGoal | null>(ctx.goal && goals.includes(ctx.goal) ? ctx.goal : null);

  const options = goals.map((value) => ({
    value,
    title: te(value),
    description: t(`goal.desc.${value}`),
    ...META[value],
  }));

  const submit = () => {
    if (!goal || save.isPending) return;
    save.mutate({ step: 'goal', body: { goal } }, { onSuccess: () => ctx.next({ goal }) });
  };

  return (
    <OnbFrame
      progress={ctx.progress}
      onBack={ctx.back}
      title={t('goal.title')}
      subtitle={t('goal.subtitle')}
      primary={{ label: t('continue'), onClick: submit, disabled: !goal, loading: save.isPending }}
    >
      <RadioCardGroup className="onb2-goals" options={options} value={goal} onChange={setGoal} label={t('goal.title')} />
      {save.isError ? <p className="onb2-error" role="alert">{t('saveError')}</p> : null}
    </OnbFrame>
  );
}

/** nbl_Onb_Goal — the answer picks the next screen (Cycle / Preg / Meno). */
export function GoalStep() {
  return <StepScreen step="goal">{(ctx) => <GoalForm ctx={ctx} />}</StepScreen>;
}
