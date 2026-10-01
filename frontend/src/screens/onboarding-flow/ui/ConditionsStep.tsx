'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { ChipGroup, Icon, PillChip } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { useSaveOnboardingStep } from '../api/onboarding';
import { CHRONIC_ILLNESSES, GYN_CONDITIONS, MEDICATIONS, toggleListItem } from '../model/state';
import { StepScreen, type StepContext } from './StepScreen';

interface GroupProps<T extends string> {
  title: string;
  noneLabel: string;
  items: readonly T[];
  itemLabel: (item: T) => string;
  value: string[] | null;
  onChange: (next: string[]) => void;
}

/** One card: a «هیچ‌کدام» chip that excludes the rest, then the items (many-of-many). */
function ConditionGroup<T extends string>({ title, noneLabel, items, itemLabel, value, onChange }: GroupProps<T>) {
  return (
    <section className="nb-card onb2-cond">
      <h2 className="onb2-panel-title">{title}</h2>
      <ChipGroup label={title}>
        <PillChip mode="multi" pressed={value !== null && value.length === 0} onPressedChange={() => onChange(toggleListItem(value, 'none'))}>
          {noneLabel}
        </PillChip>
        {items.map((item) => (
          <PillChip key={item} mode="multi" pressed={value?.includes(item) ?? false} onPressedChange={() => onChange(toggleListItem(value, item))}>
            {itemLabel(item)}
          </PillChip>
        ))}
      </ChipGroup>
    </section>
  );
}

function ConditionsForm({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow');
  const te = useTranslations('onboarding.enums');
  const save = useSaveOnboardingStep();
  const saved = ctx.state.conditions;
  // null = never touched (sent as skipped), [] = «هیچ‌کدام».
  const [chronic, setChronic] = useState<string[] | null>(saved.chronic_illnesses);
  const [gyn, setGyn] = useState<string[] | null>(saved.gyn_conditions);
  const [meds, setMeds] = useState<string[] | null>(saved.medications);

  const submit = () => {
    if (save.isPending) return;
    save.mutate(
      { step: 'conditions', body: { chronic_illnesses: chronic, gyn_conditions: gyn, medications: meds } },
      { onSuccess: () => ctx.next() },
    );
  };

  return (
    <OnbFrame
      progress={ctx.progress}
      onBack={ctx.back}
      onSkip={() => ctx.next()}
      title={t('conditions.title')}
      subtitle={t('conditions.subtitle')}
      primary={{ label: t('continue'), onClick: submit, loading: save.isPending }}
      secondary={{ label: t('notNow'), onClick: () => ctx.next() }}
    >
      <ConditionGroup
        title={t('conditions.chronic')}
        noneLabel={t('conditions.none')}
        items={CHRONIC_ILLNESSES}
        itemLabel={(i) => te(`chronicIllness.${i}`)}
        value={chronic}
        onChange={setChronic}
      />
      <ConditionGroup
        title={t('conditions.gyn')}
        noneLabel={t('conditions.none')}
        items={GYN_CONDITIONS}
        itemLabel={(i) => te(`gynCondition.${i}`)}
        value={gyn}
        onChange={setGyn}
      />
      <ConditionGroup
        title={t('conditions.meds')}
        noneLabel={t('conditions.noMeds')}
        items={MEDICATIONS}
        itemLabel={(i) => te(`medication.${i}`)}
        value={meds}
        onChange={setMeds}
      />
      <div className="nb-card onb2-hint">
        <Icon name="lock" size={16} className="onb2-hint-icon" />
        <span>{t('conditions.disclaimer')}</span>
      </div>
      {save.isError ? <p className="onb2-error" role="alert">{t('saveError')}</p> : null}
    </OnbFrame>
  );
}

/** nbl_Onb_Conditions — chronic illnesses, gynaecological conditions, medication / contraception. */
export function ConditionsStep() {
  return <StepScreen step="conditions">{(ctx) => <ConditionsForm ctx={ctx} />}</StepScreen>;
}
