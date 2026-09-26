'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type ReactNode, useState } from 'react';

import {
  MEDICATION_FORMS,
  MEDICATION_UNITS,
  type Medication,
  type MedicationDuration,
  type MedicationUnit,
  useMedication,
} from '@/entities/care-reminder';
import { useUserMode } from '@/entities/message';
import {
  useCreateMedication,
  useDeleteMedication,
  useUpdateMedication,
} from '@/features/manage-medication';
import { ApiError, getApiErrorStatus } from '@/shared/api';
import { type Locale, Link, useDirection, useRouter } from '@/shared/i18n';
import {
  calendarSystem,
  formatLongDate,
  formatNumber,
  fromApiDate,
  toApiDate,
  today,
} from '@/shared/lib/date';
import { Icon, type IconName } from '@/shared/ui';

import {
  AMOUNT_MAX,
  AMOUNT_MIN,
  durationOptions,
  emptyForm,
  type FieldErrors,
  type FormField,
  fromMedication,
  mapServerErrors,
  type MedicationFormState,
  replaceTime,
  returnHref,
  setTimesCount,
  slotPeriod,
  TIMES_OPTIONS,
  toggleWeekday,
  validateForm,
  weekdayOrder,
  weekdaySummary,
} from '../model/form';
import { DateSheet, DeleteSheet, DurationSheet, TimeSheet } from './Pickers';

type OpenSheet =
  | { kind: 'time'; index: number }
  | { kind: 'start' }
  | { kind: 'duration' }
  | { kind: 'delete' }
  | null;

const isKnownUnit = (u: string): u is MedicationUnit =>
  (MEDICATION_UNITS as readonly string[]).includes(u);

// ── Small presentational bits ───────────────────────────────────

function Card({ icon, title, children }: { icon: IconName; title: string; children: ReactNode }) {
  return (
    <section className="card fld-card flex flex-col gap-3.5">
      <div className="fld-card-hd mb-0!">
        <span className="dot fld-card-dot bg-(--surface-2) text-(--brand)">
          <Icon name={icon} size={16} />
        </span>
        <h2 className="fld-card-title">{title}</h2>
      </div>
      {children}
    </section>
  );
}

function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) return null;
  return (
    <p id={id} role="alert" className="mt-1.5 text-start text-[12px] font-bold text-(--danger-deep)">
      {message}
    </p>
  );
}

function Switch({ on, label, onClick }: { on: boolean; label: string; onClick: () => void }) {
  return (
    <button type="button" role="switch" aria-checked={on} aria-label={label} className="rmd-switch" onClick={onClick}>
      <span className="rmd-switch-knob" />
    </button>
  );
}

// ── Screen ──────────────────────────────────────────────────────

/**
 * Add / edit medication (`v13_AddMedication`), routes
 * `/reminders/medication/new` and `/reminders/medication/[id]`.
 * Save goes through `manage-medication`; on success (or delete) the user is
 * sent back to where they came from (`?from=home`, else the reminders hub).
 *
 * Privacy (§11): names, doses and notes are shown and sent, never logged.
 */
export function MedicationFormPage({ id, from }: { id?: number; from?: string }) {
  const t = useTranslations('care');
  const query = useMedication(id ?? null);

  if (id === undefined) return <MedicationForm from={from} />;
  if (query.isPending) {
    return (
      <Shell from={from} edit>
        <p className="rmd-state">{t('loading')}</p>
      </Shell>
    );
  }
  if (query.isError || !query.data) {
    const notFound = getApiErrorStatus(query.error) === 404;
    return (
      <Shell from={from} edit>
        <p className="rmd-state">{notFound ? t('medicationForm.notFound') : t('loadError')}</p>
        {!notFound && (
          <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
            {t('retry')}
          </button>
        )}
      </Shell>
    );
  }
  return <MedicationForm key={query.data.id} medication={query.data} from={from} />;
}

function Shell({ from, edit, children }: { from?: string; edit: boolean; children: ReactNode }) {
  const t = useTranslations('care');
  const dir = useDirection();
  return (
    <div className="view rmd-page">
      <div className="scroll">
        <header className="rmd-hdr">
          <Link href={returnHref(from)} className="rmd-hdr-btn" aria-label={t('back')}>
            <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={20} strokeWidth={1.8} />
          </Link>
          <div className="rmd-hdr-text">
            <h1 className="rmd-hdr-title">
              {edit ? t('medicationForm.editTitle') : t('medicationForm.title')}
            </h1>
            <p className="rmd-hdr-sub">{t('medicationForm.subtitle')}</p>
          </div>
          <span className="rmd-hdr-btn invisible" aria-hidden />
        </header>
        <div className="rmd-body flex flex-col gap-3 pb-8">{children}</div>
      </div>
    </div>
  );
}

