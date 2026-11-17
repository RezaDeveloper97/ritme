'use client';

import { useRef, type KeyboardEvent, type ReactNode } from 'react';
import { clsx } from 'clsx';

import type { PilotAttrs } from '@/shared/pilot';
import { useDirection } from '@/shared/i18n';

import { Icon, type IconName } from '../Icon';

export interface SegmentedTab<V extends string> {
  value: V;
  label: ReactNode;
  icon?: IconName;
  /** Shown but not selectable (e.g. a «به‌زودی» feature); skipped by the arrow keys. */
  disabled?: boolean;
}

interface SegmentedTabsProps<V extends string> {
  tabs: readonly SegmentedTab<V>[];
  value: V;
  onChange: (value: V) => void;
  /** Accessible name of the tab list. */
  label: string;
  /** id of the panel each tab controls, when the caller renders `role=tabpanel`. */
  panelId?: (value: V) => string;
  /** `surface` track (log sheet) instead of the default `--surface-2`. */
  track?: 'default' | 'surface';
  className?: string;
  /** Autopilot name per tab: `(value) => pilot(`…:${value}`)` (frontend/CLAUDE.md §11.6). */
  pilot?: (value: V) => PilotAttrs;
}

/**
 * `role=tablist` segmented control: track `--surface-2`, radius 18, 40px tabs,
 * selected tab solid `--brand-fill`. Roving tabindex; arrow keys follow the
 * reading direction (next = ← in RTL), Home/End jump to the ends.
 */
export function SegmentedTabs<V extends string>({
  tabs,
  value,
  onChange,
  label,
  panelId,
  track = 'default',
  className,
  pilot,
}: SegmentedTabsProps<V>) {
  const rtl = useDirection() === 'rtl';
  const refs = useRef<Array<HTMLButtonElement | null>>([]);

  const move = (index: number, step: 1 | -1) => {
    let next = (index + tabs.length) % tabs.length;
    for (let i = 0; i < tabs.length && tabs[next].disabled; i++) next = (next + step + tabs.length) % tabs.length;
    if (tabs[next].disabled) return;
    onChange(tabs[next].value);
    refs.current[next]?.focus();
  };

  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    const forward = rtl ? 'ArrowLeft' : 'ArrowRight';
    const backward = rtl ? 'ArrowRight' : 'ArrowLeft';
    if (event.key === forward) move(index + 1, 1);
    else if (event.key === backward) move(index - 1, -1);
    else if (event.key === 'Home') move(0, 1);
    else if (event.key === 'End') move(tabs.length - 1, -1);
    else return;
    event.preventDefault();
  };

  return (
    <div role="tablist" aria-label={label} className={clsx('nb-seg', track === 'surface' && 'is-surface', className)}>
      {tabs.map((tab, index) => {
        const selected = tab.value === value;
        return (
          <button
            key={tab.value}
            ref={(el) => {
              refs.current[index] = el;
            }}
            type="button"
            role="tab"
            aria-selected={selected}
            aria-controls={panelId?.(tab.value)}
            aria-disabled={tab.disabled || undefined}
            disabled={tab.disabled}
            tabIndex={selected ? 0 : -1}
            className="nb-seg-tab"
            onClick={() => (tab.disabled ? undefined : onChange(tab.value))}
            onKeyDown={(event) => onKeyDown(event, index)}
            {...pilot?.(tab.value)}
          >
            {tab.icon ? <Icon name={tab.icon} size={16} /> : null}
            {tab.label}
          </button>
        );
      })}
    </div>
  );
}
