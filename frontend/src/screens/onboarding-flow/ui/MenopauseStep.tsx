'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { type Locale, useDirection } from '@/shared/i18n';
import {
  formatMonthLabel,
  fromApiDate,
  partsToDate,
  shiftMonth,
  toApiDate,
  todayParts,
  toParts,
} from '@/shared/lib/date';
import { Icon, PillChip, RadioCardGroup, type RadioCardOption } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { useSaveOnboardingStep } from '../api/onboarding';
import { MENOPAUSE_STAGES, type MenopauseStage } from '../model/state';
import { StepScreen, type StepContext } from './StepScreen';

const META: Record<MenopauseStage, Pick<RadioCardOption<MenopauseStage>, 'icon' | 'iconTone'>> = {
  peri: { icon: 'clock', iconTone: 'period' },
  meno: { icon: 'moon', iconTone: 'brand' },
  post: { icon: 'shield', iconTone: 'data' },
  unsure: { icon: 'info', iconTone: 'neutral' },
};

type Month = { year: number; month: number };

function YesNo({ label, value, onChange }: { label: string; value: boolean | null; onChange: (v: boolean | null) => void }) {
  const t = useTranslations('onboarding.flow.menopause');
  return (
    <div className="onb2-yn" role="group" aria-label={label}>
      <span className="onb2-yn-q">{label}</span>
      <span className="onb2-yn-pills">
        <PillChip mode="multi" pressed={value === true} onPressedChange={(on) => onChange(on ? true : null)}>
          {t('yes')}
        </PillChip>
        <PillChip mode="multi" pressed={value === false} onPressedChange={(on) => onChange(on ? false : null)}>
          {t('no')}
        </PillChip>
      </span>
    </div>
  );
}

function MenopauseForm({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow');
  const te = useTranslations('onboarding.enums.menopauseStage');
  const locale = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const save = useSaveOnboardingStep();
  const saved = ctx.state.menopause;
  const [stage, setStage] = useState<MenopauseStage | null>(saved.stage);
  const [lastPeriod, setLastPeriod] = useState<Month | null>(() => {
    if (!saved.last_period) return null;
    const p = toParts(fromApiDate(saved.last_period), locale);
    return { year: p.year, month: p.month };
  });
  const [surgical, setSurgical] = useState<boolean | null>(saved.surgical);
  const [hrt, setHrt] = useState<boolean | null>(saved.hrt);

  const now = todayParts(locale);
  const nowKey = now.year * 12 + now.month;
  const shown = lastPeriod ?? { year: now.year, month: now.month };
  const step = (delta: number) => {
    const next = shiftMonth(shown.year, shown.month, lastPeriod ? delta : Math.min(0, delta));
    if (next.year * 12 + next.month <= nowKey) setLastPeriod(next);
  };

  const options = MENOPAUSE_STAGES.map((value) => ({
    value,
    title: te(value),
    description: t(`menopause.desc.${value}`),
    ...META[value],
  }));

  const submit = () => {
    if (!stage || save.isPending) return;
    save.mutate(
      {
        step: 'menopause',
        body: {
          stage,
          // Approximate: any day of the chosen month (the API's contract).
          last_period: lastPeriod ? toApiDate(partsToDate({ ...lastPeriod, day: 1 }, locale)) : null,
          surgical,
          hrt,
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
      title={t('menopause.title')}
      subtitle={t('menopause.subtitle')}
      primary={{ label: t('continue'), onClick: submit, disabled: !stage, loading: save.isPending }}
    >
      <RadioCardGroup options={options} value={stage} onChange={setStage} label={t('menopause.title')} />

      <section className="nb-card onb2-panel onb2-meno-q">
        <div className="onb2-month">
          <span className="onb2-month-label">{t('menopause.lastPeriod')}</span>
          <span className="onb2-month-ctl">
            <button type="button" className="onb2-cal-btn is-sm" aria-label={t('calendar.prev')} onClick={() => step(-1)}>
              <Icon name={rtl ? 'chevronRight' : 'chevronLeft'} size={18} />
            </button>
            <b className="onb2-month-value" aria-live="polite">
              {lastPeriod
                ? t('menopause.about', { month: formatMonthLabel(lastPeriod.year, lastPeriod.month, locale) })
                : t('menopause.notSet')}
            </b>
            <button
              type="button"
              className="onb2-cal-btn is-sm"
              aria-label={t('calendar.next')}
              disabled={!lastPeriod || shown.year * 12 + shown.month >= nowKey}
              onClick={() => step(1)}
            >
              <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} />
            </button>
          </span>
        </div>
        <YesNo label={t('menopause.surgical')} value={surgical} onChange={setSurgical} />
        <YesNo label={t('menopause.hrt')} value={hrt} onChange={setHrt} />
      </section>
      {save.isError ? <p className="onb2-error" role="alert">{t('saveError')}</p> : null}
    </OnbFrame>
  );
}

/** nbl_Onb_Meno — stage, approximate last period, surgical cause, HRT. */
export function MenopauseStep() {
  return <StepScreen step="menopause">{(ctx) => <MenopauseForm ctx={ctx} />}</StepScreen>;
}
