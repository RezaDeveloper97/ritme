'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { Icon } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { useSaveOnboardingStep } from '../api/onboarding';
import { StepScreen, type StepContext } from './StepScreen';

const MAX_NAME = 255;

function NameForm({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow');
  const save = useSaveOnboardingStep();
  const [name, setName] = useState(ctx.state.name ?? '');
  const trimmed = name.trim();

  const submit = () => {
    if (!trimmed || save.isPending) return;
    save.mutate({ step: 'name', body: { name: trimmed } }, { onSuccess: () => ctx.next() });
  };

  return (
    <OnbFrame
      progress={ctx.progress}
      onBack={ctx.back}
      title={t('name.title')}
      subtitle={t('name.subtitle')}
      primary={{ label: t('continue'), onClick: submit, disabled: !trimmed, loading: save.isPending }}
    >
      <label className="onb2-label" htmlFor="onb2-name">
        {t('name.label')}
      </label>
      <input
        id="onb2-name"
        className="onb2-input"
        autoComplete="given-name"
        maxLength={MAX_NAME}
        placeholder={t('name.placeholder')}
        value={name}
        onChange={(e) => setName(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && submit()}
      />
      <div className="nb-card onb2-hint">
        <Icon name="lock" size={18} className="onb2-hint-icon" />
        <span>{t('name.hint')}</span>
      </div>
      {save.isError ? <p className="onb2-error" role="alert">{t('saveError')}</p> : null}
    </OnbFrame>
  );
}

/** nbl_Onb_Name. */
export function NameStep() {
  return <StepScreen step="name">{(ctx) => <NameForm ctx={ctx} />}</StepScreen>;
}
