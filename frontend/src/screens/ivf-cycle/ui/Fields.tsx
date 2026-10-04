'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import type { Locale } from '@/shared/i18n';
import { type DateParts, formatNumber, fromApiDate, partsToDate, toApiDate, toParts } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, ChipGroup, Icon, PillChip, PrimaryButton, SecondaryButton, WheelPicker } from '@/shared/ui';

/*
 * Form pieces of the cycle setup + editor (CB-IVF-06b). They reuse the
 * medicine form's `ivfm-*` look (CB-IVF-03 CSS) so the IVF forms read as one.
 */

const HOURS = Array.from({ length: 24 }, (_, i) => i);
const MINUTES = Array.from({ length: 12 }, (_, i) => i * 5);
const pad = (n: number) => String(n).padStart(2, '0');

function Footer({ onCancel, onDone, disabled }: { onCancel: () => void; onDone: () => void; disabled?: boolean }) {
  const t = useTranslations('ivf.cycle');
  return (
    <div className="ivfm-sheet-btns">
      <SecondaryButton block={false} onClick={onCancel}>
        {t('cancel')}
      </SecondaryButton>
      <PrimaryButton block={false} onClick={onDone} disabled={disabled}>
        {t('done')}
      </PrimaryButton>
    </div>
  );
}

/** A date in the locale's calendar (Jalali in fa) → `Y-m-d`. Mounted only while open. */
export function DateSheet({
  title,
  value,
  onClose,
  onPick,
}: {
  title: string;
  value: string | null;
  onClose: () => void;
  onPick: (apiDate: string) => void;
}) {
  const locale = useLocale() as Locale;
  const [parts, setParts] = useState<DateParts | null>(value ? toParts(fromApiDate(value), locale) : null);
  return (
    <AppSheet
      open
      onClose={onClose}
      size="half"
      title={title}
      footer={
        <Footer
          onCancel={onClose}
          disabled={!parts}
          onDone={() => parts && onPick(toApiDate(partsToDate(parts, locale)))}
        />
      }
    >
      <CalendarPicker value={parts} onSelect={setParts} />
    </AppSheet>
  );
}

/** Hour : minute wheels (5-minute steps; Tehran wall clock). Mounted only while open. */
export function TimeSheet({
  title,
  value,
  onClose,
  onPick,
}: {
  title: string;
  value: string;
  onClose: () => void;
  onPick: (clock: string) => void;
}) {
  const locale = useLocale() as Locale;
  const match = /^(\d{2}):(\d{2})$/.exec(value);
  const [hour, setHour] = useState(match ? Math.min(23, Number(match[1])) : 9);
  const [minuteIndex, setMinuteIndex] = useState(
    match ? Math.min(MINUTES.length - 1, Math.round(Number(match[2]) / 5)) : 0,
  );
  return (
    <AppSheet
      open
      onClose={onClose}
      size="half"
      title={title}
      footer={<Footer onCancel={onClose} onDone={() => onPick(`${pad(hour)}:${pad(MINUTES[minuteIndex] ?? 0)}`)} />}
    >
      {/* Clock digits read left-to-right in every locale. */}
      <div className="ivfm-wheels" dir="ltr">
        <WheelPicker
          id="ivfc-time-hour"
          items={HOURS.map((h) => formatNumber(pad(h), locale))}
          selectedIndex={hour}
          onChange={setHour}
        />
        <span className="ivfm-wheels-sep" aria-hidden>
          :
        </span>
        <WheelPicker
          id="ivfc-time-minute"
          items={MINUTES.map((m) => formatNumber(pad(m), locale))}
          selectedIndex={minuteIndex}
          onChange={setMinuteIndex}
        />
      </div>
    </AppSheet>
  );
}

/** «label … value» field that opens a picker (dates, times), with an optional clear button. */
export function FieldButton({
  label,
  hint,
  value,
  invalid,
  ltr,
  onClick,
  onClear,
  clearLabel,
}: {
  label: string;
  hint?: string;
  value: string;
  invalid?: boolean;
  ltr?: boolean;
  onClick: () => void;
  onClear?: () => void;
  clearLabel?: string;
}) {
  return (
    <div className={clsx('ivfm-pick', invalid && 'is-invalid')}>
      <button type="button" className="ivfm-pick-btn" onClick={onClick}>
        <span className="ivfm-pick-label">
          {label}
          {hint ? <span className="ivfm-optional"> {hint}</span> : null}
        </span>
        <span className="ivfm-pick-value" dir={ltr ? 'ltr' : undefined}>
          {value}
        </span>
      </button>
      {onClear ? (
        <button type="button" className="ivfm-pick-clear" aria-label={clearLabel} onClick={onClear}>
          <Icon name="x" size={14} />
        </button>
      ) : null}
    </div>
  );
}

/** A labelled single-choice chip row; `allowNone` lets a second tap clear the choice. */
export function ChoiceRow<V extends string>({
  label,
  hint,
  options,
  value,
  labelOf,
  onChange,
  allowNone,
}: {
  label: string;
  hint?: string;
  options: readonly V[];
  value: V | null;
  labelOf: (value: V) => string;
  onChange: (value: V | null) => void;
  allowNone?: boolean;
}) {
  return (
    <div className="ivfm-group">
      <span className="ivfm-label">{label}</span>
      {hint ? <span className="ivfm-block-hint">{hint}</span> : null}
      <ChipGroup label={label} className="ivfm-chips">
        {options.map((option) => (
          <PillChip
            key={option}
            pressed={option === value}
            onPressedChange={() => onChange(option === value && allowNone ? null : option)}
          >
            {labelOf(option)}
          </PillChip>
        ))}
      </ChipGroup>
    </div>
  );
}

/** The error line under a field. */
export function FieldError({ text }: { text: string | null }) {
  return text ? (
    <span className="ivfm-error" role="alert">
      {text}
    </span>
  ) : null;
}
