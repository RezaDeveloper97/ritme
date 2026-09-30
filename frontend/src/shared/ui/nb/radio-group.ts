'use client';

import { useRef, type KeyboardEvent } from 'react';

import { useDirection } from '@/shared/i18n';

/**
 * Pure key → index step of an ARIA radiogroup (APG): ↓ and the reading-forward
 * arrow go to the next option, ↑ and reading-backward to the previous one,
 * Home/End to the ends; wraps around. `null` = not a navigation key.
 */
export function radioStep(key: string, index: number, count: number, rtl: boolean): number | null {
  const forward = rtl ? 'ArrowLeft' : 'ArrowRight';
  const backward = rtl ? 'ArrowRight' : 'ArrowLeft';
  if (key === forward || key === 'ArrowDown') return (index + 1) % count;
  if (key === backward || key === 'ArrowUp') return (index - 1 + count) % count;
  if (key === 'Home') return 0;
  if (key === 'End') return count - 1;
  return null;
}

/**
 * Roving tabindex for a radiogroup whose options are buttons: the checked
 * option (or the first when none is) is the only tab stop, and arrow keys move
 * selection *and* focus together, as native radios do.
 */
export function useRovingRadio<V>(values: readonly V[], value: V | null | undefined, onChange: (value: V) => void) {
  const rtl = useDirection() === 'rtl';
  const refs = useRef<Array<HTMLElement | null>>([]);
  const checkedIndex = values.findIndex((v) => v === value);
  const tabStop = checkedIndex >= 0 ? checkedIndex : 0;

  return {
    tabIndexOf: (index: number) => (index === tabStop ? 0 : -1),
    refOf: (index: number) => (el: HTMLElement | null) => {
      refs.current[index] = el;
    },
    onKeyDown: (event: KeyboardEvent<HTMLElement>, index: number) => {
      const next = radioStep(event.key, index, values.length, rtl);
      if (next === null) return;
      event.preventDefault();
      onChange(values[next]);
      refs.current[next]?.focus();
    },
  };
}
