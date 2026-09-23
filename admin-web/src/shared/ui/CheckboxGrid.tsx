'use client';

import { useTranslations } from 'next-intl';
import { useId, type ReactNode } from 'react';

import { Button } from './Button';

export interface CheckboxOption {
  value: string;
  label: string;
  /** Rendered after the label (e.g. a "legacy" badge). */
  extra?: ReactNode;
}

/** A multi-select as a grid of checkboxes (phase pickers), with select-all / clear. */
export function CheckboxGrid({
  label,
  hint,
  error,
  options,
  value,
  onChange,
  bulk = false,
}: {
  label: string;
  hint?: ReactNode;
  error?: string;
  options: readonly CheckboxOption[];
  value: readonly string[];
  onChange: (next: string[]) => void;
  bulk?: boolean;
}) {
  const t = useTranslations('form');
  const id = useId();
  const set = new Set(value);
  const toggle = (v: string, on: boolean) => {
    const next = new Set(set);
    if (on) next.add(v);
    else next.delete(v);
    // Keep the option order, so what is sent is stable.
    onChange(options.map((o) => o.value).filter((o) => next.has(o)));
  };
  return (
    <fieldset className="field m-0 min-w-0 border-0 p-0" aria-describedby={error ? `${id}-error` : hint ? `${id}-hint` : undefined}>
      <legend className="field-label mb-1.5 p-0">{label}</legend>
      <div className="check-grid">
        {options.map((o) => (
          <label key={o.value} className="check-item" data-checked={set.has(o.value) || undefined}>
            <input type="checkbox" checked={set.has(o.value)} onChange={(e) => toggle(o.value, e.target.checked)} />
            <span className="min-w-0 flex-1">{o.label}</span>
            {o.extra}
          </label>
        ))}
      </div>
      {bulk ? (
        <div className="mt-1 flex gap-2">
          <Button size="sm" variant="ghost" onClick={() => onChange(options.map((o) => o.value))}>
            {t('selectAll')}
          </Button>
          <Button size="sm" variant="ghost" onClick={() => onChange([])} disabled={value.length === 0}>
            {t('clearSelection')}
          </Button>
        </div>
      ) : null}
      {error ? (
        <span className="field-error" role="alert" id={`${id}-error`}>
          {error}
        </span>
      ) : hint ? (
        <span className="field-hint" id={`${id}-hint`}>
          {hint}
        </span>
      ) : null}
    </fieldset>
  );
}
