'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type ReactNode, useState } from 'react';

import type { Locale } from '@/shared/i18n';
import { type DateParts, formatNumber, fromApiDate, partsToDate, toApiDate, toParts } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, PrimaryButton, SecondaryButton, WheelPicker } from '@/shared/ui';

import { parseClock, toClock } from '../model/form';

const HOURS = Array.from({ length: 24 }, (_, i) => i);
const MINUTES = Array.from({ length: 12 }, (_, i) => i * 5);
const pad = (n: number) => String(n).padStart(2, '0');

function Footer({ onCancel, onDone, disabled }: { onCancel: () => void; onDone: () => void; disabled?: boolean }) {
  const t = useTranslations('ivf.meds.form');
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
  const t = useTranslations('ivf.meds.form');
  const locale = useLocale() as Locale;
  const initial = parseClock(value);
  const [hour, setHour] = useState(initial.hour);
  const [minuteIndex, setMinuteIndex] = useState(Math.min(MINUTES.length - 1, Math.round(initial.minute / 5)));
  return (
    <AppSheet
      open
      onClose={onClose}
      size="half"
      title={title}
      footer={<Footer onCancel={onClose} onDone={() => onPick(toClock(hour, MINUTES[minuteIndex] ?? 0))} />}
    >
      {/* Clock digits read left-to-right in every locale. */}
      <div className="ivfm-wheels" dir="ltr">
        <WheelPicker
          id="ivfm-time-hour"
          items={HOURS.map((h) => formatNumber(pad(h), locale))}
          selectedIndex={hour}
          onChange={setHour}
        />
        <span className="ivfm-wheels-sep" aria-hidden>
          :
        </span>
        <WheelPicker
          id="ivfm-time-minute"
          items={MINUTES.map((m) => formatNumber(pad(m), locale))}
          selectedIndex={minuteIndex}
          onChange={setMinuteIndex}
        />
      </div>
      <p className="sr-only">
        {t('hour')} / {t('minute')}
      </p>
    </AppSheet>
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

/** «حذف دارو؟» confirmation. */
export function ConfirmSheet({
  title,
  body,
  confirm,
  busy,
  error,
  onClose,
  onConfirm,
}: {
  title: string;
  body: string;
  confirm: string;
  busy: boolean;
  error: ReactNode;
  onClose: () => void;
  onConfirm: () => void;
}) {
  const t = useTranslations('ivf.meds.form');
  return (
    <AppSheet
      open
      onClose={onClose}
      size="half"
      title={title}
      footer={
        <div className="ivfm-sheet-btns">
          <SecondaryButton block={false} onClick={onClose}>
            {t('cancel')}
          </SecondaryButton>
          <PrimaryButton block={false} className="ivfm-danger-btn" loading={busy} onClick={onConfirm}>
            {confirm}
          </PrimaryButton>
        </div>
      }
    >
      <p className="ivfm-sheet-body">{body}</p>
      {error}
    </AppSheet>
  );
}
