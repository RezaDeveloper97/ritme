'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  DATING_SOURCES,
  usePregnancyEnums,
  type DatingPreview,
  type DatingSource,
  type SetupCopy,
} from '@/entities/pregnancy';
import { Chip, PgCard, Toggle } from '@/features/track-pregnancy';
import type { Locale } from '@/shared/i18n';
import { type DateParts, formatDayMonth, formatLongDate, fromApiDate, partsToDate } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, Icon, LocaleNumberField } from '@/shared/ui';
import { ResultBaby } from '@/shared/ui/illustrations';

import { noneFirst, toggleCondition, type SetupDating, type SetupHistory } from '../model/setup';

const BLOOD_TYPES = ['A', 'B', 'AB', 'O'];
const RH_FACTORS = ['positive', 'negative'];
const CONDITIONS = ['chronic_hypertension', 'diabetes', 'hypothyroidism', 'hyperthyroidism', 'none'];

type DynT = (key: string) => string;

/** A read-only date field that opens the calendar in a sheet (Setup artboard, ultrasound). */
function DateField({
  label,
  value,
  onSelect,
}: {
  label: string;
  value: DateParts | null;
  onSelect: (value: DateParts) => void;
}) {
  const t = useTranslations('pregnancyV2.setup');
  const loc = useLocale() as Locale;
  const [open, setOpen] = useState(false);
  return (
    <>
      <span className="pon-sublabel is-block">{label}</span>
      <button type="button" className="field pon-datefield" onClick={() => setOpen(true)} aria-haspopup="dialog">
        <span className={value ? undefined : 'pon-placeholder'}>
          {value ? formatLongDate(partsToDate(value, loc), loc) : t('pickDate')}
        </span>
        <Icon name="calendar" size={18} />
      </button>
      <AppSheet open={open} onClose={() => setOpen(false)} size="half" title={label}>
        <CalendarPicker
          value={value}
          onSelect={(d) => {
            onSelect(d);
            setOpen(false);
          }}
        />
      </AppSheet>
    </>
  );
}

/** Week + day as two number inputs on one row, digits in the locale's script (۸ in fa). */
function AgeFields({
  weeks,
  days,
  onChange,
}: {
  weeks: number | null;
  days: number;
  onChange: (weeks: number | null, days: number) => void;
}) {
  const t = useTranslations('pregnancyV2.setup');
  const loc = useLocale() as Locale;
  return (
    <div className="pon-pair">
      <LocaleNumberField
        label={t('week')}
        locale={loc}
        value={weeks ?? undefined}
        onChange={(w) => onChange(w ?? null, days)}
        max={42}
      />
      <LocaleNumberField
        label={t('day')}
        locale={loc}
        value={days}
        onChange={(d) => onChange(weeks, Math.min(6, Math.max(0, Math.trunc(d ?? 0))))}
        max={6}
      />
    </div>
  );
}

// ── Step 1: dating source + its inputs ────────────────────────
export function DatingStep({
  value,
  onChange,
  copy,
}: {
  value: SetupDating;
  onChange: (next: SetupDating) => void;
  /** Admin copy (`pregnancy_setup/source_*`); undefined → bundle. */
  copy?: SetupCopy;
}) {
  const t = useTranslations('pregnancyV2.setup');
  const loc = useLocale() as Locale;
  const set = (patch: Partial<SetupDating>) => onChange({ ...value, ...patch });

  return (
    <div className="pon-stack">
      <div>
        <div className="seg pon-seg" role="radiogroup" aria-label={t('sourceLabel')}>
          {DATING_SOURCES.map((src: DatingSource) => (
            <button
              key={src}
              type="button"
              role="radio"
              aria-checked={value.source === src}
              className={value.source === src ? 'on' : undefined}
              onClick={() => set({ source: src })}
            >
              {copy?.sources[src].label ?? t(`sources.${src}`)}
            </button>
          ))}
        </div>
        <p className="sub onb-note is-sub">{copy?.sources[value.source].hint ?? t(`sourceHints.${value.source}`)}</p>
      </div>

      {value.source === 'lmp' && (
        <div className="pon-cal">
          <div className="pon-cal-hd">
            <span className="pon-sublabel">{t('lmpPick')}</span>
            {value.lmp && (
              <span className="pon-datechip" aria-live="polite">
                {formatDayMonth(partsToDate(value.lmp, loc), loc)}
              </span>
            )}
          </div>
          <CalendarPicker value={value.lmp} onSelect={(lmp) => set({ lmp })} />
        </div>
      )}

      {value.source === 'ultrasound' && (
        <PgCard>
          <DateField label={t('ultrasoundDate')} value={value.ultrasoundDate} onSelect={(d) => set({ ultrasoundDate: d })} />
          <div className="onb-mt14">
            <span className="pon-sublabel is-block">{t('ultrasoundAge')}</span>
            <AgeFields
              weeks={value.ultrasoundWeeks}
              days={value.ultrasoundDays}
              onChange={(w, d) => set({ ultrasoundWeeks: w, ultrasoundDays: d })}
            />
          </div>
        </PgCard>
      )}

      {value.source === 'manual' && (
        <PgCard>
          <span className="pon-sublabel is-block">{t('manualAge')}</span>
          <AgeFields
            weeks={value.manualWeeks}
            days={value.manualDays}
            onChange={(w, d) => set({ manualWeeks: w, manualDays: d })}
          />
        </PgCard>
      )}
    </div>
  );
}

