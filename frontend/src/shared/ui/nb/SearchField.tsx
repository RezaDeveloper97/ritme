'use client';

import { useRef, type InputHTMLAttributes } from 'react';
import { clsx } from 'clsx';

import { Icon } from '../Icon';

interface SearchFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type' | 'onSubmit'> {
  value: string;
  onValueChange: (value: string) => void;
  /** Accessible name («جستجو»); the placeholder is not a label. */
  label: string;
  /** Name of the clear button («پاک کردن»); the button shows only with text. */
  clearLabel: string;
  /** Enter / the keyboard's search key. */
  onSubmit?: (value: string) => void;
}

/**
 * 48px pill search input on `--surface` with a leading magnifier and a 44px
 * clear button (Nav_Search, Dir_List, Shop_List, Ins_Centers). Wrapped in
 * `role=search`; the input is `type=search`.
 */
export function SearchField({
  value,
  onValueChange,
  label,
  clearLabel,
  onSubmit,
  className,
  onKeyDown,
  ...rest
}: SearchFieldProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  return (
    <div role="search" className={clsx('nb-search', className)}>
      <Icon name="search" size={18} className="nb-search-icon" />
      <input
        ref={inputRef}
        type="search"
        aria-label={label}
        enterKeyHint="search"
        className="nb-search-input"
        value={value}
        onChange={(event) => onValueChange(event.target.value)}
        onKeyDown={(event) => {
          onKeyDown?.(event);
          if (event.key === 'Enter' && !event.defaultPrevented) onSubmit?.(value);
        }}
        {...rest}
      />
      {value ? (
        <button
          type="button"
          className="nb-search-clear"
          aria-label={clearLabel}
          onClick={() => {
            onValueChange('');
            inputRef.current?.focus();
          }}
        >
          <Icon name="x" size={16} strokeWidth={2.4} />
        </button>
      ) : null}
    </div>
  );
}
