'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';
import { clsx } from 'clsx';

import { useActivatePregnancy, useCompleteOnboarding } from '@/entities/pregnancy';
import type { Locale } from '@/shared/i18n';
import { addDays, formatLongDate, formatNumber, today } from '@/shared/lib/date';
import { Icon, type IconName } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import {
  MAX_GA_DAYS,
  PREGNANCY_BASES,
  pregnancyOnboardingBody,
  pregnancySummary,
  TERM_DAYS,
  type PregnancyAnswers,
  type PregnancyBasis,
} from '../model/pregnancy';
import { DateRow, NumField } from './fields';
import { MonthCalendar } from './MonthCalendar';
import { StepScreen, type StepContext } from './StepScreen';

const TILE: Record<PregnancyBasis, { icon: IconName; tone: string }> = {
  lmp: { icon: 'drop', tone: 'period' },
  ultrasound: { icon: 'user', tone: 'bloom' },
  due: { icon: 'cake', tone: 'warm' },
  week: { icon: 'calendar', tone: 'brand' },
};

function PregnancyForm({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow');
  const locale = useLocale() as Locale;
  const activate = useActivatePregnancy();
  const onboard = useCompleteOnboarding();
  const now = today();
  const [a, setA] = useState<PregnancyAnswers>({
    basis: 'ultrasound',
    lmp: null,
    scanDate: null,
    scanWeeks: null,
    scanDays: 0,
    dueDate: null,
    weeks: null,
    days: 0,
  });
  const patch = (p: Partial<PregnancyAnswers>) => setA((prev) => ({ ...prev, ...p }));
  const summary = pregnancySummary(a, now);
  const body = pregnancyOnboardingBody(a, now);
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

  const pastRange = { min: addDays(now, -MAX_GA_DAYS), max: now };
  const dateRow = (key: 'lmp' | 'scanDate' | 'dueDate', label: string, range: { min: Date; max: Date }) => (
    <DateRow label={label} value={a[key]} placeholder={t('pregnancy.pickDate')}>
      {(close) => (
        <MonthCalendar
          label={label}
          value={a[key]}
          min={range.min}
          max={range.max}
          onChange={(d) => {
            patch({ [key]: d });
            close();
          }}
        />
      )}
    </DateRow>
  );

  return (
    <OnbFrame
      progress={ctx.progress}
      onBack={ctx.back}
      onSkip={() => ctx.next()}
      title={t('pregnancy.title')}
      subtitle={t('pregnancy.subtitle')}
      primary={{ label: t('continue'), onClick: () => void submit(), disabled: !body, loading: pending }}
    >
      <div className="onb2-basis" role="radiogroup" aria-label={t('pregnancy.basisLabel')}>
        {PREGNANCY_BASES.map((b) => (
          <button
            key={b}
            type="button"
            role="radio"
            aria-checked={a.basis === b}
            className={clsx('onb2-basis-tile', `nb-tone-${TILE[b].tone}`)}
            onClick={() => patch({ basis: b })}
          >
            <Icon name={TILE[b].icon} size={22} strokeWidth={1.8} className="onb2-basis-icon" />
            <span>{t(`pregnancy.basis.${b}`)}</span>
          </button>
        ))}
      </div>

      <section className="nb-card onb2-panel" aria-label={t(`pregnancy.panel.${a.basis}`)}>
        <h2 className="onb2-panel-title">{t(`pregnancy.panel.${a.basis}`)}</h2>
        {a.basis === 'lmp' ? dateRow('lmp', t('pregnancy.lmpDate'), pastRange) : null}
        {a.basis === 'ultrasound' ? (
          <>
            {dateRow('scanDate', t('pregnancy.scanDate'), pastRange)}
            <div className="onb2-num-pair">
              <NumField label={t('pregnancy.weeks')} value={a.scanWeeks} onChange={(v) => patch({ scanWeeks: v })} min={1} max={42} />
              <NumField label={t('pregnancy.days')} value={a.scanDays} onChange={(v) => patch({ scanDays: v ?? 0 })} min={0} max={6} />
            </div>
            <p className="onb2-panel-note">{t('pregnancy.scanNote')}</p>
          </>
        ) : null}
        {a.basis === 'due' ? dateRow('dueDate', t('pregnancy.dueDate'), { min: now, max: addDays(now, TERM_DAYS) }) : null}
        {a.basis === 'week' ? (
          <div className="onb2-num-pair">
            <NumField label={t('pregnancy.weeks')} value={a.weeks} onChange={(v) => patch({ weeks: v })} min={1} max={42} />
            <NumField label={t('pregnancy.days')} value={a.days} onChange={(v) => patch({ days: v ?? 0 })} min={0} max={6} />
          </div>
        ) : null}
      </section>

      {summary ? (
        <div className="onb2-preg-sum" aria-live="polite">
          <div>
            <div className="onb2-preg-sum-label">{t('pregnancy.today')}</div>
            <div className="onb2-preg-sum-ga">
              {t('pregnancy.ga', { weeks: formatNumber(summary.weeks, locale), days: formatNumber(summary.days, locale) })}
            </div>
          </div>
          <div className="onb2-preg-sum-due">
            <div className="onb2-preg-sum-label">{t('pregnancy.due')}</div>
            <div className="onb2-preg-sum-date">{formatLongDate(summary.due, locale)}</div>
          </div>
        </div>
      ) : null}
      {failed ? <p className="onb2-error" role="alert">{t('saveError')}</p> : null}
    </OnbFrame>
  );
}

/** nbl_Onb_Preg — dates the pregnancy (POST /pregnancy/activate + /pregnancy/onboarding). */
export function PregnancyStep() {
  return <StepScreen step="pregnancy">{(ctx) => <PregnancyForm ctx={ctx} />}</StepScreen>;
}
