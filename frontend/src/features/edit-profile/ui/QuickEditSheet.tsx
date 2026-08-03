'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { getApiErrorMessage } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import {
  allMonthNames,
  birthYearRange,
  daysInCalendarMonth,
  type DateParts,
  formatNumber,
  toApiDate,
  today,
  toParts,
  todayParts,
} from '@/shared/lib/date';
import { RulerPicker, Sheet, WheelPicker } from '@/shared/ui';

import { useUpdateProfile, type UpdateProfileInput } from '../api/mutations';
import { datePartsToApiDate } from '../lib/datePartsToApiDate';

/** The single profile value a quick-edit sheet is opened for. */
export type QuickEditField = 'cycleDuration' | 'periodDuration' | 'birthday' | 'weight' | 'height';

/** Current values the sheet seeds its picker from (all optional — see fallbacks). */
export interface QuickEditValues {
  cycleDuration?: number | null;
  periodDuration?: number | null;
  /** Gregorian ISO from the API; edited in the locale's calendar (CLAUDE.md §7). */
  birthday?: string | null;
  weight?: number | null;
  height?: number | null;
}

// API bounds (POST /profile validation).
const CYCLE_MIN = 15;
const CYCLE_MAX = 60;
const PERIOD_MIN = 1;
const PERIOD_MAX = 15;
const WEIGHT_MIN = 30;
const WEIGHT_MAX = 200;
const HEIGHT_MIN = 100;
const HEIGHT_MAX = 220;

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v));

/** Single "N days" wheel bounded by the API's validation range. */
function DaysWheel({
  id,
  min,
  max,
  value,
  onChange,
  label,
}: {
  id: string;
  min: number;
  max: number;
  value: number;
  onChange: (v: number) => void;
  label: (days: number) => string;
}) {
  const items = Array.from({ length: max - min + 1 }, (_, i) => label(min + i));
  return (
    <div className="prof-wheel-center">
      <div className="wheel-band prof-wheel-band" />
      <WheelPicker
        id={id}
        items={items}
        selectedIndex={clamp(value, min, max) - min}
        width={150}
        onChange={(i) => onChange(min + i)}
      />
    </div>
  );
}

/**
 * Day / month / year wheels for the birthday, in the locale's own calendar
 * (Jalali for `fa`, Gregorian for `en`) — the shared date layer owns it (§7).
 */
function BirthWheels({
  value,
  onChange,
  loc,
}: {
  value: DateParts;
  onChange: (parts: DateParts) => void;
  loc: Locale;
}) {
  const { min: minYear, max: maxYear } = birthYearRange(loc);
  const months = allMonthNames(loc);
  const dayCount = daysInCalendarMonth(value.year, value.month, loc);
  const days = Array.from({ length: dayCount }, (_, i) => formatNumber(i + 1, loc));
  const years = Array.from({ length: maxYear - minYear + 1 }, (_, i) =>
    formatNumber(minYear + i, loc),
  );

  // The wheels are remounted per (month, year) so a shortened month re-seeds its
  // day list instead of keeping a now-invalid index.
  return (
    <div className="jdw-row">
      <div className="wheel-band" />
      <WheelPicker
        id={`pq-day-${value.year}-${value.month}`}
        items={days}
        selectedIndex={clamp(value.day, 1, dayCount) - 1}
        width={56}
        onChange={(i) => onChange({ ...value, day: i + 1 })}
      />
      <WheelPicker
        id="pq-month"
        items={[...months]}
        selectedIndex={clamp(value.month, 1, 12) - 1}
        width={112}
        onChange={(i) => onChange({ ...value, month: i + 1 })}
      />
      <WheelPicker
        id="pq-year"
        items={years}
        selectedIndex={clamp(value.year, minYear, maxYear) - minYear}
        width={80}
        onChange={(i) => onChange({ ...value, year: minYear + i })}
      />
    </div>
  );
}

/**
 * Bottom sheet that edits exactly one profile value in place — the tap target
 * being the corresponding stat row on the profile screen. Each field gets the
 * same picker it has on the full edit screens (wheels for durations and the
 * birthday wheels, ruler for weight/height) so the two paths stay consistent.
 *
 * Sensitive health data (§11): shown and submitted, never logged.
 */
export function QuickEditSheet({
  field,
  values,
  onClose,
}: {
  /** `null` closes the sheet; a field opens it seeded from `values`. */
  field: QuickEditField | null;
  values: QuickEditValues;
  onClose: () => void;
}) {
  return field ? (
    // Remounting per field resets the pickers, which read their position on
    // mount, to the value the user just tapped.
    <QuickEditForm key={field} field={field} values={values} onClose={onClose} />
  ) : null;
}

