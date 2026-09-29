'use client';

import { useLocale } from 'next-intl';
import {
  useId,
  type InputHTMLAttributes,
  type ReactNode,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
} from 'react';

import { cn, toLocaleDigits, toNumberText } from '@/shared/lib';

/**
 * Label + control + hint + error. Controls are plain controlled elements
 * (no form library — validation is zod on submit + the API's 422 bag).
 */
export function Field({
  label,
  hint,
  error,
  required,
  htmlFor,
  className,
  children,
}: {
  label?: ReactNode;
  hint?: ReactNode;
  error?: string;
  required?: boolean;
  htmlFor?: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn('field', className)}>
      {label ? (
        <label className="field-label" htmlFor={htmlFor}>
          {label}
          {required ? (
            <span className="field-required" aria-hidden="true">
              *
            </span>
          ) : null}
        </label>
      ) : null}
      {children}
      {error ? (
        <span className="field-error" role="alert" id={htmlFor ? `${htmlFor}-error` : undefined}>
          {error}
        </span>
      ) : hint ? (
        <span className="field-hint">{hint}</span>
      ) : null}
    </div>
  );
}

type Common = { label?: ReactNode; hint?: ReactNode; error?: string };

export function TextInput({
  label,
  hint,
  error,
  required,
  id,
  className,
  ...rest
}: Common & InputHTMLAttributes<HTMLInputElement>) {
  const auto = useId();
  const inputId = id ?? auto;
  const control = {
    id: inputId,
    className: 'input',
    required,
    'aria-invalid': error ? true : undefined,
    'aria-describedby': error ? `${inputId}-error` : undefined,
    ...rest,
  };
  return (
    <Field label={label} hint={hint} error={error} required={required} htmlFor={inputId} className={className}>
      {rest.type === 'number' ? <NumberInput {...control} /> : <input {...control} />}
    </Field>
  );
}

/**
 * `type="number"` always renders ASCII digits, so number fields are text inputs with a numeric
 * keyboard: the value shows in the UI locale's digits (۱۲ in fa) and whatever digits are typed
 * (Persian, Arabic-Indic or ASCII) are stored as ASCII number text — callers keep reading
 * `e.target.value` exactly as before. Bounds (`min` / `max`) are the API's 422 to report; a text
 * input has no native range check.
 */
function NumberInput({
  value,
  defaultValue,
  onChange,
  step,
  ...rest
}: InputHTMLAttributes<HTMLInputElement>) {
  const locale = useLocale();
  // No native range check on a text input: min/max are dropped, the API reports bounds.
  const props = { ...rest };
  delete props.min;
  delete props.max;
  const integer = step === undefined || Number.isInteger(Number(step));
  const show = (v: typeof value) => (v === undefined || v === null ? v : toLocaleDigits(String(v), locale));
  return (
    <input
      {...props}
      type="text"
      inputMode={integer ? 'numeric' : 'decimal'}
      autoComplete="off"
      value={show(value) as string | undefined}
      defaultValue={show(defaultValue) as string | undefined}
      onChange={(e) => {
        const text = toNumberText(e.target.value);
        if (text !== e.target.value) e.target.value = text;
        onChange?.(e);
      }}
    />
  );
}

export function TextArea({
  label,
  hint,
  error,
  required,
  id,
  className,
  ...rest
}: Common & TextareaHTMLAttributes<HTMLTextAreaElement>) {
  const auto = useId();
  const inputId = id ?? auto;
  return (
    <Field label={label} hint={hint} error={error} required={required} htmlFor={inputId} className={className}>
      <textarea
        id={inputId}
        className="input"
        required={required}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${inputId}-error` : undefined}
        {...rest}
      />
    </Field>
  );
}

export function Select({
  label,
  hint,
  error,
  required,
  id,
  className,
  options,
  ...rest
}: Common &
  SelectHTMLAttributes<HTMLSelectElement> & { options: ReadonlyArray<{ value: string; label: string }> }) {
  const auto = useId();
  const inputId = id ?? auto;
  return (
    <Field label={label} hint={hint} error={error} required={required} htmlFor={inputId} className={className}>
      <select
        id={inputId}
        className="input"
        required={required}
        aria-invalid={error ? true : undefined}
        {...rest}
      >
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </Field>
  );
}

export function Switch({
  label,
  hint,
  checked,
  onChange,
  disabled,
  id,
}: {
  label: ReactNode;
  hint?: ReactNode;
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  id?: string;
}) {
  const auto = useId();
  const switchId = id ?? auto;
  return (
    <div className="flex items-start gap-3">
      <button
        id={switchId}
        type="button"
        role="switch"
        aria-checked={checked}
        className="switch mt-0.5"
        disabled={disabled}
        onClick={() => onChange(!checked)}
      />
      <label htmlFor={switchId} className="flex cursor-pointer flex-col">
        <span className="field-label">{label}</span>
        {hint ? <span className="field-hint">{hint}</span> : null}
      </label>
    </div>
  );
}
