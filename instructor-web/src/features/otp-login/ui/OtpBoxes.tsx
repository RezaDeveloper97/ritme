'use client';

import { useRef } from 'react';

import { cn, digitsOnly } from '@/shared/lib';

import { OTP_LENGTH } from '../model/otp';

interface OtpBoxesProps {
  value: string;
  onChange: (value: string) => void;
  label: string;
  invalid?: boolean;
  disabled?: boolean;
}

/**
 * One real input (SMS autofill via autocomplete="one-time-code", paste, screen
 * readers) laid transparently over four display boxes.
 */
export function OtpBoxes({ value, onChange, label, invalid, disabled }: OtpBoxesProps) {
  const ref = useRef<HTMLInputElement>(null);
  const cells = Array.from({ length: OTP_LENGTH }, (_, i) => value[i] ?? '');
  const active = Math.min(value.length, OTP_LENGTH - 1);
  return (
    <div className={cn('otp', invalid && 'is-invalid')} onClick={() => ref.current?.focus()}>
      <input
        ref={ref}
        className="otp__input"
        value={value}
        onChange={(e) => onChange(digitsOnly(e.target.value).slice(0, OTP_LENGTH))}
        inputMode="numeric"
        autoComplete="one-time-code"
        autoFocus
        maxLength={OTP_LENGTH}
        aria-label={label}
        aria-invalid={invalid || undefined}
        disabled={disabled}
        dir="ltr"
      />
      <div className="otp__cells" dir="ltr" aria-hidden="true">
        {cells.map((ch, i) => (
          <span key={i} className={cn('otp__cell', ch && 'is-filled', i === active && 'is-active')}>
            {ch}
          </span>
        ))}
      </div>
    </div>
  );
}