function QuickEditForm({
  field,
  values,
  onClose,
}: {
  field: QuickEditField;
  values: QuickEditValues;
  onClose: () => void;
}) {
  const t = useTranslations('profileEdit');
  const loc = useLocale() as Locale;
  const update = useUpdateProfile();
  const now = todayParts(loc);

  const [cycleDuration, setCycleDuration] = useState(() =>
    clamp(values.cycleDuration ?? 28, CYCLE_MIN, CYCLE_MAX),
  );
  const [periodDuration, setPeriodDuration] = useState(() =>
    clamp(values.periodDuration ?? 6, PERIOD_MIN, PERIOD_MAX),
  );
  const [birth, setBirth] = useState<DateParts>(() =>
    values.birthday
      ? toParts(new Date(values.birthday), loc)
      : { year: now.year - 25, month: 1, day: 1 },
  );
  const [weight, setWeight] = useState(() => clamp(values.weight ?? 60, WEIGHT_MIN, WEIGHT_MAX));
  const [height, setHeight] = useState(() => clamp(values.height ?? 165, HEIGHT_MIN, HEIGHT_MAX));
  const [localError, setLocalError] = useState<string | null>(null);

  const titleKey = {
    cycleDuration: 'health.cycleLabel',
    periodDuration: 'health.periodLabel',
    birthday: 'personal.birthdayLabel',
    weight: 'personal.weightLabel',
    height: 'personal.heightLabel',
  } as const;
  const title = t(titleKey[field]);

  const handleSave = () => {
    if (update.isPending) return;
    setLocalError(null);

    let payload: UpdateProfileInput;
    if (field === 'cycleDuration') {
      payload = { cycle_duration: cycleDuration };
    } else if (field === 'periodDuration') {
      payload = { period_duration: periodDuration };
    } else if (field === 'weight') {
      payload = { weight: Math.round(weight) };
    } else if (field === 'height') {
      payload = { height: Math.round(height) };
    } else {
      const birthday = datePartsToApiDate(birth, loc);
      // The API requires a birthday strictly before today; catch it client-side
      // so the user sees a localized message instead of a raw 422.
      if (birthday >= toApiDate(today())) {
        setLocalError(t('errors.birthdayFuture'));
        return;
      }
      payload = { birthday };
    }

    update.mutate(payload, { onSuccess: onClose });
  };

  const errorMessage =
    localError ?? (update.isError ? (getApiErrorMessage(update.error) ?? t('errors.generic')) : null);

  return (
    <Sheet open onClose={onClose} labelledBy="prof-quick-title">
      <h2 id="prof-quick-title" className="prof-quick-t">
        {title}
      </h2>

      <div className="prof-quick-body">
        {field === 'cycleDuration' ? (
          <DaysWheel
            id="pq-cycle"
            min={CYCLE_MIN}
            max={CYCLE_MAX}
            value={cycleDuration}
            onChange={setCycleDuration}
            label={(days) => t('health.dayCount', { days })}
          />
        ) : null}

        {field === 'periodDuration' ? (
          <DaysWheel
            id="pq-period"
            min={PERIOD_MIN}
            max={PERIOD_MAX}
            value={periodDuration}
            onChange={setPeriodDuration}
            label={(days) => t('health.dayCount', { days })}
          />
        ) : null}

        {field === 'birthday' ? (
          <BirthWheels value={birth} onChange={setBirth} loc={loc} />
        ) : null}

        {field === 'weight' ? (
          <RulerPicker
            min={WEIGHT_MIN}
            max={WEIGHT_MAX}
            value={weight}
            unit={t('personal.kgUnit')}
            onChange={setWeight}
            toDisplay={(v) => formatNumber(v, loc)}
          />
        ) : null}

        {field === 'height' ? (
          <RulerPicker
            min={HEIGHT_MIN}
            max={HEIGHT_MAX}
            value={height}
            unit={t('personal.cmUnit')}
            onChange={setHeight}
            toDisplay={(v) => formatNumber(v, loc)}
          />
        ) : null}
      </div>

      <div className="prof-quick-actions">
        {errorMessage ? (
          <p role="alert" className="prof-edit-error">
            {errorMessage}
          </p>
        ) : null}
        <button className="btn btn-primary" onClick={handleSave} disabled={update.isPending}>
          {update.isPending ? t('saving') : t('save')}
        </button>
      </div>
    </Sheet>
  );
}
