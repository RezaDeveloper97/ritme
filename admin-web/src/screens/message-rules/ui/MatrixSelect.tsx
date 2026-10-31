import { useId } from 'react';

import { Field } from '@/shared/ui';

export interface MatrixOption {
  value: string;
  label: string;
  /** Why the option is disabled (shown beside the label). */
  reason?: string;
}

/** A select whose invalid combinations stay visible but disabled, with the reason beside them. */
export function MatrixSelect({
  label,
  value,
  options,
  onChange,
  error,
  hint,
}: {
  label: string;
  value: string;
  options: readonly MatrixOption[];
  onChange: (v: string) => void;
  error?: string;
  hint?: string;
}) {
  const id = useId();
  return (
    <Field label={label} htmlFor={id} error={error} hint={hint}>
      <select id={id} className="input" value={value} onChange={(e) => onChange(e.target.value)} aria-invalid={error ? true : undefined}>
        {options.map((o) => (
          <option key={o.value} value={o.value} disabled={!!o.reason && o.value !== value}>
            {o.label}
            {o.reason ? ` — ${o.reason}` : ''}
          </option>
        ))}
      </select>
    </Field>
  );
}
