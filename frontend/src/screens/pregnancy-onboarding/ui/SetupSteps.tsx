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
import type { Locale } from '@/shared/i18n';
import {
  type DateParts,
  formatDayMonth,
  formatLongDate,
  formatNumber,
  fromApiDate,
  partsToDate,
} from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { AppSheet } from '@/shared/sheet';
import {
  CalendarPicker,
  Card,
  Icon,
  InfoNote,
  PillChip,
  PrimaryButton,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  WheelPicker,
} from '@/shared/ui';

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
  pilotKey,
}: {
  /** Id key of this field (`preg.setup-date:<key>`). */
  pilotKey: string;
  label: string;
  value: DateParts | null;
  onSelect: (value: DateParts) => void;
}) {
  const t = useTranslations('pregnancyV2.setup');
  const loc = useLocale() as Locale;
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" className="pgn-fieldbox" onClick={() => setOpen(true)} aria-haspopup="dialog" {...pilot(`preg.setup-date-open:${pilotKey}`)}>
        <span className="pgn-fieldbox-label">{label}</span>
        <span className={value ? 'pgn-fieldbox-value' : 'pgn-fieldbox-value is-empty'}>
          {value ? formatLongDate(partsToDate(value, loc), loc) : t('pickDate')}
        </span>
        <Icon name="calendar" size={18} className="pgn-fieldbox-icon" />
      </button>
      <AppSheet open={open} onClose={() => setOpen(false)} size="half" title={label}>
        <CalendarPicker
          value={value}
          pilot={pilot(`preg.setup-date:${pilotKey}`, 'calendar')}
          onSelect={(d) => {
            onSelect(d);
            setOpen(false);
          }}
        />
      </AppSheet>
    </>
  );
}

const WEEK_OPTIONS = Array.from({ length: 42 }, (_, i) => i + 1);
const DAY_OPTIONS = [0, 1, 2, 3, 4, 5, 6];

/** Week (1–42) + day (0–6): one field that opens two wheels in a sheet, digits in the locale's script (۸ in fa). */
function AgeFields({
  weeks,
  days,
  onChange,
  pilotKey,
}: {
  /** Id key of this field (`preg.setup-age-*:<key>`). */
  pilotKey: string;
  weeks: number | null;
  days: number;
  onChange: (weeks: number | null, days: number) => void;
}) {
  const t = useTranslations('pregnancyV2.setup');
  const loc = useLocale() as Locale;
  const [open, setOpen] = useState(false);
  const [draftWeek, setDraftWeek] = useState(0);
  const [draftDay, setDraftDay] = useState(0);
  const openSheet = () => {
    setDraftWeek(Math.max(0, WEEK_OPTIONS.indexOf(weeks ?? 8)));
    setDraftDay(Math.max(0, DAY_OPTIONS.indexOf(days)));
    setOpen(true);
  };
  return (
    <>
      <div className="pgn-pair">
        <button type="button" className="pgn-fieldbox" onClick={openSheet} aria-haspopup="dialog" {...pilot(`preg.setup-age-open:${pilotKey}`)}>
          <span className="pgn-fieldbox-label">{t('week')}</span>
          <span className={weeks ? 'pgn-fieldbox-value' : 'pgn-fieldbox-value is-empty'}>
            {weeks ? formatNumber(weeks, loc) : t('choose')}
          </span>
          <Icon name="chevronDown" size={18} className="pgn-fieldbox-icon" />
        </button>
        <button type="button" className="pgn-fieldbox" onClick={openSheet} aria-haspopup="dialog">
          <span className="pgn-fieldbox-label">{t('day')}</span>
          <span className="pgn-fieldbox-value">{formatNumber(days, loc)}</span>
          <Icon name="chevronDown" size={18} className="pgn-fieldbox-icon" />
        </button>
      </div>
      <AppSheet
        open={open}
        onClose={() => setOpen(false)}
        size="half"
        title={t('ageSheetTitle')}
        footer={
          <div className="pgn-sheet-btns">
            <SecondaryButton block={false} onClick={() => setOpen(false)} {...pilot(`preg.setup-age-cancel:${pilotKey}`)}>
              {t('cancel')}
            </SecondaryButton>
            <PrimaryButton
              block={false}
              {...pilot(`preg.setup-age-confirm:${pilotKey}`)}
              onClick={() => {
                onChange(WEEK_OPTIONS[draftWeek] ?? null, DAY_OPTIONS[draftDay] ?? 0);
                setOpen(false);
              }}
            >
              {t('confirm')}
            </PrimaryButton>
          </div>
        }
      >
        {open && (
          <div className="pgn-wheels">
            <div className="pgn-wheel-col">
              <span className="pgn-fieldbox-label">{t('week')}</span>
              <WheelPicker
                id="pgn-age-week"
                pilot={pilot(`preg.setup-age-week:${pilotKey}`, 'wheel')}
                pilotValues={WEEK_OPTIONS.map(String)}
                items={WEEK_OPTIONS.map((w) => formatNumber(w, loc))}
                selectedIndex={draftWeek}
                onChange={setDraftWeek}
              />
            </div>
            <div className="pgn-wheel-col">
              <span className="pgn-fieldbox-label">{t('day')}</span>
              <WheelPicker
                id="pgn-age-day"
                pilot={pilot(`preg.setup-age-day:${pilotKey}`, 'wheel')}
                pilotValues={DAY_OPTIONS.map(String)}
                items={DAY_OPTIONS.map((d) => formatNumber(d, loc))}
                selectedIndex={draftDay}
                onChange={setDraftDay}
              />
            </div>
          </div>
        )}
      </AppSheet>
    </>
  );
}

