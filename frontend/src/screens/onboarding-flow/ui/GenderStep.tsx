'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { type IconName, RadioCardGroup, type Tone } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { useSaveOnboardingStep } from '../api/onboarding';
import type { OnboardingGender } from '../model/flow';
import { StepScreen, type StepContext } from './StepScreen';
import '@/shared/ui/styles/onb.css';
import '@/shared/ui/styles/cmp.css';

/** companion-v2 `ob-gender`: «ترجیح می‌دهم نگویم» takes the full (woman-path) experience. */
const OPTIONS: readonly { value: OnboardingGender; icon: IconName; tone: Tone }[] = [
  { value: 'female', icon: 'female', tone: 'bloom' },
  { value: 'male', icon: 'male', tone: 'data' },
  { value: 'unspecified', icon: 'moreH', tone: 'brand' },
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
      <RadioCardGroup
        label={t('gender.title')}
        className="cmp2-gender"
        value={gender}
        onChange={setGender}
        options={OPTIONS.map((o) => ({
          value: o.value,
          title: te(o.value),
          description: t(`gender.desc.${o.value}`),
          icon: o.icon,
          iconTone: o.tone,
        }))}
      />
      <p className="onb2-note is-center">{t('gender.note')}</p>
      {save.isError ? <p className="onb2-error" role="alert">{t('saveError')}</p> : null}
    </OnbFrame>
  );
}

/** nbl_Onb_Gender (companion-v2 `ob-gender`) — a man continues to the partner-code screen; «ترجیح می‌دهم نگویم» like a woman. */
export function GenderStep() {
  return <StepScreen step="gender">{(ctx) => <GenderForm ctx={ctx} />}</StepScreen>;
}
