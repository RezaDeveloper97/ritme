'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useMemo } from 'react';

import { useLogDays, type LogCategory, type LogDayValues, type LogParam, type LogParamValue } from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { addDays, formatDecimal, fromApiDate, toApiDate } from '@/shared/lib/date';
import { ChipGroup, Icon, PillChip, type IconName, type Tone } from '@/shared/ui';

import { asNumber, asString, normalizeNumber, pickableOptions } from '../../model/draft';
import { ParamField } from '../ParamField';
import { PanelCard } from './PanelCard';

interface MeasurePanelProps {
  category: LogCategory;
  values: LogDayValues;
  setParam: (category: string, param: string, value: LogParamValue | null) => void;
  mode: string | null;
  locale: Locale;
  date: string;
}

/** How far back the «ثبت قبلی» comparison looks. */
const LOOKBACK_DAYS = 30;

const STEPPED: Record<string, { icon: IconName; tone: Tone; step: number; fallback: number }> = {
  weight: { icon: 'scaleSquare', tone: 'data', step: 0.1, fallback: 60 },
  bbt: { icon: 'thermo', tone: 'brand', step: 0.01, fallback: 36.5 },
};

const TESTS: Record<string, Tone> = { lh_test: 'warm', pregnancy_test: 'bloom' };

/**
 * Log_Measure: weight and basal temperature as big −/+ steppers (the last reading within 30 days is the
 * starting point and the comparison), the LH / pregnancy tests as chips, and any other measure the mode
 * has (blood pressure, heart rate, sugar) as plain number fields.
 */
export function MeasurePanel({ category, values, setParam, mode, locale, date }: MeasurePanelProps) {
  const t = useTranslations('logSheet');
  const cat = category.code;
  const v = values[cat] ?? {};
  const from = toApiDate(addDays(fromApiDate(date), -LOOKBACK_DAYS));
  const to = toApiDate(addDays(fromApiDate(date), -1));
  const history = useLogDays(from, to);
  const last = useMemo(() => {
    const out: Record<string, number> = {};
    const days = [...(history.data?.days ?? [])].sort((a, b) => a.date.localeCompare(b.date));
    for (const d of days) {
      for (const [code, value] of Object.entries(d.categories[cat] ?? {})) {
        if (typeof value === 'number') out[code] = value;
      }
    }
    return out;
  }, [history.data, cat]);

  const stepped = category.params.filter((p) => STEPPED[p.code] && (p.type === 'number' || p.type === 'integer'));
  const tests = category.params.filter((p) => TESTS[p.code] && p.type === 'single');
  const rest = category.params.filter((p) => !stepped.includes(p) && !tests.includes(p));

  return (
    <>
      {stepped.map((p) => (
        <BigStepper
          key={p.code}
          param={p}
          value={asNumber(v[p.code])}
          previous={last[p.code] ?? null}
          onChange={(n) => setParam(cat, p.code, n)}
          locale={locale}
        />
      ))}

      {tests.length ? (
        <PanelCard title={t('measure.tests')} icon="flask" tone="warm">
          {tests.map((p) => {
            const current = asString(v[p.code]);
            const tone = TESTS[p.code];
            return (
              <div key={p.code} className="lday-param">
                <div className="lday-plabel">{p.label}</div>
                <ChipGroup label={p.label} layout="fill">
                  <PillChip shape="square" mode="multi" tone={tone} pressed={current === null} onPressedChange={() => setParam(cat, p.code, null)}>
                    {t('measure.testNone')}
                  </PillChip>
                  {pickableOptions(p, mode).map((o) => (
                    <PillChip
                      key={o.value}
                      shape="square"
                      mode="multi"
                      tone={tone}
                      pressed={current === o.value}
                      onPressedChange={() => setParam(cat, p.code, current === o.value ? null : o.value)}
                    >
                      {o.label}
                    </PillChip>
                  ))}
                </ChipGroup>
              </div>
            );
          })}
        </PanelCard>
      ) : null}

      {rest.length ? (
        <PanelCard title={t('measure.other')} icon="heartLine" tone="bloom">
          {rest.map((p) => (
            <ParamField
              key={p.code}
              param={p}
              value={v[p.code]}
              onChange={(next) => setParam(cat, p.code, next)}
              mode={mode}
              tone="bloom"
              locale={locale}
            />
          ))}
        </PanelCard>
      ) : null}
    </>
  );
}

function BigStepper({
  param,
  value,
  previous,
  onChange,
  locale,
}: {
  param: LogParam;
  value: number | null;
  previous: number | null;
  onChange: (value: number | null) => void;
  locale: Locale;
}) {
  const t = useTranslations('logSheet');
  const look = STEPPED[param.code];
  const scale = param.type === 'integer' ? 0 : look.step < 0.1 ? 2 : 1;
  const base = value ?? previous ?? look.fallback;
  const shown = formatDecimal(base.toFixed(scale), locale);
  const unit = param.unit?.label ?? '';
  const nudge = (dir: 1 | -1) => onChange(normalizeNumber(param, (value ?? base) + (value === null ? 0 : dir * look.step)));

  let caption = '';
  if (param.code === 'weight') {
    if (value !== null && previous !== null) {
      const diff = Math.round((value - previous) * 10) / 10;
      const n = formatDecimal(Math.abs(diff).toFixed(1), locale);
      caption = diff === 0 ? t('measure.diffSame') : diff < 0 ? t('measure.diffLess', { n }) : t('measure.diffMore', { n });
    }
  } else if (param.code === 'bbt') {
    caption = t('measure.bbtHint');
  }

  return (
    <PanelCard title={param.label} icon={look.icon} tone={look.tone}>
      <div className="lday-big" role="group" aria-label={param.label}>
        <button type="button" className="lday-big-btn" aria-label={t('measure.decrease', { name: param.label })} onClick={() => nudge(-1)}>
          <Icon name="minus" size={20} strokeWidth={2.4} />
        </button>
        <output className={clsx('lday-big-num', value === null && 'is-empty')} aria-live="polite">
          {shown}
        </output>
        <button type="button" className="lday-big-btn" aria-label={t('measure.increase', { name: param.label })} onClick={() => nudge(1)}>
          <Icon name="plus" size={20} strokeWidth={2.4} />
        </button>
      </div>
      <p className="lday-big-cap">
        {value === null ? t('measure.empty') : [unit, caption].filter(Boolean).join(' · ')}
        {value !== null ? (
          <button type="button" className="lday-big-clear" onClick={() => onChange(null)}>
            {t('measure.clear')}
          </button>
        ) : null}
      </p>
    </PanelCard>
  );
}
