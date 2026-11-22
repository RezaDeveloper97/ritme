'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type ReactNode, useEffect, useRef, useState } from 'react';

import type { Medication, MedicationDuration } from '@/entities/care-reminder';
import {
  isCompanionForbidden,
  parseForUserId,
  RecordedFor,
  RecordForRow,
  RecordForSheet,
  useRecordFor,
} from '@/entities/companion';
import { useUserMode } from '@/entities/message';
import { canAsk, NotificationExplainerSheet } from '@/features/push-notifications';
import { useCurrentUser } from '@/entities/user';
import {
  useCreateMedication,
  useDeleteMedication,
  useMedicationFor,
  useUpdateMedication,
} from '@/features/manage-medication';
import { ApiError, getApiErrorStatus, getApiLimitMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { pilot, pilotOption } from '@/shared/pilot';
import {
  formatDayMonth,
  formatLongDate,
  formatNumber,
  fromApiDate,
  toApiDate,
  today,
} from '@/shared/lib/date';
import {
  Icon,
  PrimaryButton,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  Switch as NbSwitch,
} from '@/shared/ui';

import {
  addDose,
  canSubmit,
  defaultDuration,
  type DoseBlock,
  type DoseFieldErrors,
  durationOptions,
  emptyForm,
  type FieldErrors,
  type FormField,
  fromMedication,
  mapServerErrors,
  MAX_DOSES,
  type MedicationFormState,
  patchDose,
  removeDose,
  returnHref,
  slotLabel,
  toggleMeal,
  validateForm,
} from '../model/form';
import { DateSheet, DeleteSheet, DosageSheet, DurationSheet, FrequencySheet, TimeSheet } from './Pickers';
import '@/shared/ui/styles/rmd.css';
import '@/shared/ui/styles/cfm.css';

type OpenSheet =
  | { kind: 'time'; key: string }
  | { kind: 'dosage'; key: string }
  | { kind: 'frequency' }
  | { kind: 'start' }
  | { kind: 'duration' }
  | { kind: 'delete' }
  | { kind: 'recordFor' }
  | null;

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

function Switch({ on, label, onClick, pilotId }: { on: boolean; label: string; onClick: () => void; pilotId: string }) {
  return (
    <NbSwitch compact className="rmd-hit" checked={on} label={label} onCheckedChange={() => onClick()} pilot={pilot(pilotId)} />
  );
}

/** A schedule / dose tile: small label + chevron on line one, the value (or a muted placeholder) on line two. */
function Tile({
  label,
  value,
  placeholder,
  invalid,
  ltr,
  describedBy,
  onClick,
  pilotId,
}: {
  pilotId: string;
  label: string;
  value: string | null;
  placeholder: string;
  invalid?: boolean;
  ltr?: boolean;
  describedBy?: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className="cfm-tile"
      data-invalid={invalid || undefined}
      aria-describedby={describedBy}
      onClick={onClick}
      {...pilot(pilotId)}
    >
      <span className="cfm-tile-h">
        {label}
        <Icon name="chevronDown" size={16} strokeWidth={2} />
      </span>
      <span className={clsx('cfm-tile-v', !value && 'is-empty')} dir={value && ltr ? 'ltr' : undefined}>
        {value ?? placeholder}
      </span>
    </button>
  );
}

// ── Screen ──────────────────────────────────────────────────────

/**
 * Add / edit medication (`v13_AddMedication`), routes
 * `/reminders/medication/new` and `/reminders/medication/[id]`.
 * Save goes through `manage-medication`; on success (or delete) the user is
 * sent back to where they came from (`?from=home|companion`, else the reminders hub).
 *
 * «ثبت برای چه کسی؟» (B-N4-06, Hamdam_RecordFor): a companion with edit on an
 * owner's meds picks «خودم» or her before saving a new medication (`?for=`
 * preselects her); `/reminders/medication/[id]?for=<ownerId>` edits her record
 * (`for_user_id`), without the owner-only delete.
 *
 * Privacy (§11): names, doses and notes are shown and sent, never logged.
 */
export function MedicationFormPage({ id, from, forParam }: { id?: number; from?: string; forParam?: string }) {
  const forUserId = parseForUserId(forParam);
  const t = useTranslations('care');
  const tr = useTranslations('companions.recordFor');
  const query = useMedicationFor(id ?? null, id === undefined ? null : forUserId);

  if (id === undefined) return <MedicationForm from={from} forUserId={forUserId} />;
  if (query.isPending) {
    return (
      <Shell from={from} edit>
        <SkeletonGroup label={t('loading')} className="rmd-form-skel">
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }
  if (query.isError || !query.data) {
    const notFound = getApiErrorStatus(query.error) === 404;
    const forbidden = forUserId !== null && isCompanionForbidden(query.error);
    return (
      <Shell from={from} edit>
        <p className="rmd-state">
          {forbidden ? tr('forbidden') : notFound ? t('medicationForm.notFound') : t('loadError')}
        </p>
        {!notFound && !forbidden && (
          <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
            {t('retry')}
          </button>
        )}
      </Shell>
    );
  }
  return <MedicationForm key={query.data.id} medication={query.data} from={from} forUserId={forUserId} />;
}

function Shell({ from, edit, children }: { from?: string; edit: boolean; children: ReactNode }) {
  const t = useTranslations('care');
  const router = useRouter();
  return (
    <div className="view rmd-page">
      <div className="scroll rmd-screen">
        <SkyLayer />
        <ScreenHeader
          title={edit ? t('medicationForm.editTitle') : t('medicationForm.title')}
          onBack={() => router.push(returnHref(from))}
          backLabel={t('back')}
        />
        <div className="rmd-body">{children}</div>
      </div>
    </div>
  );
}

function MedicationForm({
  medication,
  from,
  forUserId,
}: {
  medication?: Medication;
  from?: string;
  /** `?for=`: preselected owner (new) or the owner whose record this is (edit). */
  forUserId: number | null;
}) {
  const t = useTranslations('care');
  const tf = useTranslations('care.medicationForm');
  const tc = useTranslations('care');
  const tr = useTranslations('companions.recordFor');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mode = useUserMode();
  const pregnancy = mode.data?.mode === 'pregnancy';
  const edit = medication !== undefined;
  // Editing an owner's record: the target is fixed to her (no picker, no delete).
  const delegatedEdit = edit && forUserId !== null;
  const recordFor = useRecordFor('meds', forUserId, delegatedEdit);
  const selfName = useCurrentUser().data?.name ?? null;
  const [savedFor, setSavedFor] = useState<{ name: string | null } | null>(null);

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
  const [doseErrors, setDoseErrors] = useState<DoseFieldErrors>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [sheet, setSheet] = useState<OpenSheet>(null);
  // «روشن کردن اعلان‌ها»: asked once, in context, right after a save with the notify switch on.
  const [askNotify, setAskNotify] = useState<{ proceed: () => void } | null>(null);
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
  const setDoses = (next: DoseBlock[]) => {
    setState((s) => ({ ...s, doses: next }));
    setDoseErrors({});
    if (errors.doses) setErrors((e) => ({ ...e, doses: undefined }));
  };
  const openSheet = (next: NonNullable<OpenSheet>) => {
    setSheetKey((k) => k + 1);
    setSheet(next);
  };
  const close = () => setSheet(null);
  const back = () => router.push(returnHref(from));

  const onSave = (confirmed = false) => {
    setFormError(null);
    const result = validateForm(state);
    if (!result.ok) {
      const next: FieldErrors = {};
      for (const [field, key] of Object.entries(result.errors.fields)) {
        next[field as FormField] = tf(`errors.${key}`);
      }
      const nextDoses: DoseFieldErrors = {};
      for (const [i, e] of Object.entries(result.errors.doses)) {
        nextDoses[Number(i)] = {
          time: e.time ? tf(`errors.${e.time}`) : undefined,
          amount: e.amount ? tf(`errors.${e.amount}`) : undefined,
        };
      }
      setErrors(next);
      setDoseErrors(nextDoses);
      return;
    }
    setErrors({});
    setDoseErrors({});
    // A new medication with someone to record for: ask «برای چه کسی؟» first;
    // the sheet's CTA comes back here confirmed.
    if (!edit && recordFor.showPicker && !confirmed) {
      openSheet({ kind: 'recordFor' });
      return;
    }
    const target = delegatedEdit ? forUserId : edit ? null : recordFor.target;
    const targetName = recordFor.chosen?.name ?? null;
    const afterSave = () => {
      if (!target) return back();
      recordFor.onSaved();
      setSavedFor({ name: targetName });
    };
    const onSaved = () => {
      // Only for her own record, with reminders on, while the permission is still askable.
      if (target || !state.notify) return afterSave();
      void canAsk().then((ask) => (ask ? setAskNotify({ proceed: afterSave }) : afterSave()));
    };
    const onError = (error: unknown) => {
      // The owner revoked the grant meanwhile: say so and refresh the links.
      if (target && isCompanionForbidden(error)) {
        recordFor.onRevoked();
        setFormError(tr('revoked'));
        close();
        return;
      }
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
        setDoseErrors(mapped.doses);
        setFormError(mapped.unknown ?? (Object.keys(mapped.fields).length || Object.keys(mapped.doses).length ? null : t('saveError')));
        return;
      }
      setFormError(t('saveError'));
    };
    if (medication) {
      update.mutate({ id: medication.id, patch: result.input, forUserId: target }, { onSuccess: onSaved, onError });
    } else {
      create.mutate({ ...result.input, forUserId: target }, { onSuccess: onSaved, onError });
    }
  };

  const onDelete = () => {
    if (!medication || delegatedEdit) return;
    remove.mutate(medication.id, { onSuccess: back });
  };

  // ── Derived display ───────────────────────────────────────────
  const num = (v: string | number) => formatNumber(v, locale);
  const freq = state.frequency;
  const freqText =
    freq.unit === 'hour'
      ? tc('frequencySummary.hours', { n: num(freq.every) })
      : freq.every === 1
        ? tc('frequencySummary.day')
        : tc('frequencySummary.days', { n: num(freq.every) });
  const hourMode = freq.unit === 'hour';
  const startDate = fromApiDate(state.startsOn);
  const startText =
    state.startsOn === todayApi
      ? tf('todayWithDate', { date: formatDayMonth(startDate, locale) })
      : formatLongDate(startDate, locale);
  const doseText = (d: DoseBlock) =>
    d.amount !== null && d.unit !== null
      ? tc('doseLine', { amount: num(d.amount), unit: tc(`doseUnits.${d.unit}`) })
      : null;
  const durationText =
    state.duration === 'until_date' && state.endsOn
      ? tf('untilDate', { date: formatLongDate(fromApiDate(state.endsOn), locale) })
      : tf(`durations.${state.duration}`);
  const described = (field: FormField) => (errors[field] ? `med-err-${field}` : undefined);

  if (savedFor) {
    return (
      <Shell from={from} edit={edit}>
        <RecordedFor
          section="meds"
          name={savedFor.name}
          onDone={back}
          doneLabel={from === 'companion' ? tr('backToCompanion') : tr('back')}
        />
      </Shell>
    );
  }

  return (
    <Shell from={from} edit={edit}>
      {delegatedEdit ? (
        <RecordForRow section="meds" target={recordFor.chosen} selfName={selfName} />
      ) : recordFor.showPicker ? (
        <RecordForRow
          section="meds"
          target={recordFor.chosen}
          selfName={selfName}
          onChange={() => openSheet({ kind: 'recordFor' })}
        />
      ) : null}
      {delegatedEdit && !recordFor.canEdit && (
        <p role="alert" className="cfm-error is-center">
          {tr('viewOnly')}
        </p>
      )}

      {/* ── 1. Medicine info ── */}
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
              {...pilot('med.name', 'text')}
            />
          </span>
          <FieldError id="med-err-title" message={errors.title} />
        </label>
      </Card>

      {/* ── 2. Schedule ── */}
      <Card>
        <h2 className="cfm-label">{tf('sectionSchedule')}</h2>
        <div className="cfm-tiles">
          <Tile
            label={tf('startsOn')}
            value={startText}
            placeholder={tf('pickDate')}
            invalid={!!errors.startsOn}
            describedBy={described('startsOn')}
            onClick={() => openSheet({ kind: 'start' })}
            pilotId="med.start"
          />
          <Tile
            label={tf('frequency')}
            value={freqText}
            placeholder={tf('frequencyPlaceholder')}
            invalid={!!errors.frequency}
            describedBy={described('frequency')}
            onClick={() => openSheet({ kind: 'frequency' })}
            pilotId="med.frequency"
          />
        </div>
        <FieldError id="med-err-startsOn" message={errors.startsOn} />
        <FieldError id="med-err-frequency" message={errors.frequency} />
      </Card>

      {/* ── 3. Dose blocks ── */}
      <Card>
        <h2 className="cfm-label">{tf('sectionDose')}</h2>
        {state.doses.map((d, i) => {
          const err = doseErrors[i];
          return (
            <div key={d.key} className="cfm-dose" role="group" aria-label={tf('doseBlock', { n: num(i + 1) })}>
              {i > 0 && (
                <div className="cfm-dose-head">
                  <span className="cfm-dose-n">{tf('doseBlock', { n: num(i + 1) })}</span>
                  <button
                    type="button"
                    className="cfm-dose-remove"
                    {...pilot(`med.dose-remove:${i}`)}
                    aria-label={tf('removeDose', { n: num(i + 1) })}
                    onClick={() => setDoses(removeDose(state.doses, d.key))}
                  >
                    <Icon name="x" size={16} strokeWidth={2} />
                  </button>
                </div>
              )}
              <div className="cfm-tiles">
                <Tile
                  label={tf('dose')}
                  value={doseText(d)}
                  placeholder={tf('dosePlaceholder')}
                  invalid={!!err?.amount}
                  describedBy={err?.amount ? `med-err-dose-${i}-amount` : undefined}
                  onClick={() => openSheet({ kind: 'dosage', key: d.key })}
                  pilotId={`med.dose:${i}`}
                />
                <Tile
                  label={hourMode ? tf('firstDose') : tf('time')}
                  value={d.time ? num(slotLabel(d.time)) : null}
                  placeholder={tf('timePlaceholder')}
                  ltr
                  invalid={!!err?.time}
                  describedBy={err?.time ? `med-err-dose-${i}-time` : undefined}
                  onClick={() => openSheet({ kind: 'time', key: d.key })}
                  pilotId={`med.time:${i}`}
                />
              </div>
              <FieldError id={`med-err-dose-${i}-amount`} message={err?.amount} />
              <FieldError id={`med-err-dose-${i}-time`} message={err?.time} />
              <div className="cfm-chips" role="group" aria-label={tf('mealGroup')} {...pilot(`med.meal:${i}`, 'choice')}>
                {(['before', 'after'] as const).map((m) => (
                  <button
                    key={m}
                    type="button"
                    className="cfm-chip"
                    aria-pressed={d.meal === m}
                    {...pilotOption(m)}
                    onClick={() => setDoses(patchDose(state.doses, d.key, { meal: toggleMeal(d.meal, m) }))}
                  >
                    {tc(`meals.${m}`)}
                  </button>
                ))}
              </div>
            </div>
          );
        })}
        <FieldError id="med-err-doses" message={errors.doses} />
        {state.doses.length < MAX_DOSES && (
          <button type="button" className="cfm-link" onClick={() => setDoses(addDose(state.doses))} {...pilot('med.dose-add')}>
            {tf('addDose')}
          </button>
        )}
      </Card>

      {/* ── 4. More ── */}
      <Card tight>
        <h2 className="cfm-label">{tf('sectionMore')}</h2>
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
            {...pilot('med.duration')}
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
          <Switch on={state.notify} label={tf('notify')} onClick={() => set('notify', !state.notify)} pilotId="med.notify" />
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
            {...pilot('med.notes', 'text')}
          />
          <FieldError id="med-err-notes" message={errors.notes} />
        </label>
      </Card>

      {formError && (
        <p role="alert" className="cfm-error is-center">
          {formError}
        </p>
      )}

      {edit && !delegatedEdit && (
        <button type="button" className="cfm-delete" onClick={() => openSheet({ kind: 'delete' })}>
          <Icon name="trash" size={16} />
          {tf('delete')}
        </button>
      )}

      <div className="cfm-sticky">
        <PrimaryButton
          loading={saving}
          disabled={!canSubmit(state) || (delegatedEdit && !recordFor.canEdit)}
          onClick={() => onSave()}
          {...pilot('med.save')}
        >
          {saving ? tf('saving') : edit ? tf('saveChanges') : tf('create')}
        </PrimaryButton>
      </div>

      {sheet?.kind === 'time' && (
        <TimeSheet
          key={sheetKey}
          open
          value={state.doses.find((d) => d.key === sheet.key)?.time ?? '08:00'}
          onClose={close}
          onPick={(slot) => {
            setDoses(patchDose(state.doses, sheet.key, { time: slot }));
            close();
          }}
        />
      )}
      {sheet?.kind === 'dosage' && (
        <DosageSheet
          key={sheetKey}
          open
          amount={state.doses.find((d) => d.key === sheet.key)?.amount ?? null}
          unit={state.doses.find((d) => d.key === sheet.key)?.unit ?? null}
          onClose={close}
          onPick={(amount, unit) => {
            setDoses(patchDose(state.doses, sheet.key, { amount, unit }));
            close();
          }}
        />
      )}
      {sheet?.kind === 'frequency' && (
        <FrequencySheet
          key={sheetKey}
          open
          value={state.frequency}
          onClose={close}
          onPick={(frequency) => {
            set('frequency', frequency);
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
      {sheet?.kind === 'recordFor' && (
        <RecordForSheet
          key={sheetKey}
          open
          onClose={close}
          section="meds"
          targets={recordFor.targets}
          selfName={selfName}
          value={recordFor.target}
          onChange={recordFor.setTarget}
          onConfirm={() => onSave(true)}
          pending={saving}
          error={formError}
        />
      )}
      <NotificationExplainerSheet
        open={askNotify !== null}
        onDone={() => {
          const next = askNotify;
          setAskNotify(null);
          next?.proceed();
        }}
      />
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
