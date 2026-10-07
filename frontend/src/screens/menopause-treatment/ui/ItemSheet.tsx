'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import {
  TREATMENT_GOAL_UNITS,
  TREATMENT_LIMITS,
  TREATMENT_SCHEDULES,
  type TreatmentGoalUnit,
  type TreatmentItem,
  type TreatmentItemInput,
  type TreatmentKind,
  treatmentItemInput,
  useDeleteTreatmentItem,
  useSaveTreatmentItem,
  useTreatmentIntake,
} from '@/entities/menopause';
import { ApiError, type ApiEnvelope, getApiErrorStatus, getApiSaveErrorMessage } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { type DateParts, formatLongDate, fromApiDate, partsToDate, toApiDate, toParts } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  CalendarPicker,
  ChipGroup,
  Icon,
  InfoNote,
  NumberStepper,
  PillChip,
  PrimaryButton,
  SecondaryButton,
  Switch,
} from '@/shared/ui';

type FieldErrors = Record<string, string>;

/** The API's per-field 422 messages (already in her language). */
function fieldErrorsOf(error: unknown): FieldErrors {
  if (!(error instanceof ApiError) || getApiErrorStatus(error) !== 422) return {};
  const body = error.response?.data as ApiEnvelope<unknown> | undefined;
  const out: FieldErrors = {};
  for (const [key, messages] of Object.entries(body?.errors ?? {})) {
    const first = Array.isArray(messages) ? messages.find((m) => typeof m === 'string') : undefined;
    if (first && key !== 'limit') out[key] = first;
  }
  return out;
}

function emptyInput(kind: TreatmentKind, preset?: Partial<TreatmentItemInput>): TreatmentItemInput {
  return {
    kind,
    name: '',
    dose: null,
    schedule: kind === 'lifestyle' ? null : 'morning',
    form: null,
    startedOn: null,
    reviewOn: null,
    stoppedOn: null,
    weeklyGoal: kind === 'lifestyle' ? 2 : null,
    goalUnit: kind === 'lifestyle' ? 'sessions' : null,
    remind: true,
    ...preset,
  };
}

const goalBounds = (unit: TreatmentGoalUnit | null) =>
  unit === 'minutes'
    ? { min: 10, max: TREATMENT_LIMITS.goalMinutesMax, step: 10 }
    : { min: 1, max: TREATMENT_LIMITS.goalSessionsMax, step: 1 };

/**
 * Add / edit one treatment item (full sheet). hrt / supplement: name, dose as
 * written on her prescription, intake time (a care medication reminder),
 * start and — for HRT — review dates. Lifestyle: a weekly goal in sessions or
 * minutes. Editing adds «قطع مصرف» (stopped today; the reminder goes off),
 * «ادامه دوباره» for a stopped one, and delete.
 */
