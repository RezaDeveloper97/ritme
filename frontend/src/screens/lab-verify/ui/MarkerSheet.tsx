'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import { type CatalogMarker, type LabMarker, type MarkerInput, parseLabNumber, trimNumber } from '@/entities/lab';
import { AppSheet } from '@/shared/sheet';
import { PrimaryButton, SecondaryButton } from '@/shared/ui';

interface MarkerSheetProps {
  /** The row being corrected, or null for «افزودن شاخص جاافتاده». */
  marker: LabMarker | null;
  catalog: readonly CatalogMarker[];
  busy: boolean;
  /** Server 422 field → message. */
  fieldErrors: Record<string, string>;
  error: string | null;
  onClose: () => void;
  onSave: (input: MarkerInput) => void;
  onDelete?: () => void;
}

const asText = (n: number | null) => (n === null ? '' : trimNumber(n));

/**
 * Correct one marker (value, unit, the sheet's reference range) or add a
 * missed one, with name suggestions from the catalog. Numbers accept Persian
 * digits and «٫». A value that is not a number («منفی») is kept as text.
 * PUT replaces the row, so the sheet's own range is sent back unchanged.
 */
export function MarkerSheet({ marker, catalog, busy, fieldErrors, error, onClose, onSave, onDelete }: MarkerSheetProps) {
  const t = useTranslations('labs.verify');
  const listId = useId();
  const sheetRange = marker?.reference.source === 'sheet';
  const [name, setName] = useState(marker?.printedName ?? '');
  const [value, setValue] = useState(marker ? (marker.value !== null ? asText(marker.value) : (marker.valueText ?? '')) : '');
  const [unit, setUnit] = useState(marker?.unit ?? '');
  const [low, setLow] = useState(marker && sheetRange ? asText(marker.reference.low) : '');
  const [high, setHigh] = useState(marker && sheetRange ? asText(marker.reference.high) : '');
  const [local, setLocal] = useState<Record<string, string>>({});

  const pickName = (v: string) => {
    setName(v);
    const hit = catalog.find((c) => c.name === v || c.code === v);
    if (hit?.unit && !unit) setUnit(hit.unit);
  };

  const save = () => {
    const errs: Record<string, string> = {};
    if (!name.trim()) errs.name = t('errors.name');
    if (!value.trim()) errs.value = t('errors.value');
    const lo = low.trim() ? parseLabNumber(low) : null;
    const hi = high.trim() ? parseLabNumber(high) : null;
    if (low.trim() && lo === null) errs.ref_low = t('errors.number');
    if (high.trim() && hi === null) errs.ref_high = t('errors.number');
    if (lo !== null && hi !== null && lo > hi) errs.ref_low = t('errors.order');
    setLocal(errs);
    if (Object.keys(errs).length) return;
    const num = parseLabNumber(value);
    onSave({
      name: name.trim(),
      value: num,
      valueText: num === null ? value.trim() : null,
      unit: unit.trim() || null,
      refLow: lo,
      refHigh: hi,
      refText: marker && sheetRange && lo === marker.reference.low && hi === marker.reference.high ? marker.reference.text : null,
    });
  };

  const err = (k: string) => local[k] ?? fieldErrors[k] ?? null;
  const field = (key: string, label: string, val: string, set: (v: string) => void, opts: { decimal?: boolean; list?: string; readOnly?: boolean } = {}) => {
    const id = `${listId}-${key}`;
    const e = err(key);
    return (
      <div className="lab-field">
        <label htmlFor={id} className="lab-field-label">
          {label}
        </label>
        <input
          id={id}
          className={clsx('lab-input', e && 'is-invalid')}
          type="text"
          inputMode={opts.decimal ? 'decimal' : 'text'}
          autoComplete="off"
          list={opts.list}
          readOnly={opts.readOnly}
          value={val}
          aria-invalid={e ? true : undefined}
          aria-describedby={e ? `${id}-err` : undefined}
          onChange={(ev) => set(ev.target.value)}
        />
        {e ? (
          <p id={`${id}-err`} className="lab-error" role="alert">
            {e}
          </p>
        ) : null}
      </div>
    );
  };

  return (
    <AppSheet
      open
      onClose={onClose}
      size="full"
      title={marker ? marker.name : t('addTitle')}
      footer={
        <div className="lab-sheet-btns">
          {marker && onDelete ? (
            <SecondaryButton block={false} icon="trash" onClick={onDelete} disabled={busy} className="lab-delete">
              {t('delete')}
            </SecondaryButton>
          ) : (
            <SecondaryButton block={false} onClick={onClose} disabled={busy}>
              {t('cancel')}
            </SecondaryButton>
          )}
          <PrimaryButton block={false} loading={busy} onClick={save}>
            {t('save')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="lab-sheet-form">
        {field('name', t('fields.name'), name, pickName, { list: marker ? undefined : `${listId}-names` })}
        {!marker ? (
          <datalist id={`${listId}-names`}>
            {catalog.map((c) => (
              <option key={c.code} value={c.name} />
            ))}
          </datalist>
        ) : null}
        <div className="lab-field-pair">
          {field('value', t('fields.value'), value, setValue, { decimal: true })}
          {field('unit', t('fields.unit'), unit, setUnit)}
        </div>
        <div className="lab-field-pair">
          {field('ref_low', t('fields.refLow'), low, setLow, { decimal: true })}
          {field('ref_high', t('fields.refHigh'), high, setHigh, { decimal: true })}
        </div>
        <p className="lab-sheet-hint">{t('rangeHint')}</p>
        {error ? (
          <p className="lab-error is-block" role="alert">
            {error}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}
