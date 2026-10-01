'use client';

import { useLocale } from 'next-intl';
import { useId, useState, type ReactNode } from 'react';
import { clsx } from 'clsx';

import type { Locale } from '@/shared/i18n';
import { formatLongDate, formatNumber } from '@/shared/lib/date';
import { toAsciiDigits } from '@/shared/lib/phone';
import { Icon } from '@/shared/ui';

interface NumFieldProps {
  label: string;
  value: number | null;
  onChange: (value: number | null) => void;
  min: number;
  max: number;
}

/** Label-inside numeric box («هفته ۷», «روز ۳») — locale digits in, clamped number out. */
export function NumField({ label, value, onChange, min, max }: NumFieldProps) {
  const locale = useLocale() as Locale;
  const id = useId();
  return (
    <label className="onb2-num" htmlFor={id}>
      <span className="onb2-num-label">{label}</span>
      <input
        id={id}
        className="onb2-num-input"
        inputMode="numeric"
        value={value == null ? '' : formatNumber(value, locale)}
        // Typing replaces the number instead of appending to it.
        onFocus={(e) => e.target.select()}
        onChange={(e) => {
          const digits = toAsciiDigits(e.target.value).replace(/\D/g, '');
          if (!digits) return onChange(null);
          onChange(Math.min(max, Math.max(min, Number(digits.slice(-2)))));
        }}
      />
    </label>
  );
}

interface DateRowProps {
  label: string;
  value: Date | null;
  placeholder: string;
  /** Extra at the end of the row («۳۳ سال»). */
  trailing?: ReactNode;
  icon?: 'cake' | 'calendar';
  /** The picker shown under the row while it is open. */
  children: (close: () => void) => ReactNode;
}

/** A tappable date row that opens its picker inline (no navigation, no sheet). */
export function DateRow({ label, value, placeholder, trailing, icon, children }: DateRowProps) {
  const locale = useLocale() as Locale;
  const panelId = useId();
  const [open, setOpen] = useState(false);
  return (
    <div className="onb2-daterow-wrap">
      <button
        type="button"
        className={clsx('onb2-daterow', icon && 'has-icon')}
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setOpen((o) => !o)}
      >
        {icon ? (
          <span className="onb2-daterow-disc" aria-hidden>
            <Icon name={icon} size={22} strokeWidth={1.8} />
          </span>
        ) : null}
        <span className="onb2-daterow-text">
          <span className="onb2-daterow-label">{label}</span>
          <span className={clsx('onb2-daterow-value', !value && 'is-empty')}>
            {value ? formatLongDate(value, locale) : placeholder}
          </span>
        </span>
        {trailing ? <span className="onb2-daterow-trailing">{trailing}</span> : null}
      </button>
      <div id={panelId} hidden={!open} className="onb2-daterow-panel">
        {open ? children(() => setOpen(false)) : null}
      </div>
    </div>
  );
}
