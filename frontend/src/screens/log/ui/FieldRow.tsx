'use client';

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
    <div style={{ fontSize: 13.5, fontWeight: 700, color: 'var(--ink)', textAlign: 'start' }}>
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
        background: on ? 'var(--brand)' : '#D8DEE5',
        display: 'flex',
        justifyContent: on ? 'flex-end' : 'flex-start',
        transition: 'background .18s',
      }}
    >
      <span
        style={{
          width: 22,
          height: 22,
          borderRadius: '50%',
          background: '#fff',
          boxShadow: '0 1px 3px rgba(0,0,0,.25)',
        }}
      />
    </button>
  );
}

// ── Enum single-select (chips) ─────────────────────────────────
function ChipsField({ field, enums, value, onChange, t }: FieldRowProps & { t: T }) {
  if (field.control.kind !== 'chips') return null;
  const options = enums?.[field.control.enumKey] ?? [];
  const enumKey = field.control.enumKey;
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <FieldLabel>{t(`fields.${field.key}`)}</FieldLabel>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
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
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
      {options.map((opt) => (
        <Chip key={opt} on={selected.includes(opt)} label={enumLabel(t, enumKey, opt)} onClick={() => toggle(opt)} />
      ))}
    </div>
  );
}

// ── Intensity degree (label + inline segmented) ────────────────
function DegreeField({ field, enums, value, onChange, t }: FieldRowProps & { t: T }) {
  if (field.control.kind !== 'degree') return null;
  const options = enums?.[field.control.enumKey] ?? [];
  const enumKey = field.control.enumKey;
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 10,
      }}
    >
      <FieldLabel>{t(`fields.${field.key}`)}</FieldLabel>
      <div className="seg" style={{ flex: '0 0 auto', padding: 3 }}>
        {options.map((opt) => (
          <button
            key={opt}
            type="button"
            className={value === opt ? 'on' : ''}
            style={{ padding: '0 12px', height: 30, fontSize: 12.5 }}
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
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10 }}>
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

  const active = typeof value === 'number';
  const defaultValue = round((min + max) / 2, step);
  const selectedIndex = active
    ? Math.max(0, Math.min(count - 1, Math.round(((value as number) - min) / step)))
    : Math.round((defaultValue - min) / step);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10 }}>
        <FieldLabel>{t(`fields.${field.key}`)}</FieldLabel>
        <Switch
          on={active}
          onClick={() => onChange(field.key, active ? undefined : defaultValue)}
        />
      </div>
      {active && (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 12 }}>
          <WheelPicker
            id={`measure-${field.key}`}
            items={items}
            selectedIndex={selectedIndex}
            width={120}
            onChange={(i) => onChange(field.key, valueAt(i))}
          />
          <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--muted)' }}>
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
      className="field"
      style={{
        height: 'auto',
        minHeight: 96,
        padding: 14,
        alignItems: 'flex-start',
        resize: 'none',
        fontFamily: 'inherit',
        textAlign: 'start',
        lineHeight: 1.8,
      }}
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
