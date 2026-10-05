'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId, useState } from 'react';

import { type ChildMeasurement, childFieldError, parseDecimalInput } from '@/entities/child';
import { getApiErrorStatus } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import {
  type DateParts,
  formatDecimal,
  formatLongDate,
  fromApiDate,
  partsToDate,
  toApiDate,
  today,
  toParts,
} from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, Icon, IconCircle, PrimaryButton, SecondaryButton } from '@/shared/ui';

import { isReadOnlyError, useDeleteMeasurement, useSaveMeasurement } from '../api/queries';
import { checkForm, emptyForm, formFromMeasurement, MEASUREMENT_RANGES, type MeasurementField } from '../model/form';
import type { MeasurementForm } from '../model/types';

const API_FIELD: Record<MeasurementField | 'measuredOn', string> = {
  measuredOn: 'measured_on',
  weightKg: 'weight_kg',
  lengthCm: 'length_cm',
  headCm: 'head_cm',
};

interface Props {
  childId: number;
  birthDate: string;
  /** Null = a new measurement. */
  measurement: ChildMeasurement | null;
  open: boolean;
  onClose: () => void;
}

/**
 * Add / edit a measurement (date + weight / length / head, at least one). The
 * server's 422 messages are shown under their field; dates are picked in the
 * locale's calendar and must lie between the birth and today.
 */
