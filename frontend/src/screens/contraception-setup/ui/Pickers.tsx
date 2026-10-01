'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import type { Locale } from '@/shared/i18n';
import { type DateParts, formatNumber, fromApiDate, partsToDate, toApiDate, toParts } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, PrimaryButton, SecondaryButton, WheelPicker } from '@/shared/ui';

import { parseClock, toClock } from '../model/draft';

const HOURS = Array.from({ length: 24 }, (_, i) => i);
const MINUTES = Array.from({ length: 60 }, (_, i) => i);
const pad = (n: number) => String(n).padStart(2, '0');

function Footer({ onCancel, onDone, disabled }: { onCancel: () => void; onDone: () => void; disabled?: boolean }) {
  const t = useTranslations('contraception.setup');
  return (
    <div className="ctr-sheet-btns">
      <SecondaryButton block={false} onClick={onCancel}>
        {t('cancel')}
      </SecondaryButton>
      <PrimaryButton block={false} onClick={onDone} disabled={disabled}>
        {t('done')}
      </PrimaryButton>
    </div>
  );
}

/** Hour : minute wheels for the pill reminder (Tehran wall clock). Mounted only while open. */
export function TimeSheet({ value, onClose, onPick }: { value: string; onClose: () => void; onPick: (clock: string) => void }) {
  const t = useTranslations('contraception.setup');
  const locale = useLocale() as Locale;
  const initial = parseClock(value);
  const [hour, setHour] = useState(initial.hour);
  const [minute, setMinute] = useState(initial.minute);
  return (
    <AppSheet
      open
      onClose={onClose}
      size="half"
      title={t('reminderTime')}
      footer={<Footer onCancel={onClose} onDone={() => onPick(toClock(hour, minute))} />}
    >
      {/* Clock digits read left-to-right in every locale. */}
      <div className="ctr-wheels" dir="ltr">
        <WheelPicker
          id="ctr-time-hour"
          items={HOURS.map((h) => formatNumber(pad(h), locale))}
          selectedIndex={hour}
          onChange={setHour}
        />
        <span className="ctr-wheels-sep" aria-hidden>
          :
        </span>
        <WheelPicker
          id="ctr-time-minute"
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