// ── Step 2: optional history ──────────────────────────────────
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
    <div className="pon-stack">
      <PgCard>
        <div className="pon-toggle-row">
          <span className="pon-toggle-lbl">{t('miscarriage')}</span>
          <Toggle on={value.miscarriage} onClick={() => set({ miscarriage: !value.miscarriage })} />
        </div>
        <div className="pon-toggle-row">
          <span className="pon-toggle-lbl">{t('highRisk')}</span>
          <Toggle on={value.highRisk} onClick={() => set({ highRisk: !value.highRisk })} />
        </div>
      </PgCard>

      <div>
        <span className="pon-sublabel is-block">{t('conditions')}</span>
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
      </div>

      <div className="pon-pair">
        <label className="fld-label">
          <span className="fld-label-t">{t('bloodGroup')}</span>
          <select
            className="field pon-select"
            value={value.bloodType ?? ''}
            onChange={(e) => set({ bloodType: e.target.value || null })}
          >
            <option value="">{t('choose')}</option>
            {bloodTypes.map((b) => (
              <option key={b} value={b}>
                {b}
              </option>
            ))}
          </select>
        </label>
        <label className="fld-label">
          <span className="fld-label-t">{t('rh')}</span>
          <select
            className="field pon-select"
            value={value.rhFactor ?? ''}
            onChange={(e) => set({ rhFactor: e.target.value || null })}
          >
            <option value="">{t('choose')}</option>
            {rhFactors.map((r) => (
              <option key={r} value={r}>
                {r === 'negative' ? t('rhNegative') : t('rhPositive')}
              </option>
            ))}
          </select>
        </label>
      </div>

      {/* Truthful copy: history is stored on the server (README open point 1). */}
      <p className="sub onb-note is-sub">{copy?.history.disclaimer ?? t('historyDisclaimer')}</p>
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

  if (failed) return <p className="onb-error">{t('setup.previewError')}</p>;
  if (loading || !preview) return <p className="sub onb-note">{t('setup.calculating')}</p>;

  const { copy } = preview;
  const level = preview.confidence.level;
  const levelText = level ? t(`common.confidence.levels.${level}`) : null;
  // Admin copy (`pregnancy_setup/result`) first, the bundle as the fallback.
  const confidence = levelText
    ? copy.confidence
      ? copy.confidence.replace('{confidence}', levelText)
      : t('common.confidence.label', { level: levelText })
    : null;
  const range =
    preview.rangeLabel ??
    (preview.range
      ? t('setup.usualRange', {
          from: formatDayMonth(fromApiDate(preview.range.from), loc),
          to: formatDayMonth(fromApiDate(preview.range.to), loc),
        })
      : null);
  const detail = [range, preview.basis].filter(Boolean).join(' ');

  return (
    <div className="pon-result">
      <ResultBaby size={170} />
      <h2 className="pon-result-hero">
        <span>{copy.lead ?? t('setup.resultLead')}</span>
        <span className="pon-result-age">{t('common.age', { weeks: preview.age.weeks, days: preview.age.days })}</span>
        <span>{copy.suffix ?? t('setup.resultTail')}</span>
      </h2>

      <PgCard>
        <div className="pon-due">
          <div className="min-w-0">
            <div className="pon-due-lbl">{copy.dueLabel ?? t('common.dueDate')}</div>
            <div className="pon-due-date">{preview.dueDateLabel ?? formatLongDate(fromApiDate(preview.dueDate), loc)}</div>
          </div>
          {confidence && (
            <span className="pon-conf">
              <Icon name="info" size={14} />
              {confidence}
            </span>
          )}
        </div>
        {detail && <p className="pon-due-detail">{detail}</p>}
      </PgCard>
    </div>
  );
}
