'use client';

import { useTranslations } from 'next-intl';

import { usePregnancyEnums, type SetupCopy } from '@/entities/pregnancy';
import { pilot, pilotOption } from '@/shared/pilot';
import { ChipGroup, Icon, PillChip } from '@/shared/ui';

import { noneFirst, toggleCondition, type SetupHistory } from '../model/setup';

const BLOOD_TYPES = ['A', 'B', 'AB', 'O'];
const RH_FACTORS = ['positive', 'negative'];
const CONDITIONS = ['chronic_hypertension', 'diabetes', 'hypothyroidism', 'hyperthyroidism', 'none'];

type DynT = (key: string) => string;

/** One-of-many chips for an optional field: tapping the picked chip clears it. */
function SingleChips({
  title,
  pilotId,
  options,
  value,
  onChange,
  render,
}: {
  title: string;
  pilotId: string;
  options: string[];
  value: string | null;
  onChange: (next: string | null) => void;
  render: (option: string) => string;
}) {
  return (
    <section className="nb-card onb2-cond">
      <h2 className="onb2-panel-title">{title}</h2>
      <ChipGroup label={title} pilot={pilot(pilotId, 'choice')}>
        {options.map((o) => (
          <PillChip key={o} pressed={value === o} onPressedChange={() => onChange(value === o ? null : o)} {...pilotOption(o)}>
            <bdi>{render(o)}</bdi>
          </PillChip>
        ))}
      </ChipGroup>
    </section>
  );
}

/**
 * The optional pregnancy history of `/pregnancy/setup`, in the sign-up
 * screens' cards and chips (nbl_Onb_Conditions look): history flags,
 * pre-existing conditions, blood group and Rh.
 */
export function HistoryStep({
  value,
  onChange,
  copy,
}: {
  value: SetupHistory;
  onChange: (next: SetupHistory) => void;
  /** Admin copy (`pregnancy_setup/history`); undefined → bundle. */
  copy?: SetupCopy;
}) {
  const t = useTranslations('pregnancyV2.setup');
  const tp = useTranslations('pregnancy') as unknown as DynT;
  const enums = usePregnancyEnums();
  const set = (patch: Partial<SetupHistory>) => onChange({ ...value, ...patch });

  const conditions = noneFirst(enums.data?.preExistingConditions ?? CONDITIONS);
  const bloodTypes = enums.data?.bloodTypes ?? BLOOD_TYPES;
  const rhFactors = enums.data?.rhFactors ?? RH_FACTORS;

  return (
    <>
      <section className="nb-card onb2-cond" aria-label={copy?.history.title ?? t('historyTitle')}>
        <div className="nb-chips">
          <PillChip
            mode="multi"
            pressed={value.miscarriage}
            onPressedChange={(on) => set({ miscarriage: on })}
            {...pilot('preg.setup-miscarriage', 'toggle')}
          >
            {t('miscarriage')}
          </PillChip>
          <PillChip
            mode="multi"
            pressed={value.highRisk}
            onPressedChange={(on) => set({ highRisk: on })}
            {...pilot('preg.setup-high-risk', 'toggle')}
          >
            {t('highRisk')}
          </PillChip>
        </div>
      </section>

      <section className="nb-card onb2-cond">
        <h2 className="onb2-panel-title">{t('conditions')}</h2>
        <ChipGroup label={t('conditions')} pilot={pilot('preg.setup-conditions', 'choice')}>
          {conditions.map((c) => (
            <PillChip
              key={c}
              mode="multi"
              pressed={value.conditions.includes(c)}
              onPressedChange={() => set({ conditions: toggleCondition(value.conditions, c) })}
              {...pilotOption(c)}
            >
              {c === 'none' ? t('conditionNone') : tp(`onboarding.conditions.${c}`)}
            </PillChip>
          ))}
        </ChipGroup>
      </section>

      <SingleChips
        title={t('bloodGroup')}
        pilotId="preg.setup-blood"
        options={bloodTypes}
        value={value.bloodType}
        onChange={(bloodType) => set({ bloodType })}
        render={(b) => b}
      />
      <SingleChips
        title={t('rh')}
        pilotId="preg.setup-rh"
        options={rhFactors}
        value={value.rhFactor}
        onChange={(rhFactor) => set({ rhFactor })}
        render={(r) => (r === 'negative' ? t('rhNegative') : t('rhPositive'))}
      />

      {/* Truthful copy: history is stored on the server (README open point 1). */}
      <div className="nb-card onb2-hint">
        <Icon name="lock" size={16} className="onb2-hint-icon" />
        <span>{copy?.history.disclaimer ?? t('historyDisclaimer')}</span>
      </div>
    </>
  );
}