export function ItemSheet({
  kind,
  item,
  preset,
  today,
  onClose,
}: {
  kind: TreatmentKind;
  item: TreatmentItem | null;
  preset?: Partial<TreatmentItemInput>;
  today: string | null;
  onClose: () => void;
}) {
  const t = useTranslations('menopause.treatment.form');
  const tRoot = useTranslations('menopause.treatment');
  const locale = useLocale() as Locale;
  const ids = useId();
  const save = useSaveTreatmentItem();
  const remove = useDeleteTreatmentItem();
  const [input, setInput] = useState<TreatmentItemInput>(() => (item ? treatmentItemInput(item) : emptyInput(kind, preset)));
  const [local, setLocal] = useState<FieldErrors>({});
  const [picking, setPicking] = useState<'startedOn' | 'reviewOn' | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const busy = save.isPending || remove.isPending;
  const serverErrors = fieldErrorsOf(save.error);
  const err = (key: string) => local[key] ?? serverErrors[key] ?? null;
  const set = (patch: Partial<TreatmentItemInput>) => setInput((prev) => ({ ...prev, ...patch }));
  const isMedicine = kind !== 'lifestyle';
  const bounds = goalBounds(input.goalUnit);

  const submit = (override?: Partial<TreatmentItemInput>) => {
    const next = { ...input, ...override };
    const errors: FieldErrors = {};
    if (!next.name.trim()) errors.name = t('nameRequired');
    if (isMedicine && !next.schedule) errors.schedule = t('scheduleRequired');
    setLocal(errors);
    if (Object.keys(errors).length) return;
    save.mutate({ id: item?.id ?? null, input: next }, { onSuccess: onClose });
  };

  const generalError =
    save.isError && !Object.keys(serverErrors).length ? getApiSaveErrorMessage(save.error, t('saveError')) : null;
  const stopped = Boolean(item?.stoppedOn);

  const text = (key: 'name' | 'dose', value: string, placeholder: string, max: number) => {
    const id = `${ids}-${key}`;
    const e = err(key);
    return (
      <div className="mtr-field">
        <label htmlFor={id} className="mtr-field-label">
          {t(key)}
        </label>
        <input
          id={id}
          className={clsx('mtr-input', e && 'is-invalid')}
          type="text"
          autoComplete="off"
          maxLength={max}
          placeholder={placeholder}
          value={value}
          aria-invalid={e ? true : undefined}
          aria-describedby={e ? `${id}-err` : undefined}
          onChange={(ev) => set(key === 'name' ? { name: ev.target.value } : { dose: ev.target.value })}
        />
        {e ? (
          <p id={`${id}-err`} className="mtr-error" role="alert">
            {e}
          </p>
        ) : null}
      </div>
    );
  };

  const dateField = (key: 'startedOn' | 'reviewOn', label: string) => {
    const value = input[key];
    const e = err(key === 'startedOn' ? 'started_on' : 'review_on');
    return (
      <div className={clsx('mtr-pick', e && 'is-invalid')}>
        <button type="button" className="mtr-pick-btn" onClick={() => setPicking(key)}>
          <span className="mtr-pick-label">{label}</span>
          <span className={clsx('mtr-pick-value', !value && 'is-empty')}>
            {value ? formatLongDate(fromApiDate(value), locale) : t('notSet')}
          </span>
        </button>
        {value ? (
          <button type="button" className="mtr-pick-clear" aria-label={t('clear', { field: label })} onClick={() => set({ [key]: null })}>
            <Icon name="x" size={16} />
          </button>
        ) : null}
        {e ? (
          <p className="mtr-error" role="alert">
            {e}
          </p>
        ) : null}
      </div>
    );
  };

  return (
    <>
      <AppSheet
        open
        onClose={busy ? () => undefined : onClose}
        size="full"
        title={item ? t('editTitle') : t(`addTitle.${kind}`)}
        footer={
          <div className="mtr-sheet-btns">
            <SecondaryButton block={false} onClick={onClose} disabled={busy}>
              {t('cancel')}
            </SecondaryButton>
            <PrimaryButton block={false} loading={save.isPending} disabled={remove.isPending} onClick={() => submit()}>
              {t('save')}
            </PrimaryButton>
          </div>
        }
      >
        <div className="mtr-form">
          {text('name', input.name, t(`namePlaceholder.${kind}`), TREATMENT_LIMITS.nameMax)}
          {isMedicine ? (
            <>
              {text('dose', input.dose ?? '', t('dosePlaceholder'), TREATMENT_LIMITS.doseMax)}
              <div className="mtr-field">
                <span className="mtr-field-label" id={`${ids}-schedule`}>
                  {t('schedule')}
                </span>
                <ChipGroup label={t('schedule')}>
                  {TREATMENT_SCHEDULES.map((s) => (
                    <PillChip key={s} mode="single" pressed={input.schedule === s} onPressedChange={() => set({ schedule: s })}>
                      {tRoot(`schedule.${s}`)}
                    </PillChip>
                  ))}
                </ChipGroup>
                {err('schedule') ? (
                  <p className="mtr-error" role="alert">
                    {err('schedule')}
                  </p>
                ) : null}
              </div>
              <div className="mtr-switch-row">
                <span className="mtr-switch-text">
                  <span id={`${ids}-remind`} className="mtr-field-label">
                    {t('remind')}
                  </span>
                  <span className="mtr-hint">{t('remindHint')}</span>
                </span>
                <Switch checked={input.remind} onCheckedChange={(remind) => set({ remind })} labelledBy={`${ids}-remind`} />
              </div>
            </>
          ) : (
            <>
              <div className="mtr-field">
                <span className="mtr-field-label">{t('unit')}</span>
                <ChipGroup label={t('unit')}>
                  {TREATMENT_GOAL_UNITS.map((u) => (
                    <PillChip
                      key={u}
                      mode="single"
                      pressed={input.goalUnit === u}
                      onPressedChange={() => {
                        const b = goalBounds(u);
                        set({ goalUnit: u, weeklyGoal: Math.min(b.max, Math.max(b.min, u === 'minutes' ? 150 : 2)) });
                      }}
                    >
                      {t(`units.${u}`)}
                    </PillChip>
                  ))}
                </ChipGroup>
              </div>
              <NumberStepper
                boxed
                label={t('goal')}
                value={input.weeklyGoal ?? bounds.min}
                min={bounds.min}
                max={bounds.max}
                step={bounds.step}
                unit={t(`units.${input.goalUnit ?? 'sessions'}`)}
                decrementLabel={t('decrease')}
                incrementLabel={t('increase')}
                locale={locale}
                onChange={(weeklyGoal) => set({ weeklyGoal })}
              />
              {err('weekly_goal') ? (
                <p className="mtr-error" role="alert">
                  {err('weekly_goal')}
                </p>
              ) : null}
            </>
          )}
          {dateField('startedOn', t('startedOn'))}
          {kind === 'hrt' ? dateField('reviewOn', t('reviewOn')) : null}
          {kind === 'hrt' ? <p className="mtr-hint">{t('reviewHint')}</p> : null}

          {generalError ? (
            <p className="mtr-error is-block" role="alert">
              {generalError}
            </p>
          ) : null}
          {err('stopped_on') ? (
            <p className="mtr-error is-block" role="alert">
              {err('stopped_on')}
            </p>
          ) : null}

          {item ? (
            <div className="mtr-manage">
              {isMedicine ? <InfoNote className="mtr-note">{tRoot('doctorOnly')}</InfoNote> : null}
              {stopped ? (
                <SecondaryButton icon="refresh" disabled={busy} onClick={() => submit({ stoppedOn: null })}>
                  {t('resume')}
                </SecondaryButton>
              ) : (
                <SecondaryButton icon="x" disabled={busy || !today} onClick={() => today && submit({ stoppedOn: today })}>
                  {t(isMedicine ? 'stop' : 'stopGoal')}
                </SecondaryButton>
              )}
              <SecondaryButton variant="text" icon="trash" className="mtr-delete" disabled={busy} onClick={() => setConfirmDelete(true)}>
                {t('delete')}
              </SecondaryButton>
            </div>
          ) : null}
        </div>
      </AppSheet>

      {picking ? (
        <DateSheet
          title={picking === 'startedOn' ? t('startedOn') : t('reviewOn')}
          value={input[picking]}
          onClose={() => setPicking(null)}
          onPick={(date) => {
            set({ [picking]: date });
            setPicking(null);
          }}
        />
      ) : null}

      {confirmDelete && item ? (
        <AppSheet
          open
          onClose={remove.isPending ? () => undefined : () => setConfirmDelete(false)}
          size="half"
          title={t('deleteTitle')}
          footer={
            <div className="mtr-sheet-btns">
              <SecondaryButton block={false} disabled={remove.isPending} onClick={() => setConfirmDelete(false)}>
                {t('cancel')}
              </SecondaryButton>
              <PrimaryButton
                block={false}
                className="mtr-danger"
                loading={remove.isPending}
                onClick={() =>
                  remove.mutate(item.id, {
                    onSuccess: () => {
                      setConfirmDelete(false);
                      onClose();
                    },
                  })
                }
              >
                {t('deleteYes')}
              </PrimaryButton>
            </div>
          }
        >
          <p className="mtr-sheet-body">{t('deleteBody', { name: item.name })}</p>
          {remove.isError ? (
            <p className="mtr-error" role="alert">
              {t('saveError')}
            </p>
          ) : null}
        </AppSheet>
      ) : null}
    </>
  );
}

