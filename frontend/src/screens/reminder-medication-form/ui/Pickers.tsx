'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import type { MedicationDuration } from '@/entities/care-reminder';
import type { Locale } from '@/shared/i18n';
import {
  type DateParts,
  formatNumber,
  fromApiDate,
  partsToDate,
  toApiDate,
  toParts,
} from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, PrimaryButton, SecondaryButton, WheelPicker } from '@/shared/ui';

import { parseSlot, toSlot } from '../model/form';

const HOURS = Array.from({ length: 24 }, (_, i) => i);
const MINUTES = Array.from({ length: 60 }, (_, i) => i);
const pad = (n: number) => String(n).padStart(2, '0');

function SheetFooter({ onCancel, onDone, disabled }: { onCancel: () => void; onDone: () => void; disabled?: boolean }) {
  const t = useTranslations('care.medicationForm');
  return (
    <div className="flex gap-2.5">
      <SecondaryButton block={false} className="flex-1" onClick={onCancel}>
        {t('cancel')}
      </SecondaryButton>
      <PrimaryButton block={false} className="flex-1" onClick={onDone} disabled={disabled}>
        {t('done')}
      </PrimaryButton>
    </div>
  );
}

/** Hour : minute wheels (the app's time-wheel picker) in a half sheet. */
export function TimeSheet({
  open,
  value,
  onClose,
  onPick,
}: {
  open: boolean;
  value: string;
  onClose: () => void;
  onPick: (slot: string) => void;
}) {
  const t = useTranslations('care.medicationForm');
  const locale = useLocale() as Locale;
  const initial = parseSlot(value);
  const [hour, setHour] = useState(initial.hour);
  const [minute, setMinute] = useState(initial.minute);

  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="half"
      title={t('pickTime')}
      footer={<SheetFooter onCancel={onClose} onDone={() => onPick(toSlot(hour, minute))} />}
    >
      {/* Clock digits read left-to-right in every locale. */}
      <div className="flex items-center justify-center gap-3 py-2" dir="ltr">
        <WheelPicker
          id="med-time-hour"
          items={HOURS.map((h) => formatNumber(pad(h), locale))}
          selectedIndex={hour}
          onChange={setHour}
        />
        <span className="text-xl font-extrabold text-(--muted)" aria-hidden>
          :
        </span>
        <WheelPicker
          id="med-time-minute"
          items={MINUTES.map((m) => formatNumber(pad(m), locale))}
          selectedIndex={minute}
          onChange={setMinute}
        />
      </div>
      <p className="sr-only">
        {t('hour')} / {t('minute')}
      </p>
    </AppSheet>
  );
}

/** A locale-calendar date (Jalali in fa) → `Y-m-d`. */
export function DateSheet({
  open,
  title,
  value,
  onClose,
  onPick,
}: {
  open: boolean;
  title: string;
  value: string | null;
  onClose: () => void;
  onPick: (apiDate: string) => void;
}) {
  const locale = useLocale() as Locale;
  const [parts, setParts] = useState<DateParts | null>(value ? toParts(fromApiDate(value), locale) : null);

  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="half"
      title={title}
      footer={
        <SheetFooter
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

/** «مدت مصرف»: ongoing / until a date / until the end of pregnancy. */
export function DurationSheet({
  open,
  options,
  value,
  endsOn,
  onClose,
  onPick,
}: {
  open: boolean;
  options: MedicationDuration[];
  value: MedicationDuration;
  endsOn: string | null;
  onClose: () => void;
  onPick: (duration: MedicationDuration, endsOn: string | null) => void;
}) {
  const t = useTranslations('care.medicationForm');
  const locale = useLocale() as Locale;
  const [duration, setDuration] = useState<MedicationDuration>(value);
  const [parts, setParts] = useState<DateParts | null>(endsOn ? toParts(fromApiDate(endsOn), locale) : null);
  const needsDate = duration === 'until_date';

  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="half"
      title={t('duration')}
      footer={
        <SheetFooter
          onCancel={onClose}
          disabled={needsDate && !parts}
          onDone={() =>
            onPick(duration, needsDate && parts ? toApiDate(partsToDate(parts, locale)) : null)
          }
        />
      }
    >
      <div className="flex flex-col gap-2.5" role="radiogroup" aria-label={t('duration')}>
        {options.map((opt) => (
          <button
            key={opt}
            type="button"
            role="radio"
            aria-checked={duration === opt}
            className={clsx('chip justify-between', duration === opt && 'on')}
            onClick={() => setDuration(opt)}
          >
            {t(`durations.${opt}`)}
          </button>
        ))}
      </div>
      {needsDate && (
        <div className="mt-3">
          <p className="fld-label-t">{t('endsOn')}</p>
          <CalendarPicker value={parts} onSelect={setParts} />
        </div>
      )}
    </AppSheet>
  );
}

/** Delete confirm for edit mode. */
export function DeleteSheet({
  open,
  pending,
  error,
  onClose,
  onConfirm,
}: {
  open: boolean;
  pending: boolean;
  error: boolean;
  onClose: () => void;
  onConfirm: () => void;
}) {
  const t = useTranslations('care.medicationForm');
  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="half"
      title={t('deleteTitle')}
      footer={
        <div className="flex gap-2.5">
          <SecondaryButton block={false} className="flex-1" onClick={onClose}>
            {t('cancel')}
          </SecondaryButton>
          <PrimaryButton block={false} className="flex-1 rmd-danger-btn" onClick={onConfirm} loading={pending}>
            {pending ? t('deleting') : t('delete')}
          </PrimaryButton>
        </div>
      }
    >
      <p className="sub">{t('deleteBody')}</p>
      {error && (
        <p role="alert" className="mt-2 text-[12.5px] font-bold text-(--danger-deep)">
          {t('deleteError')}
        </p>
      )}
    </AppSheet>
  );
}
