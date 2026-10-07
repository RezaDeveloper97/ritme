'use client';

import { useLocale } from 'next-intl';
import { useState } from 'react';

import type { Locale } from '@/shared/i18n';
import { formatLongDate, fromApiDate, partsToDate, toApiDate, toParts } from '@/shared/lib/date';
import { CalendarPicker, SecondaryButton } from '@/shared/ui';

/**
 * An optional date of a record document form (CB-REC-04): a field button that opens the locale's calendar (Jalali
 * in fa) inline, plus «بدون تاریخ». Inline rather than a second sheet — `AppSheet` does not portal, so a sheet
 * inside a sheet would be clipped by its parent.
 */
export function RecordDateField({
  label,
  value,
  unsetLabel,
  clearLabel,
  disabled,
  onChange,
}: {
  label: string;
  value: string | null;
  unsetLabel: string;
  clearLabel: string;
  disabled?: boolean;
  onChange: (apiDate: string | null) => void;
}) {
  const locale = useLocale() as Locale;
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" className="lab-up-field" aria-expanded={open} disabled={disabled} onClick={() => setOpen((o) => !o)}>
        <span className="lab-up-field-label">{label}</span>
        <span className="lab-up-field-value">{value ? formatLongDate(fromApiDate(value), locale) : unsetLabel}</span>
      </button>
      {open ? (
        <div className="rec-calendar">
          <CalendarPicker
            value={value ? toParts(fromApiDate(value), locale) : null}
            onSelect={(p) => {
              onChange(toApiDate(partsToDate(p, locale)));
              setOpen(false);
            }}
          />
          <SecondaryButton
            variant="text"
            onClick={() => {
              onChange(null);
              setOpen(false);
            }}
          >
            {clearLabel}
          </SecondaryButton>
        </div>
      ) : null}
    </>
  );
}
