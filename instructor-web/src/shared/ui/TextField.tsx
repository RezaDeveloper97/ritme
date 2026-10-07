import { useId, type InputHTMLAttributes, type ReactNode, type TextareaHTMLAttributes } from 'react';

import { cn } from '@/shared/lib';

interface FieldMeta {
  label: string;
  hint?: string;
  error?: string;
  className?: string;
}

function FieldFrame({
  id,
  label,
  hint,
  error,
  className,
  children,
}: FieldMeta & { id: string; children: ReactNode }) {
  return (
    <div className={cn('field', className)}>
      <label className="field__label" htmlFor={id}>
        {label}
      </label>
      {children}
      {error ? (
        <p id={`${id}-err`} className="field__error">
          {error}
        </p>
      ) : hint ? (
        <p id={`${id}-hint`} className="field__hint">
          {hint}
        </p>
      ) : null}
    </div>
  );
}

const describedBy = (id: string, { error, hint }: FieldMeta) =>
  error ? `${id}-err` : hint ? `${id}-hint` : undefined;

type InputProps = FieldMeta & Omit<InputHTMLAttributes<HTMLInputElement>, 'className'> & { multiline?: false };
type AreaProps = FieldMeta & Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'className'> & { multiline: true };

/** Labelled field (label above, 54px control, hint or error below). */
export function TextField(props: InputProps | AreaProps) {
  const id = useId();
  const meta: FieldMeta = { label: props.label, hint: props.hint, error: props.error, className: props.className };
  const common = {
    id,
    'aria-invalid': props.error ? true : undefined,
    'aria-describedby': describedBy(id, meta),
  };
  if (props.multiline) {
    const { label, hint, error, className, multiline, ...rest } = props;
    void [label, hint, error, className, multiline];
    return (
      <FieldFrame id={id} {...meta}>
        <textarea {...rest} {...common} className={cn('field__control field__control--area', props.error && 'is-invalid')} />
      </FieldFrame>
    );
  }
  const { label, hint, error, className, multiline, ...rest } = props;
  void [label, hint, error, className, multiline];
  return (
    <FieldFrame id={id} {...meta}>
      <input {...rest} {...common} className={cn('field__control', props.error && 'is-invalid')} />
    </FieldFrame>
  );
}
