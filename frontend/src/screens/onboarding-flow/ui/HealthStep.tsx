'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import type { Locale } from '@/shared/i18n';
import {
  birthYearRange,
  diffInDays,
  formatNumber,
  fromApiDate,
  partsToDate,
  toApiDate,
  today,
  toParts,
  todayParts,
} from '@/shared/lib/date';
import { NumberStepper } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';
import { DateWheels } from '@/features/edit-profile';

import { useSaveOnboardingStep } from '../api/onboarding';
import { DateRow } from './fields';
import { StepScreen, type StepContext } from './StepScreen';

const HEIGHT = { min: 120, max: 220, fallback: 160 } as const;
const WEIGHT = { min: 30, max: 200, fallback: 60 } as const;

/** Whole years between `birthday` and today. */
const ageOf = (birthday: Date) => Math.floor(diffInDays(today(), birthday) / 365.2425);

function HealthForm({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow');
  const locale = useLocale() as Locale;
  const save = useSaveOnboardingStep();
  const saved = ctx.state.health;
  const [birthday, setBirthday] = useState<Date | null>(saved.birthday ? fromApiDate(saved.birthday) : null);
  // Every field may stay empty: a stepper only counts once it was touched or saved.
  const [height, setHeight] = useState<number | null>(saved.height);
  const [weight, setWeight] = useState<number | null>(saved.weight != null ? Math.round(saved.weight) : null);
  const years = birthYearRange(locale);

  const submit = () => {
    if (save.isPending) return;
    save.mutate(
      {
        step: 'health',
        body: {
          ...(birthday ? { birthday: toApiDate(birthday) } : {}),
          ...(height != null ? { height } : {}),
          ...(weight != null ? { weight } : {}),
        },
      },
      { onSuccess: () => ctx.next() },
    );
  };

  return (
    <OnbFrame
      progress={ctx.progress}
      onBack={ctx.back}
      onSkip={() => ctx.next()}
      title={t('health.title')}
      subtitle={t('health.subtitle')}
      primary={{ label: t('continue'), onClick: submit, loading: save.isPending }}
    >
      <DateRow
        icon="cake"
        label={t('health.birthday')}
        value={birthday}
        placeholder={t('health.pickBirthday')}
        trailing={birthday ? t('health.age', { age: formatNumber(ageOf(birthday), locale) }) : null}
      >
        {() => (
          <DateWheels
            idPrefix="onb2-bd"
            value={birthday ? toParts(birthday, locale) : { ...todayParts(locale), year: years.max - 24 }}
            onChange={(p) => setBirthday(partsToDate(p, locale))}
            minYear={years.min}
            maxYear={years.max}
          />
        )}
      </DateRow>
      <NumberStepper
        boxed
        className={height == null ? 'onb2-stepper is-unset' : 'onb2-stepper'}
        label={t('health.height')}
        unit={t('health.cm')}
        value={height ?? HEIGHT.fallback}
        onChange={setHeight}
        min={HEIGHT.min}
        max={HEIGHT.max}
        decrementLabel={t('decrease')}
        incrementLabel={t('increase')}
        locale={locale}
      />
      <NumberStepper
        boxed
        className={weight == null ? 'onb2-stepper is-unset' : 'onb2-stepper'}
        label={t('health.weight')}
        unit={t('health.kg')}
        value={weight ?? WEIGHT.fallback}
        onChange={setWeight}
        min={WEIGHT.min}
        max={WEIGHT.max}
        decrementLabel={t('decrease')}
        incrementLabel={t('increase')}
        locale={locale}
      />
      <p className="onb2-note">{t('health.optional')}</p>
      {save.isError ? <p className="onb2-error" role="alert">{t('saveError')}</p> : null}
    </OnbFrame>
  );
}

/** nbl_Onb_Health — merges the old birthday, height and weight screens. */
export function HealthStep() {
  return <StepScreen step="health">{(ctx) => <HealthForm ctx={ctx} />}</StepScreen>;
}
