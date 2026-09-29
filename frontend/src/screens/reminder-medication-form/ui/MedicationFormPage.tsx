'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type ReactNode, useEffect, useRef, useState } from 'react';

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
import { ApiError, getApiErrorStatus, getApiLimitMessage } from '@/shared/api';
import { type Locale, Link, useDirection, useRouter } from '@/shared/i18n';
import {
  calendarSystem,
  formatDayMonth,
  formatLongDate,
  formatNumber,
  fromApiDate,
  toApiDate,
  today,
} from '@/shared/lib/date';
import { toAsciiDigits } from '@/shared/lib/phone';
import { Icon } from '@/shared/ui';

import {
  AMOUNT_MAX,
  AMOUNT_MIN,
  defaultDuration,
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
  slotLabel,
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

function Card({ tight = false, children }: { tight?: boolean; children: ReactNode }) {
  return <section className={clsx('cfm-card', tight && 'is-tight')}>{children}</section>;
}

function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) return null;
  return (
    <p id={id} role="alert" className="cfm-error">
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
        <div className="rmd-body">{children}</div>
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
    medication ? fromMedication(medication, todayApi) : emptyForm(todayApi, pregnancy),
  );
  // A new form opened before the mode loaded starts as «بدون تاریخ پایان»; once
  // the mode says pregnant, switch to «تا پایان بارداری» unless the user already
  // picked a duration.
  const durationPicked = useRef(false);
  useEffect(() => {
    if (edit || !pregnancy || durationPicked.current) return;
    setState((s) =>
      s.duration === defaultDuration(false) ? { ...s, duration: defaultDuration(true), endsOn: null } : s,
    );
  }, [edit, pregnancy]);
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
      // A per-user cap (422 limit_reached) shows the server's localized message.
      const limit = getApiLimitMessage(error);
      if (limit) {
        setFormError(limit);
        return;
      }
      // 429 = the per-user write limit (60/min): a localized «wait a moment», not
      // the framework's English body.
      if (getApiErrorStatus(error) === 429) {
        setFormError(t('tooManyRequests'));
        return;
      }
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
      ? tf('todayWithDate', { date: formatDayMonth(startDate, locale) })
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
      <Card>
        <label className="cfm-group">
          <span className="cfm-label">{tf('name')}</span>
          <span className="cfm-field">
            <Icon name="pill" size={18} strokeWidth={1.8} />
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

        <div className="cfm-two is-dose">
          <label className="cfm-group">
            <span className="cfm-label">{tf('dose')}</span>
            <span className="cfm-field">
              <input
                // Shown in the locale's digits; stored and sent as ASCII.
                value={num(state.dose)}
                inputMode="decimal"
                maxLength={50}
                aria-invalid={!!errors.dose}
                aria-describedby={described('dose')}
                onChange={(e) => set('dose', toAsciiDigits(e.target.value).replace(/٫/g, '.'))}
              />
            </span>
            <FieldError id="med-err-dose" message={errors.dose} />
          </label>
          <label className="cfm-group">
            <span className="cfm-label">{tf('unit')}</span>
            <span className="cfm-field">
              <select value={state.unit} onChange={(e) => set('unit', e.target.value)}>
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

        <div className="cfm-group is-loose">
          <span className="cfm-label" id="med-form-label">{tf('form')}</span>
          <div role="radiogroup" aria-labelledby="med-form-label" className="cfm-chips">
            {MEDICATION_FORMS.map((f) => (
              <button
                key={f}
                type="button"
                role="radio"
                aria-checked={state.form === f}
                className="cfm-chip"
                onClick={() => set('form', f)}
              >
                {tf(`forms.${f}`)}
              </button>
            ))}
          </div>
        </div>
      </Card>

      {/* ── Card 2: when ── */}
      <Card>
        <div className="cfm-group is-loose">
          <span className="cfm-label" id="med-times-label">{tf('timesPerDay')}</span>
          <div className="cfm-chips is-row" role="radiogroup" aria-labelledby="med-times-label">
            {TIMES_OPTIONS.map((n) => (
              <button
                key={n}
                type="button"
                role="radio"
                aria-checked={state.times.length === n}
                className="cfm-chip is-count"
                onClick={() => set('times', setTimesCount(state.times, n))}
              >
                {tf('timesOption', { count: num(n) })}
              </button>
            ))}
          </div>
        </div>

        <div className="cfm-group is-loose" role="group" aria-labelledby="med-slots-label">
          <span className="cfm-label" id="med-slots-label">{tf('slotTimes')}</span>
          <div className="cfm-slots">
            {state.times.map((slot, i) => (
              <div key={i} className="cfm-slot">
                <span className="cfm-slot-n">{tf('slotLabel', { n: num(i + 1) })}</span>
                <span className="cfm-slot-p">{t(`slotPeriod.${slotPeriod(slot)}`)}</span>
                <button
                  type="button"
                  className="cfm-slot-btn"
                  dir="ltr"
                  aria-label={`${tf('slotLabel', { n: num(i + 1) })} ${num(slotLabel(slot))}`}
                  onClick={() => openSheet({ kind: 'time', index: i })}
                >
                  {num(slotLabel(slot))}
                </button>
              </div>
            ))}
          </div>
          <FieldError id="med-err-times" message={errors.times} />
        </div>

        <div className="cfm-group is-loose">
          <span className="cfm-label" id="med-days-label">{tf('weekdays')}</span>
          <div className="cfm-days" role="group" aria-labelledby="med-days-label">
            {weekdayOrder(calendarSystem(locale) === 'jalali').map((d) => (
              <button
                key={d}
                type="button"
                aria-pressed={state.weekdays.includes(d)}
                className="cfm-day"
                onClick={() => set('weekdays', toggleWeekday(state.weekdays, d))}
              >
                {tf(`weekdayShort.${d}`)}
              </button>
            ))}
          </div>
          <p className="cfm-caption" aria-live="polite">
            {summaryText}
          </p>
          <FieldError id="med-err-weekdays" message={errors.weekdays} />
        </div>

        <div className="cfm-group is-loose">
          <span className="cfm-label">{tf('amount')}</span>
          <div className="cfm-stepper">
            <span className="cfm-stepper-v" aria-live="polite">
              <span className="cfm-stepper-n">{num(state.amount)}</span>
              <span className="cfm-stepper-u">{tf(`forms.${state.form}`)}</span>
            </span>
            <button
              type="button"
              className="cfm-stepper-btn"
              aria-label={tf('decrease')}
              disabled={state.amount <= AMOUNT_MIN}
              onClick={() => set('amount', state.amount - 1)}
            >
              <span aria-hidden>−</span>
            </button>
            <button
              type="button"
              className="cfm-stepper-btn"
              aria-label={tf('increase')}
              disabled={state.amount >= AMOUNT_MAX}
              onClick={() => set('amount', state.amount + 1)}
            >
              <span aria-hidden>+</span>
            </button>
          </div>
          <FieldError id="med-err-amount" message={errors.amount} />
        </div>
      </Card>

      {/* ── Card 3: how long ── */}
      <Card tight>
        <div className="cfm-set">
          <span className="cfm-set-body">
            <span className="cfm-set-t">{tf('startsOn')}</span>
            <span className="cfm-set-s">{startText}</span>
          </span>
          <button
            type="button"
            className="cfm-round"
            aria-label={`${tf('startsOn')}: ${tf('pickDate')}`}
            onClick={() => openSheet({ kind: 'start' })}
          >
            <Icon name="calendar" size={18} strokeWidth={1.8} />
          </button>
        </div>
        <FieldError id="med-err-startsOn" message={errors.startsOn} />

        <div className="cfm-set">
          <span className="cfm-set-body">
            <span className="cfm-set-t">{tf('duration')}</span>
            <span className="cfm-set-s">{durationText}</span>
          </span>
          <button
            type="button"
            className="cfm-pill"
            aria-label={`${tf('duration')}: ${tf('change')}`}
            onClick={() => openSheet({ kind: 'duration' })}
          >
            {tf('change')}
          </button>
        </div>
        <FieldError id="med-err-endsOn" message={errors.endsOn ?? errors.duration} />

        <div className="cfm-set">
          <span className="cfm-set-body">
            <span className="cfm-set-t">{tf('notify')}</span>
            <span className="cfm-set-s">{tf('notifyHint')}</span>
          </span>
          <Switch on={state.notify} label={tf('notify')} onClick={() => set('notify', !state.notify)} />
        </div>

        <label className="cfm-group">
          <span className="cfm-label">{tf('notes')}</span>
          <textarea
            className="cfm-textarea"
            rows={2}
            maxLength={2000}
            value={state.notes}
            placeholder={tf('notesPlaceholder')}
            aria-invalid={!!errors.notes}
            aria-describedby={described('notes')}
            onChange={(e) => set('notes', e.target.value)}
          />
          <FieldError id="med-err-notes" message={errors.notes} />
        </label>
      </Card>

      {formError && (
        <p role="alert" className="cfm-error is-center">
          {formError}
        </p>
      )}

      <button type="button" className="rmd-cta" disabled={saving} onClick={onSave}>
        {saving ? tf('saving') : tf('save')}
      </button>
      {edit && (
        <button type="button" className="cfm-delete" onClick={() => openSheet({ kind: 'delete' })}>
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
            durationPicked.current = true;
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
