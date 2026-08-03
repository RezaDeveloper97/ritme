'use client';

import clsx from 'clsx';

import { useFormatter, useTranslations } from 'next-intl';

import type { FieldDef, HealthLogEnums, HealthLogField } from '@/entities/health-log';
import { WheelPicker } from '@/shared/ui';

type T = ReturnType<typeof useTranslations>;

// Enum *values* come from the API at runtime, so their message keys can't be
// statically verified by next-intl's typed-key checker. This narrow alias is
// the honest escape hatch — the fa/en `log.enums.*` maps mirror the API enums.
type DynT = (key: string) => string;
const enumLabel = (t: T, enumKey: string, value: string) =>
  (t as unknown as DynT)(`enums.${enumKey}.${value}`);

interface FieldRowProps {
  field: FieldDef;
  enums: HealthLogEnums | undefined;
  /** Current draft value for this field (string | string[] | boolean | number). */
  value: unknown;
  onChange: (key: HealthLogField, value: unknown) => void;
}

// ── Small building blocks ──────────────────────────────────────
function FieldLabel({ children }: { children: string }) {
  return (
    <div className="lfr-label">
      {children}
    </div>
  );
}

function Chip({ on, label, onClick }: { on: boolean; label: string; onClick: () => void }) {
  return (
    <button type="button" className={`chip${on ? ' on' : ''}`} onClick={onClick}>
      {label}
    </button>
  );
}

/** iOS-style on/off switch, RTL-safe (the knob tracks the writing direction). */
function Switch({ on, onClick }: { on: boolean; onClick: () => void }) {
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
        transition: 'background .18s',
      }}
    >
      <span className="lfr-knob" />
    </button>
  );
}

// ── Enum single-select (chips) ─────────────────────────────────
function ChipsField({ field, enums, value, onChange, t }: FieldRowProps & { t: T }) {
  if (field.control.kind !== 'chips') return null;
  const options = enums?.[field.control.enumKey] ?? [];
  const enumKey = field.control.enumKey;
  return (
    <div className="lfr-stack">
      <FieldLabel>{t(`fields.${field.key}`)}</FieldLabel>
      <div className="lfr-chips">
        {options.map((opt) => (
          <Chip
            key={opt}
            on={value === opt}
            label={enumLabel(t, enumKey, opt)}
            onClick={() => onChange(field.key, value === opt ? undefined : opt)}
          />
        ))}
      </div>
    </div>
  );
}

// ── Enum multi-select (chips) ──────────────────────────────────
function MultiField({ field, enums, value, onChange, t }: FieldRowProps & { t: T }) {
  if (field.control.kind !== 'multi') return null;
  const options = enums?.[field.control.enumKey] ?? [];
  const enumKey = field.control.enumKey;
  const selected = Array.isArray(value) ? (value as string[]) : [];
  const toggle = (opt: string) => {
    const next = selected.includes(opt)
      ? selected.filter((v) => v !== opt)
      : [...selected, opt];
    onChange(field.key, next.length ? next : undefined);
  };
  return (
    <div className="lfr-stack">
      <FieldLabel>{t(`fields.${field.key}`)}</FieldLabel>
      <div className="lfr-chips">
        {options.map((opt) => (
          <Chip key={opt} on={selected.includes(opt)} label={enumLabel(t, enumKey, opt)} onClick={() => toggle(opt)} />
        ))}
      </div>
    </div>
  );
}

// ── Intensity degree (label + inline segmented) ────────────────
function DegreeField({ field, enums, value, onChange, t }: FieldRowProps & { t: T }) {
  if (field.control.kind !== 'degree') return null;
  const options = enums?.[field.control.enumKey] ?? [];
  const enumKey = field.control.enumKey;
  return (
    <div className="lfr-row">
      <FieldLabel>{t(`fields.${field.key}`)}</FieldLabel>
      <div className="seg lfr-seg">
        {options.map((opt) => (
          <button
            key={opt}
            type="button"
            className={clsx('lfr-seg-btn', value === opt && 'on')}
            onClick={() => onChange(field.key, value === opt ? undefined : opt)}
          >
            {enumLabel(t, enumKey, opt)}
          </button>
        ))}
      </div>
    </div>
  );
}