function MedicationForm({ medication, from }: { medication?: Medication; from?: string }) {
  const t = useTranslations('care');
  const tf = useTranslations('care.medicationForm');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mode = useUserMode();
  const pregnancy = mode.data?.mode === 'pregnancy';
  const edit = medication !== undefined;

  const todayApi = toApiDate(today());
  const [state, setState] = useState<MedicationFormState>(() =>
    medication ? fromMedication(medication, todayApi) : emptyForm(todayApi),
  );
  const [errors, setErrors] = useState<FieldErrors>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [sheet, setSheet] = useState<OpenSheet>(null);
  const [sheetKey, setSheetKey] = useState(0);

  const create = useCreateMedication();
  const update = useUpdateMedication();
  const remove = useDeleteMedication();
  const saving = create.isPending || update.isPending;

  const set = <K extends keyof MedicationFormState>(key: K, value: MedicationFormState[K]) => {
    setState((s) => ({ ...s, [key]: value }));
    const field = key as FormField;
    if (errors[field]) setErrors((e) => ({ ...e, [field]: undefined }));
  };
  const openSheet = (next: NonNullable<OpenSheet>) => {
    setSheetKey((k) => k + 1);
    setSheet(next);
  };
  const close = () => setSheet(null);
  const back = () => router.push(returnHref(from));

  const onSave = () => {
    setFormError(null);
    const result = validateForm(state);
    if (!result.ok) {
      const next: FieldErrors = {};
      for (const [field, key] of Object.entries(result.errors)) {
        next[field as FormField] = tf(`errors.${key}`);
      }
      setErrors(next);
      return;
    }
    setErrors({});
    const onError = (error: unknown) => {
      if (error instanceof ApiError && error.response?.status === 422) {
        const mapped = mapServerErrors(error.response.data);
        setErrors(mapped.fields);
        setFormError(mapped.unknown ?? (Object.keys(mapped.fields).length ? null : t('saveError')));
        return;
      }
      setFormError(t('saveError'));
    };
    if (medication) {
      update.mutate({ id: medication.id, patch: result.input }, { onSuccess: back, onError });
    } else {
      create.mutate(result.input, { onSuccess: back, onError });
    }
  };

  const onDelete = () => {
    if (!medication) return;
    remove.mutate(medication.id, { onSuccess: back });
  };

  // ── Derived display ───────────────────────────────────────────
  const num = (v: string | number) => formatNumber(v, locale);
  const summary = weekdaySummary(state.weekdays);
  const summaryText =
    summary.kind === 'everyDay'
      ? tf('everyDay')
      : summary.kind === 'none'
        ? tf('noDays')
        : tf('daysPerWeek', { count: summary.count });
  const startDate = fromApiDate(state.startsOn);
  const startText =
    state.startsOn === todayApi
      ? tf('todayWithDate', { date: formatLongDate(startDate, locale) })
      : formatLongDate(startDate, locale);
  const durationText =
    state.duration === 'until_date' && state.endsOn
      ? tf('untilDate', { date: formatLongDate(fromApiDate(state.endsOn), locale) })
      : tf(`durations.${state.duration}`);
  const unitOptions: string[] = isKnownUnit(state.unit)
    ? [...MEDICATION_UNITS]
    : [state.unit, ...MEDICATION_UNITS];
  const described = (field: FormField) => (errors[field] ? `med-err-${field}` : undefined);

  return (
    <Shell from={from} edit={edit}>
      {/* ── Card 1: what ── */}
      <Card icon="pill" title={tf('form')}>
        <label className="fld-label">
          <span className="fld-label-t">{tf('name')}</span>
          <span className="field">
            <Icon name="pill" size={18} />
            <input
              value={state.title}
              maxLength={255}
              placeholder={tf('namePlaceholder')}
              aria-invalid={!!errors.title}
              aria-describedby={described('title')}
              onChange={(e) => set('title', e.target.value)}
            />
          </span>
          <FieldError id="med-err-title" message={errors.title} />
        </label>

        <div className="grid grid-cols-[1.2fr_1fr] gap-2.5">
          <label className="fld-label">
            <span className="fld-label-t">{tf('dose')}</span>
            <span className="field">
              <input
                value={state.dose}
                inputMode="decimal"
                maxLength={50}
                aria-invalid={!!errors.dose}
                aria-describedby={described('dose')}
                onChange={(e) => set('dose', e.target.value)}
              />
            </span>
            <FieldError id="med-err-dose" message={errors.dose} />
          </label>
          <label className="fld-label">
            <span className="fld-label-t">{tf('unit')}</span>
            <span className="field">
              <select
                className="w-full flex-1 border-0 bg-transparent font-[inherit] text-(--ink) outline-0"
                value={state.unit}
                onChange={(e) => set('unit', e.target.value)}
              >
                {unitOptions.map((u) => (
                  <option key={u} value={u}>
                    {isKnownUnit(u) ? tf(`units.${u}`) : u}
                  </option>
                ))}
              </select>
            </span>
            <FieldError id="med-err-unit" message={errors.unit} />
          </label>
        </div>

        <div role="radiogroup" aria-label={tf('form')} className="fld-chips">
          {MEDICATION_FORMS.map((f) => (
            <button
              key={f}
              type="button"
              role="radio"
              aria-checked={state.form === f}
              className={clsx('chip', state.form === f && 'on')}
              onClick={() => set('form', f)}
            >
              {tf(`forms.${f}`)}
            </button>
          ))}
        </div>
      </Card>

      {/* ── Card 2: when ── */}
      <Card icon="clock" title={tf('timesPerDay')}>
        <div className="seg" role="radiogroup" aria-label={tf('timesPerDay')}>
          {TIMES_OPTIONS.map((n) => (
            <button
              key={n}
              type="button"
              role="radio"
              aria-checked={state.times.length === n}
              className={clsx('flex-1', state.times.length === n && 'on')}
              onClick={() => set('times', setTimesCount(state.times, n))}
            >
              {tf('timesOption', { count: num(n) })}
            </button>
          ))}
        </div>

        <div className="flex flex-col" aria-label={tf('slotTimes')} role="group">
          {state.times.map((slot, i) => (
            <div key={i} className="fld-row">
              <span className="fld-row-label text-(--ink)">
                {tf('slotRow', { n: num(i + 1), period: t(`slotPeriod.${slotPeriod(slot)}`) })}
              </span>
              <button
                type="button"
                className="chip on"
                dir="ltr"
                aria-label={`${tf('slotLabel', { n: num(i + 1) })} ${num(slot)}`}
                onClick={() => openSheet({ kind: 'time', index: i })}
              >
                {num(slot)}
              </button>
            </div>
          ))}
          <FieldError id="med-err-times" message={errors.times} />
        </div>

        <div>
          <div className="fld-row">
            <span className="fld-row-label text-(--ink)">{tf('weekdays')}</span>
            <span className="text-[12.5px] font-bold text-(--brand)" aria-live="polite">
              {summaryText}
            </span>
          </div>
          <div className="flex justify-between gap-1" role="group" aria-label={tf('weekdays')}>
            {weekdayOrder(calendarSystem(locale) === 'jalali').map((d) => {
              const on = state.weekdays.includes(d);
              return (
                <button
                  key={d}
                  type="button"
                  aria-pressed={on}
                  className={clsx(
                    'grid size-9 place-items-center rounded-full border-[1.5px] text-[13px] font-bold transition-colors',
                    on
                      ? 'border-(--brand-fill) bg-(--brand-fill) text-(--on-accent)'
                      : 'border-(--field-border) bg-(--surface) text-(--steel)',
                  )}
                  onClick={() => set('weekdays', toggleWeekday(state.weekdays, d))}
                >
                  {tf(`weekdayShort.${d}`)}
                </button>
              );
            })}
          </div>
          <FieldError id="med-err-weekdays" message={errors.weekdays} />
        </div>

        <div className="fld-row">
          <span className="fld-row-label text-(--ink)">{tf('amount')}</span>
          <div className="flex items-center gap-2.5">
            <button
              type="button"
              className="iconbtn grid size-8 place-items-center rounded-full bg-(--surface-2) text-(--brand)"
              aria-label={tf('decrease')}
              disabled={state.amount <= AMOUNT_MIN}
              onClick={() => set('amount', state.amount - 1)}
            >
              <span aria-hidden>−</span>
            </button>
            <span className="min-w-16 text-center text-[13.5px] font-bold text-(--ink)" aria-live="polite">
              {tf('amountValue', { amount: num(state.amount), form: tf(`forms.${state.form}`) })}
            </span>
            <button
              type="button"
              className="iconbtn grid size-8 place-items-center rounded-full bg-(--surface-2) text-(--brand)"
              aria-label={tf('increase')}
              disabled={state.amount >= AMOUNT_MAX}
              onClick={() => set('amount', state.amount + 1)}
            >
              <Icon name="plus" size={14} strokeWidth={2.4} />
            </button>
          </div>
        </div>
        <FieldError id="med-err-amount" message={errors.amount} />
      </Card>

      {/* ── Card 3: how long ── */}
      <Card icon="calendar" title={tf('duration')}>
        <div className="fld-row">
          <span className="fld-row-label text-(--ink)">{tf('startsOn')}</span>
          <button type="button" className="chip" onClick={() => openSheet({ kind: 'start' })}>
            {startText}
          </button>
        </div>
        <FieldError id="med-err-startsOn" message={errors.startsOn} />

        <div className="fld-row">
          <span className="flex flex-col text-start">
            <span className="fld-row-label text-(--ink)">{tf('duration')}</span>
            <span className="sub">{durationText}</span>
          </span>
          <button type="button" className="chip" onClick={() => openSheet({ kind: 'duration' })}>
            {tf('change')}
          </button>
        </div>
        <FieldError id="med-err-endsOn" message={errors.endsOn ?? errors.duration} />

        <div className="fld-row">
          <span className="flex flex-col text-start">
            <span className="fld-row-label text-(--ink)">{tf('notify')}</span>
            <span className="sub">{tf('notifyHint')}</span>
          </span>
          <Switch on={state.notify} label={tf('notify')} onClick={() => set('notify', !state.notify)} />
        </div>

        <label className="fld-label">
          <span className="fld-label-t">{tf('notes')}</span>
          <textarea
            className="field fld-textarea h-auto min-h-20 py-3"
            rows={2}
            maxLength={2000}
            value={state.notes}
            placeholder={tf('notesPlaceholder')}
            aria-invalid={!!errors.notes}
            onChange={(e) => set('notes', e.target.value)}
          />
          <FieldError id="med-err-notes" message={errors.notes} />
        </label>
      </Card>

      {formError && (
        <p role="alert" className="text-center text-[12.5px] font-bold text-(--danger-deep)">
          {formError}
        </p>
      )}

      <button type="button" className="btn btn-primary w-full" disabled={saving} onClick={onSave}>
        {saving ? tf('saving') : tf('save')}
      </button>
      {edit && (
        <button
          type="button"
          className="btn w-full bg-transparent text-(--danger-deep)"
          onClick={() => openSheet({ kind: 'delete' })}
        >
          <Icon name="trash" size={16} />
          {tf('delete')}
        </button>
      )}

      {sheet?.kind === 'time' && (
        <TimeSheet
          key={sheetKey}
          open
          value={state.times[sheet.index] ?? '08:00'}
          onClose={close}
          onPick={(slot) => {
            set('times', replaceTime(state.times, sheet.index, slot));
            close();
          }}
        />
      )}
      {sheet?.kind === 'start' && (
        <DateSheet
          key={sheetKey}
          open
          title={tf('startsOn')}
          value={state.startsOn}
          onClose={close}
          onPick={(date) => {
            set('startsOn', date);
            close();
          }}
        />
      )}
      {sheet?.kind === 'duration' && (
        <DurationSheet
          key={sheetKey}
          open
          options={durationOptions(pregnancy, state.duration)}
          value={state.duration}
          endsOn={state.endsOn}
          onClose={close}
          onPick={(duration: MedicationDuration, endsOn) => {
            setState((s) => ({ ...s, duration, endsOn }));
            setErrors((e) => ({ ...e, duration: undefined, endsOn: undefined }));
            close();
          }}
        />
      )}
      {sheet?.kind === 'delete' && (
        <DeleteSheet
          key={sheetKey}
          open
          pending={remove.isPending}
          error={remove.isError}
          onClose={close}
          onConfirm={onDelete}
        />
      )}
    </Shell>
  );
}