/** Single-choice chips for an optional field: tapping the picked chip clears it. */
function RadioChips({
  id,
  pilotId,
  label,
  options,
  value,
  onChange,
  render,
}: {
  id: string;
  pilotId: string;
  label: string;
  options: string[];
  value: string | null;
  onChange: (next: string | null) => void;
  render: (option: string) => string;
}) {
  return (
    <>
      <span id={id} className="pgn-field-label">
        {label}
      </span>
      <div className="nb-chips pgn-chips" role="radiogroup" aria-labelledby={id} {...pilot(pilotId, 'choice')}>
        {options.map((o) => (
          <button
            key={o}
            type="button"
            role="radio"
            aria-checked={value === o}
            className="nb-chip is-multi nb-tone-bloom pgn-chip pgn-radio-chip"
            {...pilotOption(o)}
            onClick={() => onChange(value === o ? null : o)}
          >
            <bdi>{render(o)}</bdi>
            {value === o && <Icon name="check" size={14} strokeWidth={2.6} className="nb-chip-check" />}
          </button>
        ))}
      </div>
    </>
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
    <Card as="section" className="pgn-sect" aria-labelledby="pgn-setup-dating">
      <h2 id="pgn-setup-dating" className="pgn-sect-title">
        {copy?.dating.title ?? t('datingTitle')}
      </h2>
      <p className="pgn-body-text">{copy?.dating.body ?? t('datingBody')}</p>
      <div className="nb-chips pgn-chips" role="radiogroup" aria-label={t('sourceLabel')} {...pilot('preg.setup-source', 'choice')}>
        {DATING_SOURCES.map((src: DatingSource) => (
          <button
            key={src}
            type="button"
            role="radio"
            aria-checked={value.source === src}
            className="nb-chip is-multi nb-tone-bloom pgn-chip pgn-radio-chip"
            {...pilotOption(src)}
            onClick={() => set({ source: src })}
          >
            {copy?.sources[src].label ?? t(`sources.${src}`)}
            {value.source === src && <Icon name="check" size={14} strokeWidth={2.6} className="nb-chip-check" />}
          </button>
        ))}
      </div>
      <p className="pgn-caption">{copy?.sources[value.source].hint ?? t(`sourceHints.${value.source}`)}</p>

      {value.source === 'lmp' && (
        <div className="pgn-cal">
          <div className="pgn-sect-head">
            <span className="pgn-field-label">{t('lmpPick')}</span>
            {value.lmp && (
              <span className="pgn-opill nb-tone-bloom" aria-live="polite">
                {formatDayMonth(partsToDate(value.lmp, loc), loc)}
              </span>
            )}
          </div>
          <CalendarPicker value={value.lmp} onSelect={(lmp) => set({ lmp })} pilot={pilot('preg.setup-lmp', 'calendar')} />
        </div>
      )}

      {value.source === 'ultrasound' && (
        <>
          <DateField pilotKey="ultrasound" label={t('ultrasoundDate')} value={value.ultrasoundDate} onSelect={(d) => set({ ultrasoundDate: d })} />
          <span className="pgn-field-label">{t('ultrasoundAge')}</span>
          <AgeFields
            pilotKey="ultrasound"
            weeks={value.ultrasoundWeeks}
            days={value.ultrasoundDays}
            onChange={(w, d) => set({ ultrasoundWeeks: w, ultrasoundDays: d })}
          />
        </>
      )}

      {value.source === 'manual' && (
        <>
          <span className="pgn-field-label">{t('manualAge')}</span>
          <AgeFields
            pilotKey="manual"
            weeks={value.manualWeeks}
            days={value.manualDays}
            onChange={(w, d) => set({ manualWeeks: w, manualDays: d })}
          />
        </>
      )}
    </Card>
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
    <>
      <Card as="section" className="pgn-sect" aria-labelledby="pgn-setup-history">
        <h2 id="pgn-setup-history" className="pgn-sect-title">
          {copy?.history.title ?? t('historyTitle')}
        </h2>
        <p className="pgn-body-text">{copy?.history.body ?? t('historyBody')}</p>
        <div className="nb-chips pgn-chips">
          <PillChip
            mode="multi"
            tone="bloom"
            className="pgn-chip"
            {...pilot('preg.setup-miscarriage', 'toggle')}
            pressed={value.miscarriage}
            onPressedChange={(on) => set({ miscarriage: on })}
          >
            {t('miscarriage')}
          </PillChip>
          <PillChip
            mode="multi"
            tone="bloom"
            className="pgn-chip"
            {...pilot('preg.setup-high-risk', 'toggle')}
            pressed={value.highRisk}
            onPressedChange={(on) => set({ highRisk: on })}
          >
            {t('highRisk')}
          </PillChip>
        </div>
        <span id="pgn-setup-conditions" className="pgn-field-label">
          {t('conditions')}
        </span>
        <div className="nb-chips pgn-chips" role="group" aria-labelledby="pgn-setup-conditions" {...pilot('preg.setup-conditions', 'choice')}>
          {conditions.map((c) => (
            <PillChip
              key={c}
              mode="multi"
              tone="bloom"
              className="pgn-chip"
              {...pilotOption(c)}
              pressed={value.conditions.includes(c)}
              onPressedChange={() => set({ conditions: toggleCondition(value.conditions, c) })}
            >
              {c === 'none' ? t('conditionNone') : tp(`onboarding.conditions.${c}`)}
            </PillChip>
          ))}
        </div>
        <RadioChips
          id="pgn-setup-blood"
          pilotId="preg.setup-blood"
          label={t('bloodGroup')}
          options={bloodTypes}
          value={value.bloodType}
          onChange={(bloodType) => set({ bloodType })}
          render={(b) => b}
        />
        <RadioChips
          id="pgn-setup-rh"
          pilotId="preg.setup-rh"
          label={t('rh')}
          options={rhFactors}
          value={value.rhFactor}
          onChange={(rhFactor) => set({ rhFactor })}
          render={(r) => (r === 'negative' ? t('rhNegative') : t('rhPositive'))}
        />
      </Card>
      {/* Truthful copy: history is stored on the server (README open point 1). */}
      <InfoNote icon="shield">{copy?.history.disclaimer ?? t('historyDisclaimer')}</InfoNote>
    </>
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

  if (failed)
    return (
      <p role="alert" className="pgn-error-text">
        {t('setup.previewError')}
      </p>
    );
  if (loading || !preview)
    return (
      <SkeletonGroup label={t('setup.calculating')} className="pgn-skel">
        <Skeleton shape="card" className="pgn-skel-hero" />
      </SkeletonGroup>
    );

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
  const due = preview.dueDateLabel ?? formatLongDate(fromApiDate(preview.dueDate), loc);

  return (
    <Card as="section" className="pgn-result" aria-live="polite">
      <span className="pgn-caption">{copy.lead ?? t('setup.resultLead')}</span>
      <span className="pgn-display is-lg">{t('common.age', { weeks: preview.age.weeks, days: preview.age.days })}</span>
      <span className="pgn-result-line">
        {copy.suffix ?? t('setup.resultTail')} · {copy.dueLabel ?? t('common.dueDate')} {due}
      </span>
      {confidence && <span className="pgn-opill nb-tone-warm">{confidence}</span>}
      {detail && <p className="pgn-caption">{detail}</p>}
    </Card>
  );
}
