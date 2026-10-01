'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import type { Locale } from '@/shared/i18n';
import { addDays, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { Checkbox, NumberStepper } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { useSaveOnboardingStep } from '../api/onboarding';
import { MonthCalendar } from './MonthCalendar';
import { StepScreen, type StepContext } from './StepScreen';

/** Same ranges as the cycle-settings screen (B-N1-09). */
const PERIOD = { min: 2, max: 10, fallback: 5 } as const;
const CYCLE = { min: 20, max: 45, fallback: 28 } as const;
const clamp = (n: number | null, r: { min: number; max: number; fallback: number }) =>
  n == null ? r.fallback : Math.min(r.max, Math.max(r.min, n));

function CycleForm({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow');
  const locale = useLocale() as Locale;
  const save = useSaveOnboardingStep();
  const saved = ctx.state.cycle;
  const [lastPeriod, setLastPeriod] = useState<Date | null>(
    saved.last_period_start ? fromApiDate(saved.last_period_start) : null,
  );
  const [unsure, setUnsure] = useState(false);
  const [periodLen, setPeriodLen] = useState(clamp(saved.period_duration, PERIOD));
  const [cycleLen, setCycleLen] = useState(clamp(saved.cycle_duration, CYCLE));
  const now = today();

  const canContinue = unsure || lastPeriod !== null;

  const submit = () => {
    if (!canContinue || save.isPending) return;
    save.mutate(
      {
        step: 'cycle',
        body: {
          // «دقیق یادم نیست» sends no date: the engine starts from the lengths.
          ...(unsure || !lastPeriod ? {} : { last_period_start: toApiDate(lastPeriod) }),
          period_duration: periodLen,
          cycle_duration: cycleLen,
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
      title={t('cycle.title')}
      primary={{ label: t('continue'), onClick: submit, disabled: !canContinue, loading: save.isPending }}
    >
      <MonthCalendar
        label={t('cycle.title')}
        value={unsure ? null : lastPeriod}
        onChange={setLastPeriod}
        min={addDays(now, -365)}
        max={now}
        span={periodLen}
        disabled={unsure}
      />
      <Checkbox className="onb2-unsure" checked={unsure} onCheckedChange={setUnsure} label={t('cycle.unsure')} />
      <NumberStepper
        boxed
        className="onb2-stepper"
        label={t('cycle.periodLabel')}
        description={t('cycle.periodDesc')}
        unit={t('unitDay')}
        value={periodLen}
        onChange={setPeriodLen}
        min={PERIOD.min}
        max={PERIOD.max}
        decrementLabel={t('decrease')}
        incrementLabel={t('increase')}
        locale={locale}
      />
      <NumberStepper
        boxed
        className="onb2-stepper"
        label={t('cycle.cycleLabel')}
        description={t('cycle.cycleDesc')}
        unit={t('unitDay')}
        value={cycleLen}
        onChange={setCycleLen}
        min={CYCLE.min}
        max={CYCLE.max}
        decrementLabel={t('decrease')}
        incrementLabel={t('increase')}
        locale={locale}
      />
      <p className="onb2-note">{t('cycle.note')}</p>
      {save.isError ? <p className="onb2-error" role="alert">{t('saveError')}</p> : null}
    </OnbFrame>
  );
}

/** nbl_Onb_Cycle — merges the old last-period, period-length and cycle-length screens. */
export function CycleStep() {
  return <StepScreen step="cycle">{(ctx) => <CycleForm ctx={ctx} />}</StepScreen>;
}