export function MeasurementSheet({ childId, birthDate, measurement, open, onClose }: Props) {
  const t = useTranslations('children');
  const locale = useLocale() as Locale;
  const save = useSaveMeasurement(childId);
  const remove = useDeleteMeasurement(childId);
  const [form, setForm] = useState<MeasurementForm>(() => emptyForm(toApiDate(today())));
  const [dateOpen, setDateOpen] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const [problem, setProblem] = useState<ReturnType<typeof checkForm>>(null);

  // A fresh form each time the sheet opens (new → today, edit → the stored values).
  useEffect(() => {
    if (!open) return;
    setForm(measurement ? formFromMeasurement(measurement) : emptyForm(toApiDate(today())));
    setDateOpen(false);
    setConfirm(false);
    setProblem(null);
    save.reset();
    remove.reset();
    // Only when the sheet opens or its target changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, measurement]);

  const set = (key: keyof MeasurementForm, value: string) => {
    setForm((f) => ({ ...f, [key]: value }));
    setProblem(null);
  };

  const fieldError = (key: MeasurementField | 'measuredOn'): string | undefined => {
    if (problem && problem.kind === 'range' && problem.field === key) {
      const r = MEASUREMENT_RANGES[problem.field];
      return t('growth.form.range', { min: formatDecimal(r.min, locale), max: formatDecimal(r.max, locale) });
    }
    return childFieldError(save.error, API_FIELD[key]);
  };

  const submit = () => {
    const p = checkForm(form);
    if (p) {
      setProblem(p);
      return;
    }
    save.mutate({ measurementId: measurement?.id ?? null, form }, { onSuccess: onClose });
  };

  const onPick = (parts: DateParts) => {
    const iso = toApiDate(partsToDate(parts, locale));
    const now = toApiDate(today());
    set('measuredOn', iso > now ? now : iso < birthDate ? birthDate : iso);
    setDateOpen(false);
  };

  const busy = save.isPending || remove.isPending;
  const saveFailed = save.isError && getApiErrorStatus(save.error) !== 422;
  const dateLabel = formatLongDate(fromApiDate(form.measuredOn), locale);

  return (
    <>
      <AppSheet
        open={open && !confirm}
        onClose={() => (busy ? undefined : onClose())}
        size="half"
        title={measurement ? t('growth.form.titleEdit') : t('growth.form.titleNew')}
        footer={
          <div className="chd-sheet-btns">
            <SecondaryButton onClick={onClose} disabled={busy}>
              {t('growth.form.cancel')}
            </SecondaryButton>
            <PrimaryButton loading={save.isPending} disabled={remove.isPending} onClick={submit}>
              {t('growth.form.save')}
            </PrimaryButton>
          </div>
        }
      >
        <div className="cgr-form">
          <div className="chd-field">
            <span className="chd-label" id="cgr-date-label">
              {t('growth.form.date')}
            </span>
            <button
              type="button"
              className="chd-date-btn"
              aria-expanded={dateOpen}
              aria-labelledby="cgr-date-label cgr-date-value"
              onClick={() => setDateOpen((o) => !o)}
            >
              <span id="cgr-date-value" className="chd-date-value">
                {dateLabel}
              </span>
              <Icon name="calendar" size={20} />
            </button>
            {dateOpen ? (
              <CalendarPicker value={toParts(fromApiDate(form.measuredOn), locale)} onSelect={onPick} />
            ) : null}
            {fieldError('measuredOn') ? (
              <p className="chd-error" role="alert">
                {fieldError('measuredOn')}
              </p>
            ) : null}
          </div>

          <DecimalField
            label={t('growth.form.weight')}
            unit={t('growth.form.kg')}
            value={form.weightKg}
            decimals={MEASUREMENT_RANGES.weightKg.decimals}
            error={fieldError('weightKg')}
            onChange={(v) => set('weightKg', v)}
          />
          <div className="chd-grid2">
            <DecimalField
              label={t('growth.form.length')}
              unit={t('growth.form.cm')}
              value={form.lengthCm}
              decimals={MEASUREMENT_RANGES.lengthCm.decimals}
              error={fieldError('lengthCm')}
              onChange={(v) => set('lengthCm', v)}
            />
            <DecimalField
              label={t('growth.form.head')}
              unit={t('growth.form.cm')}
              value={form.headCm}
              decimals={MEASUREMENT_RANGES.headCm.decimals}
              error={fieldError('headCm')}
              onChange={(v) => set('headCm', v)}
            />
          </div>
          {problem?.field === 'all' ? (
            <p className="chd-error" role="alert">
              {t('growth.form.needOne')}
            </p>
          ) : (
            <p className="chd-hint">{t('growth.form.hint')}</p>
          )}
          {saveFailed ? (
            <p className="chd-error" role="alert">
              {isReadOnlyError(save.error) ? t('growth.form.readOnly') : t('growth.form.saveError')}
            </p>
          ) : null}
          {measurement?.id ? (
            <button type="button" className="chd-delete" disabled={busy} onClick={() => setConfirm(true)}>
              <Icon name="trash" size={18} />
              {t('growth.form.delete')}
            </button>
          ) : null}
        </div>
      </AppSheet>

      <AppSheet
        open={open && confirm}
        onClose={() => (remove.isPending ? undefined : setConfirm(false))}
        size="half"
        title={t('growth.deleteTitle')}
        footer={
          <div className="chd-sheet-btns">
            <SecondaryButton onClick={() => setConfirm(false)} disabled={remove.isPending}>
              {t('growth.form.cancel')}
            </SecondaryButton>
            <SecondaryButton
              variant="text"
              className="chd-danger-text"
              loading={remove.isPending}
              onClick={() => {
                if (measurement?.id) remove.mutate(measurement.id, { onSuccess: onClose });
              }}
            >
              {t('growth.deleteConfirm')}
            </SecondaryButton>
          </div>
        }
      >
        <div className="chd-sheet-body">
          <IconCircle icon="trash" tone="danger" size="lg" />
          <p>{t('growth.deleteBody', { date: measurement ? formatLongDate(fromApiDate(measurement.measuredOn), locale) : '' })}</p>
        </div>
        {remove.isError ? (
          <p className="chd-error" role="alert">
            {t('growth.deleteError')}
          </p>
        ) : null}
      </AppSheet>
    </>
  );
}

function DecimalField({
  label,
  unit,
  value,
  decimals,
  error,
  onChange,
}: {
  label: string;
  unit: string;
  value: string;
  decimals: number;
  error?: string;
  onChange: (v: string) => void;
}) {
  const id = useId();
  const locale = useLocale() as Locale;
  // Shown in the locale's digits; kept canonical (ASCII, «.») in state.
  const shown = value === '' ? '' : formatDecimal(value, locale);
  return (
    <div className="chd-field">
      <label htmlFor={id} className="chd-label">
        {label}
      </label>
      <div className={clsx('chd-input', error && 'is-invalid')}>
        <input
          id={id}
          type="text"
          inputMode="decimal"
          autoComplete="off"
          value={shown}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? `${id}-err` : undefined}
          onChange={(e) => onChange(parseDecimalInput(e.target.value, decimals).text)}
        />
        <span className="chd-unit">{unit}</span>
      </div>
      {error ? (
        <p id={`${id}-err`} className="chd-error" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
}