/** A date in the locale's calendar (Jalali in fa) → `Y-m-d`. Mounted only while open. */
function DateSheet({
  title,
  value,
  onClose,
  onPick,
}: {
  title: string;
  value: string | null;
  onClose: () => void;
  onPick: (apiDate: string) => void;
}) {
  const t = useTranslations('menopause.treatment.form');
  const locale = useLocale() as Locale;
  const [parts, setParts] = useState<DateParts | null>(value ? toParts(fromApiDate(value), locale) : null);
  return (
    <AppSheet
      open
      onClose={onClose}
      size="half"
      title={title}
      footer={
        <div className="mtr-sheet-btns">
          <SecondaryButton block={false} onClick={onClose}>
            {t('cancel')}
          </SecondaryButton>
          <PrimaryButton block={false} disabled={!parts} onClick={() => parts && onPick(toApiDate(partsToDate(parts, locale)))}>
            {t('done')}
          </PrimaryButton>
        </div>
      }
    >
      <CalendarPicker value={parts} onSelect={setParts} />
    </AppSheet>
  );
}

/**
 * Today's minutes of a minutes goal (brisk walk, relaxation): a stepper
 * prefilled with what she logged, save = PUT intake, «حذف ثبت امروز» = DELETE.
 */
export function MinutesSheet({ item, today, onClose }: { item: TreatmentItem; today: string; onClose: () => void }) {
  const t = useTranslations('menopause.treatment.minutes');
  const locale = useLocale() as Locale;
  const intake = useTreatmentIntake();
  const logged = item.week.find((d) => d.date === today)?.amount ?? 0;
  const [minutes, setMinutes] = useState(logged || 10);
  return (
    <AppSheet
      open
      onClose={intake.isPending ? () => undefined : onClose}
      size="half"
      title={t('title', { name: item.name })}
      footer={
        <div className="mtr-sheet-btns">
          {logged ? (
            <SecondaryButton
              block={false}
              disabled={intake.isPending}
              onClick={() => intake.mutate({ id: item.id, date: today, taken: false }, { onSuccess: onClose })}
            >
              {t('remove')}
            </SecondaryButton>
          ) : (
            <SecondaryButton block={false} disabled={intake.isPending} onClick={onClose}>
              {t('cancel')}
            </SecondaryButton>
          )}
          <PrimaryButton
            block={false}
            loading={intake.isPending}
            onClick={() => intake.mutate({ id: item.id, date: today, taken: true, amount: minutes }, { onSuccess: onClose })}
          >
            {t('save')}
          </PrimaryButton>
        </div>
      }
    >
      <NumberStepper
        boxed
        label={t('label')}
        value={minutes}
        min={1}
        max={TREATMENT_LIMITS.intakeMinutesMax}
        step={5}
        unit={t('unit')}
        decrementLabel={t('decrease')}
        incrementLabel={t('increase')}
        locale={locale}
        onChange={setMinutes}
      />
      {intake.isError ? (
        <p className="mtr-error" role="alert">
          {getApiSaveErrorMessage(intake.error, t('error'))}
        </p>
      ) : null}
    </AppSheet>
  );
}
