'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type ReactNode, useEffect, useRef } from 'react';

import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

interface PasscodePadProps {
  /** Digits expected; `onComplete` fires when this many are entered. */
  length: number;
  value: string;
  onChange: (next: string) => void;
  onComplete: (value: string) => void;
  /** Error line under the dots (wrong passcode, mismatch, back-off). */
  error?: ReactNode;
  disabled?: boolean;
  /** Optional key in the bottom-start slot (biometric unlock). */
  extraKey?: ReactNode;
  /** Id of the visible heading the pad belongs to. */
  labelledBy: string;
}

const KEYS = ['1', '2', '3', '4', '5', '6', '7', '8', '9'] as const;

/**
 * Passcode dots + a 3×4 numeric keypad (72px keys) for the app lock. Also
 * takes the hardware keyboard (digits, Backspace). The digits are never
 * rendered, only filled dots; the live region announces the count.
 */
export function PasscodePad({ length, value, onChange, onComplete, error, disabled, extraKey, labelledBy }: PasscodePadProps) {
  const t = useTranslations('common.appLock');
  const loc = useLocale() as Locale;
  const latest = useRef({ value, disabled, length, onChange, onComplete });
  latest.current = { value, disabled, length, onChange, onComplete };

  const press = (digit: string) => {
    const s = latest.current;
    if (s.disabled || s.value.length >= s.length) return;
    const next = s.value + digit;
    s.onChange(next);
    if (next.length === s.length) s.onComplete(next);
  };
  const erase = () => {
    const s = latest.current;
    if (s.disabled || !s.value) return;
    s.onChange(s.value.slice(0, -1));
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      if (/^[0-9]$/.test(e.key)) press(e.key);
      else if (/^[۰-۹]$/.test(e.key)) press(String('۰۱۲۳۴۵۶۷۸۹'.indexOf(e.key)));
      else if (e.key === 'Backspace') erase();
      else return;
      e.preventDefault();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
    // press/erase read the latest props through the ref.
     
  }, []);

  return (
    <div className="lk-pad" role="group" aria-labelledby={labelledBy}>
      <div className="lk-dots" aria-hidden>
        {Array.from({ length }, (_, i) => (
          <span key={i} className={clsx('lk-dot', i < value.length && 'is-on', error ? 'is-error' : null)} />
        ))}
      </div>
      <p className="lk-sr" aria-live="polite">
        {t('entered', { count: formatNumber(value.length, loc), total: formatNumber(length, loc) })}
      </p>
      <p className={clsx('lk-error', !error && 'is-empty')} role={error ? 'alert' : undefined}>
        {error ?? ' '}
      </p>
      <div className="lk-keys">
        {KEYS.map((k) => (
          <button key={k} type="button" className="lk-key" disabled={disabled} onClick={() => press(k)}>
            {formatNumber(k, loc)}
          </button>
        ))}
        <span className="lk-key-slot">{extraKey}</span>
        <button type="button" className="lk-key" disabled={disabled} onClick={() => press('0')}>
          {formatNumber('0', loc)}
        </button>
        <button
          type="button"
          className="lk-key is-ghost"
          disabled={disabled || !value}
          onClick={erase}
          aria-label={t('erase')}
        >
          <Icon name="arrowR" size={22} className="lk-erase" />
        </button>
      </div>
    </div>
  );
}
