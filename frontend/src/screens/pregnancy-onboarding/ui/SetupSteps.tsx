'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo } from 'react';

import {
  DATING_SOURCES,
  usePregnancyEnums,
  type DatingPreview,
  type DatingSource,
} from '@/entities/pregnancy';
import { Chip, NumberField, PgCard, Segmented, Toggle } from '@/features/track-pregnancy';
import type { Locale } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { CalendarPicker } from '@/shared/ui';

import { toggleCondition, type SetupDating, type SetupHistory } from '../model/setup';

const BLOOD_TYPES = ['A', 'B', 'AB', 'O'];
const RH_FACTORS = ['positive', 'negative'];
const CONDITIONS = ['chronic_hypertension', 'diabetes', 'hypothyroidism', 'hyperthyroidism', 'none'];

type DynT = (key: string) => string;

function useDayOptions() {
  const loc = useLocale() as Locale;
  return useMemo(
    () => Array.from({ length: 7 }, (_, i) => ({ value: String(i), label: formatNumber(i, loc) })),
    [loc],
  );
}

// ── Step 1: dating source + its inputs ────────────────────────
export function DatingStep({
  value,
  onChange,
}: {
  value: SetupDating;
  onChange: (next: SetupDating) => void;
}) {
  const t = useTranslations('pregnancyV2.setup');
  const dayOptions = useDayOptions();
  const set = (patch: Partial<SetupDating>) => onChange({ ...value, ...patch });

  return (
    <div className="pon-stack">
      <div>
        <span className="pon-sublabel is-block">{t('sourceLabel')}</span>
        <div className="seg" role="radiogroup" aria-label={t('sourceLabel')}>
          {DATING_SOURCES.map((src: DatingSource) => (
            <button
              key={src}
              type="button"
              role="radio"
              aria-checked={value.source === src}
              className={value.source === src ? 'on' : undefined}
              onClick={() => set({ source: src })}
            >
              {t(`sources.${src}`)}
            </button>
          ))}
        </div>
        <p className="sub onb-note is-sub">{t(`sourceHints.${value.source}`)}</p>
      </div>

      {value.source === 'lmp' && (
        <PgCard title={t('lmpPick')} icon="calendar">
          <CalendarPicker value={value.lmp} onSelect={(lmp) => set({ lmp })} />
        </PgCard>
      )}

      {value.source === 'ultrasound' && (
        <PgCard title={t('ultrasoundDate')} icon="calendar">
          <CalendarPicker value={value.ultrasoundDate} onSelect={(d) => set({ ultrasoundDate: d })} />
          <div className="onb-mt12">
            <span className="pon-sublabel is-block">{t('ultrasoundAge')}</span>
            <NumberField
              label={t('week')}
              value={value.ultrasoundWeeks ?? undefined}
              onChange={(w) => set({ ultrasoundWeeks: w ?? null })}
              min={1}
              max={42}
            />
            <div className="onb-mt10">
              <span className="pon-sublabel is-block">{t('day')}</span>
              <Segmented
                options={dayOptions}
                value={String(value.ultrasoundDays)}
                onChange={(v) => set({ ultrasoundDays: v == null ? 0 : Number(v) })}
              />
            </div>
          </div>
        </PgCard>
      )}

      {value.source === 'manual' && (
        <PgCard title={t('manualAge')} icon="calendar">
          <NumberField
            label={t('week')}
            value={value.manualWeeks ?? undefined}
            onChange={(w) => set({ manualWeeks: w ?? null })}
            min={1}
            max={42}
          />
          <div className="onb-mt10">
            <span className="pon-sublabel is-block">{t('day')}</span>
            <Segmented
              options={dayOptions}
              value={String(value.manualDays)}
              onChange={(v) => set({ manualDays: v == null ? 0 : Number(v) })}
            />
          </div>
        </PgCard>
      )}
    </div>
  );
}