// ── Boolean toggle row ─────────────────────────────────────────
function BoolField({ field, value, onChange, t }: FieldRowProps & { t: T }) {
  const on = value === true;
  return (
    <div className="lfr-row">
      <FieldLabel>{t(`fields.${field.key}`)}</FieldLabel>
      <Switch on={on} onClick={() => onChange(field.key, on ? undefined : true)} />
    </div>
  );
}

// ── Measurement (weight / BBT) via a wheel picker ──────────────
function round(value: number, step: number) {
  const decimals = step < 1 ? 1 : 0;
  return Number(value.toFixed(decimals));
}

function MeasureField({ field, value, onChange, t }: FieldRowProps & { t: T }) {
  const format = useFormatter();
  if (field.control.kind !== 'measure') return null;
  const { min, max, step, unit } = field.control;

  const count = Math.round((max - min) / step) + 1;
  const valueAt = (i: number) => round(min + i * step, step);
  const items = Array.from({ length: count }, (_, i) =>
    format.number(valueAt(i), { maximumFractionDigits: step < 1 ? 1 : 0 }),
  );

  // `alwaysOn` fields (weight, BBT) skip the enable switch: the wheel is the
  // whole sheet, so it stays live and its value is committed by "ثبت" like any
  // other draft edit — closing without submitting still records nothing.
  const alwaysOn = field.control.alwaysOn === true;
  const active = typeof value === 'number';
  // The wheel opens on the field's own sensible starting point (30 min of
  // exercise), falling back to the middle of the range.
  const defaultValue = field.control.default ?? valueAt(Math.round((count - 1) / 2));
  const selectedIndex = active
    ? Math.max(0, Math.min(count - 1, Math.round(((value as number) - min) / step)))
    : Math.round((defaultValue - min) / step);

  return (
    <div className="lfr-group">
      <div className="lfr-row">
        <FieldLabel>{t(`fields.${field.key}`)}</FieldLabel>
        {!alwaysOn && (
          <Switch
            on={active}
            onClick={() => onChange(field.key, active ? undefined : defaultValue)}
          />
        )}
      </div>
      {(alwaysOn || active) && (
        <div className="lfr-stepper">
          <WheelPicker
            id={`measure-${field.key}`}
            items={items}
            selectedIndex={selectedIndex}
            width={120}
            onChange={(i) => onChange(field.key, valueAt(i))}
          />
          <span className="lfr-stepper-v">
            {t(`units.${unit}`)}
          </span>
        </div>
      )}
    </div>
  );
}

// ── Free-text note ─────────────────────────────────────────────
function NoteField({ field, value, onChange, t }: FieldRowProps & { t: T }) {
  return (
    <textarea
      value={typeof value === 'string' ? value : ''}
      onChange={(e) => onChange(field.key, e.target.value.trim() ? e.target.value : undefined)}
      placeholder={t('notesPlaceholder')}
      rows={4}
      className="field lfr-textarea"
    />
  );
}

/**
 * Renders one draft field with the control its config declares. Presentational:
 * value in → render out, every option label and field name through i18n
 * (CLAUDE.md §6), logical spacing for RTL (§12). It never fetches or logs.
 */
export function FieldRow(props: FieldRowProps) {
  const t = useTranslations('log');
  switch (props.field.control.kind) {
    case 'chips':
      return <ChipsField {...props} t={t} />;
    case 'multi':
      return <MultiField {...props} t={t} />;
    case 'degree':
      return <DegreeField {...props} t={t} />;
    case 'bool':
      return <BoolField {...props} t={t} />;
    case 'measure':
      return <MeasureField {...props} t={t} />;
    case 'note':
      return <NoteField {...props} t={t} />;
    default:
      return null;
  }
}
