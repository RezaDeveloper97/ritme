'use client';

import { clsx } from 'clsx';
import { useId } from 'react';

import { normalizeCompanionCode } from '../lib/code';
import { COMPANION_CODE_LENGTH } from '../model/companion-side';

interface CompanionCodeFieldProps {
  value: string;
  onChange: (code: string) => void;
  /** Enter on a complete code. */
  onSubmit?: () => void;
  /** Accessible name of the field («کد ۶ کاراکتری همدم»). */
  label: string;
  /** Hint read with the field (`aria-describedby`). */
  describedBy?: string;
  invalid?: boolean;
  disabled?: boolean;
  autoFocus?: boolean;
  className?: string;
}

/**
 * The 6-character companion code (nbl_Onb_Partner): six boxes painted over one
 * real text input, so typing, deleting, paste and IME all behave like a normal
 * field. Letters are upper-cased and anything that isn't A–Z / 0–9 is dropped
 * (`normalizeCompanionCode`); Persian digits are read as ASCII. Always LTR.
 */
export function CompanionCodeField({
  value,
  onChange,
  onSubmit,
  label,
  describedBy,
  invalid,
  disabled,
  autoFocus,
  className,
}: CompanionCodeFieldProps) {
  const id = useId();
  const chars = value.split('');
  const caret = Math.min(value.length, COMPANION_CODE_LENGTH - 1);
  return (
    <div dir="ltr" className={clsx('cmh-code', invalid && 'is-invalid', className)}>
      <input
        id={id}
        className="cmh-code-input"
        value={value}
        onChange={(e) => onChange(normalizeCompanionCode(e.target.value))}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && value.length === COMPANION_CODE_LENGTH) onSubmit?.();
        }}
        aria-label={label}
        aria-describedby={describedBy}
        aria-invalid={invalid || undefined}
        inputMode="text"
        autoCapitalize="characters"
        autoComplete="off"
        autoCorrect="off"
        spellCheck={false}
        maxLength={COMPANION_CODE_LENGTH + 24}
        disabled={disabled}
        // The field is the screen's only task; focusing it opens the keyboard on arrival.
         
        autoFocus={autoFocus}
      />
      <div className="cmh-code-boxes" aria-hidden>
        {Array.from({ length: COMPANION_CODE_LENGTH }, (_, i) => (
          <span
            key={i}
            className={clsx('cmh-code-box', chars[i] && 'is-filled', i === caret && 'is-caret')}
          >
            {chars[i] ?? ''}
          </span>
        ))}
      </div>
    </div>
  );
}