// ── Step 2: optional history ──────────────────────────────────
export function HistoryStep({
  value,
  onChange,
}: {
  value: SetupHistory;
  onChange: (next: SetupHistory) => void;
}) {
  const t = useTranslations('pregnancyV2.setup');
  const tp = useTranslations('pregnancy') as unknown as DynT;
  const enums = usePregnancyEnums();
  const set = (patch: Partial<SetupHistory>) => onChange({ ...value, ...patch });

  const conditions = enums.data?.preExistingConditions ?? CONDITIONS;
  const bloodTypes = enums.data?.bloodTypes ?? BLOOD_TYPES;
  const rhFactors = enums.data?.rhFactors ?? RH_FACTORS;

  return (
    <div className="pon-stack">
      <PgCard icon="shield">
        <div className="pon-toggle-row">
          <span className="pon-toggle-lbl">{t('miscarriage')}</span>
          <Toggle on={value.miscarriage} onClick={() => set({ miscarriage: !value.miscarriage })} />
        </div>
        <div className="pon-toggle-row">
          <span className="pon-toggle-lbl">{t('highRisk')}</span>
          <Toggle on={value.highRisk} onClick={() => set({ highRisk: !value.highRisk })} />
        </div>
      </PgCard>

      <PgCard title={t('conditions')} icon="stetho">
        <div className="pon-chips">
          {conditions.map((c) => (
            <Chip
              key={c}
              on={value.conditions.includes(c)}
              label={c === 'none' ? t('conditionNone') : tp(`onboarding.conditions.${c}`)}
              onClick={() => set({ conditions: toggleCondition(value.conditions, c) })}
            />
          ))}
        </div>
      </PgCard>

      <PgCard title={t('bloodGroup')} icon="drop">
        <div className="pon-chips">
          {bloodTypes.map((b) => (
            <Chip
              key={b}
              on={value.bloodType === b}
              label={b}
              onClick={() => set({ bloodType: value.bloodType === b ? null : b })}
            />
          ))}
        </div>
        <div className="onb-mt12">
          <span className="pon-sublabel is-block">{t('rh')}</span>
          <div className="pon-chips is-row">
            {rhFactors.map((r) => (
              <Chip
                key={r}
                on={value.rhFactor === r}
                label={r === 'negative' ? t('rhNegative') : t('rhPositive')}
                onClick={() => set({ rhFactor: value.rhFactor === r ? null : r })}
              />
            ))}
          </div>
        </div>
      </PgCard>

      {/* Truthful copy: history is stored on the server (README open point 1). */}
      <p className="sub onb-note is-sub">{t('historyDisclaimer')}</p>
    </div>
  );
}

// ── Step 3: result from dating-preview ────────────────────────
export function ResultStep({
  preview,
  loading,
  failed,
}: {
  preview: DatingPreview | undefined;
  loading: boolean;
  failed: boolean;
}) {
  const t = useTranslations('pregnancyV2');
  const loc = useLocale() as Locale;
  const date = (iso: string) => formatLongDate(fromApiDate(iso), loc);

  if (failed) return <p className="onb-error">{t('setup.previewError')}</p>;
  if (loading || !preview) return <p className="sub onb-note">{t('setup.calculating')}</p>;

  const level = preview.confidence.level;
  return (
    <div className="pon-stack">
      <PgCard icon="heart">
        <p className="sub">{t('setup.resultLead')}</p>
        <div className="titr">{t('common.age', { weeks: preview.age.weeks, days: preview.age.days })}</div>
        <p className="sub">{t('setup.resultTail')}</p>
      </PgCard>

      <PgCard title={t('common.dueDate')} icon="calendar">
        <div className="card-titr">{preview.dueDateLabel ?? date(preview.dueDate)}</div>
        {preview.range && (
          <p className="sub onb-mt8">
            {t('setup.usualRange', { from: date(preview.range.from), to: date(preview.range.to) })}
          </p>
        )}
        {level && (
          <p className="sub onb-mt8">
            {preview.uncertaintyDays != null
              ? t('common.confidence.withRange', {
                  level: t(`common.confidence.levels.${level}`),
                  days: preview.uncertaintyDays,
                })
              : t('common.confidence.label', { level: t(`common.confidence.levels.${level}`) })}
          </p>
        )}
        {preview.basis && <p className="sub onb-note is-sub">{preview.basis}</p>}
      </PgCard>

      <p className="sub onb-note is-sub">{t('common.estimateNote')}</p>
    </div>
  );
}
