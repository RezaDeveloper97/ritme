'use client';

import clsx from 'clsx';

import type { ReactNode } from 'react';

import { Icon, type IconName } from '@/shared/ui';

/**
 * Presentational form primitives shared by the pregnancy screens (tracker,
 * onboarding, daily/weekly logs). They live in a feature slice because screens
 * are siblings and can't import each other — the three log screens all reuse
 * these. Every control is RTL-safe (logical alignment, no hardcoded left/right)
 * and carries no domain logic (§3, §12).
 */

// ── Section card ───────────────────────────────────────────────
export function PgCard({
  title,
  icon,
  accent,
  hint,
  children,
}: {
  title?: string;
  icon?: IconName;
  accent?: string;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <section className="card fld-card">
      {title && (
        <div className={clsx('fld-card-hd', hint && 'has-hint')}>
          {icon && (
            <span
              className="dot fld-card-dot"
              style={{ background: (accent ?? 'var(--brand)') + '1A', color: accent ?? 'var(--brand)' }}
            >
              <Icon name={icon} size={16} />
            </span>
          )}
          <span className="fld-card-title">{title}</span>
        </div>
      )}
      {hint && <p className="sub fld-card-hint">{hint}</p>}
      {children}
    </section>
  );
}

// ── Chip (single toggle / multi-select member) ─────────────────
export function Chip({
  on,
  label,
  onClick,
  tone,
}: {
  on: boolean;
  label: string;
  onClick: () => void;
  tone?: 'default' | 'danger';
}) {
  const danger = tone === 'danger';
  return (
    <button
      type="button"
      className={`chip${on ? ' on' : ''}`}
      onClick={onClick}
      style={
        on && danger
          ? { background: 'var(--danger)', borderColor: 'var(--danger)', color: 'var(--on-accent)' }
          : danger
            ? { borderColor: 'var(--danger-line)', color: 'var(--danger-deep)' }
            : undefined
      }
    >
      {label}
    </button>
  );
}

// ── iOS-style on/off toggle (RTL-safe knob) ────────────────────
export function Toggle({ on, onClick }: { on: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={on}
      onClick={onClick}
      style={{
        width: 46,
        height: 28,
        borderRadius: 99,
        border: 0,
        cursor: 'pointer',
        padding: 3,
        flex: '0 0 auto',
        background: on ? 'var(--brand)' : 'var(--track)',
        display: 'flex',
        justifyContent: on ? 'flex-end' : 'flex-start',
        transition: 'background .15s',
      }}
    >
      <span className="fld-switch-knob" />
    </button>
  );
}

// ── Labelled row with a trailing control ───────────────────────
export function FieldRow({
  label,
  danger,
  children,
}: {
  label: string;
  danger?: boolean;
  children: ReactNode;
}) {
  return (
    <div className="fld-row">
      <span className="fld-row-label" style={{ color: danger ? 'var(--danger-deep)' : 'var(--ink)' }}>
        {label}
      </span>
      {children}
    </div>
  );
}

// ── Segmented single-select (severity, mood, movement status) ──
export function Segmented({
  options,
  value,
  onChange,
}: {
  options: { value: string; label: string }[];
  value: string | undefined;
  onChange: (value: string | undefined) => void;
}) {
  return (
    <div className="fld-chips">
      {options.map((opt) => (
        <Chip
          key={opt.value}
          on={value === opt.value}
          label={opt.label}
          onClick={() => onChange(value === opt.value ? undefined : opt.value)}
        />
      ))}
    </div>
  );
}

// ── Numeric field with a unit suffix ───────────────────────────
export function NumberField({
  label,
  value,
  onChange,
  placeholder,
  min,
  max,
  step,
}: {
  label: string;
  value: number | undefined;
  onChange: (value: number | undefined) => void;
  placeholder?: string;
  min?: number;
  max?: number;
  step?: number;
}) {
  return (
    <label className="fld-label">
      <span className="fld-label-t">
        {label}
      </span>
      <input
        className="field fld-input"
        type="number"
        inputMode="decimal"
        value={value ?? ''}
        placeholder={placeholder}
        min={min}
        max={max}
        step={step}
        onChange={(e) => {
          const v = e.target.value;
          onChange(v === '' ? undefined : Number(v));
        }}
      />
    </label>
  );
}

// ── Free-text notes ────────────────────────────────────────────
export function NotesField({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string;
  value: string | undefined;
  onChange: (value: string | undefined) => void;
  placeholder?: string;
}) {
  return (
    <label className="fld-label">
      <span className="fld-label-t">
        {label}
      </span>
      <textarea
        className="field fld-textarea"
        rows={3}
        value={value ?? ''}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value === '' ? undefined : e.target.value)}
      />
    </label>
  );
}
