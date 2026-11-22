'use client';

import { useLocale, useTranslations } from 'next-intl';
import { clsx } from 'clsx';

import { DateRow, MonthCalendar, NumField } from '@/features/auth';
import type { Locale } from '@/shared/i18n';
import { addDays, formatLongDate, formatNumber, today } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { Icon, type IconName } from '@/shared/ui';

import {
  MAX_GA_DAYS,
  PREGNANCY_BASES,
  pregnancySummary,
  TERM_DAYS,
  type PregnancyAnswers,
  type PregnancyBasis,
} from '../model/dating';
import '@/shared/ui/styles/onb.css';

const TILE: Record<PregnancyBasis, { icon: IconName; tone: string }> = {
  lmp: { icon: 'drop', tone: 'period' },
  ultrasound: { icon: 'user', tone: 'bloom' },
  due: { icon: 'cake', tone: 'warm' },
  week: { icon: 'calendar', tone: 'brand' },
};

type DateKey = 'lmp' | 'scanDate' | 'dueDate';
/** Stable autopilot keys of the three date rows. */
const DATE_PILOT: Record<DateKey, string> = { lmp: 'lmp', scanDate: 'scan', dueDate: 'due' };

interface PregnancyDatingFormProps {
  value: PregnancyAnswers;
  onChange: (next: PregnancyAnswers) => void;
  /**
   * Autopilot area of the controls (`<area>-basis`, `<area>-date:<key>`, …).
   * Omitted in the sign-up flow, whose screens are deliberately unnamed.
   */
  pilotArea?: string;
}

/**
 * The pregnancy dating form of nbl_Onb_Preg — four basis tiles (last period,
 * ultrasound, due date, current week), the basis' own inputs, and the live
 * «امروز / تاریخ زایمان» summary. Controlled; shared by the sign-up step and
 * the in-app `/pregnancy/setup`, so both look the same.
 */
export function PregnancyDatingForm({ value: a, onChange, pilotArea }: PregnancyDatingFormProps) {
  const t = useTranslations('onboarding.flow');
  const locale = useLocale() as Locale;
  const now = today();
  const patch = (p: Partial<PregnancyAnswers>) => onChange({ ...a, ...p });
  const summary = pregnancySummary(a, now);
  const named = (id: string, kind?: Parameters<typeof pilot>[1]) => (pilotArea ? pilot(`${pilotArea}-${id}`, kind) : undefined);

  const pastRange = { min: addDays(now, -MAX_GA_DAYS), max: now };
  const dateRow = (key: DateKey, label: string, range: { min: Date; max: Date }) => (
    <DateRow label={label} value={a[key]} placeholder={t('pregnancy.pickDate')} pilot={named(`date-open:${DATE_PILOT[key]}`)}>
      {(close) => (
        <MonthCalendar
          label={label}
          value={a[key]}
          min={range.min}
          max={range.max}
          pilot={named(`date:${DATE_PILOT[key]}`, 'calendar')}
          onChange={(d) => {
            patch({ [key]: d });
            close();
          }}
        />
      )}
    </DateRow>
  );
  const ageFields = (weeksKey: 'scanWeeks' | 'weeks', daysKey: 'scanDays' | 'days', basis: PregnancyBasis) => (
    <div className="onb2-num-pair">
      <NumField
        label={t('pregnancy.weeks')}
        value={a[weeksKey]}
        onChange={(v) => patch({ [weeksKey]: v })}
        min={1}
        max={42}
        pilot={named(`weeks:${basis}`, 'number')}
      />
      <NumField
        label={t('pregnancy.days')}
        value={a[daysKey]}
        onChange={(v) => patch({ [daysKey]: v ?? 0 })}
        min={0}
        max={6}
        pilot={named(`days:${basis}`, 'number')}
      />
    </div>
  );

  return (
    <>
      <div className="onb2-basis" role="radiogroup" aria-label={t('pregnancy.basisLabel')} {...named('basis', 'choice')}>
        {PREGNANCY_BASES.map((b) => (
          <button
            key={b}
            type="button"
            role="radio"
            aria-checked={a.basis === b}
            className={clsx('onb2-basis-tile', `nb-tone-${TILE[b].tone}`)}
            onClick={() => patch({ basis: b })}
            {...(pilotArea ? pilotOption(b) : undefined)}
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
            {ageFields('scanWeeks', 'scanDays', 'ultrasound')}
            <p className="onb2-panel-note">{t('pregnancy.scanNote')}</p>
          </>
        ) : null}
        {a.basis === 'due' ? dateRow('dueDate', t('pregnancy.dueDate'), { min: now, max: addDays(now, TERM_DAYS) }) : null}
        {a.basis === 'week' ? ageFields('weeks', 'days', 'week') : null}
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
    </>
  );
}
