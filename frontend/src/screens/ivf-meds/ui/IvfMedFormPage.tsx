'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import {
  IVF_MAX_TIMES,
  IVF_ROLES,
  IVF_ROUTES,
  IVF_STOCK_UNITS,
  type IvfMed,
  type IvfMedPreset,
  useAddIvfMed,
  useDeleteIvfMed,
  useIvfMedPresets,
  useIvfMeds,
  useUpdateIvfMed,
} from '@/entities/ivf';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatDecimal, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { toAsciiDigits } from '@/shared/lib/phone';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  ChipGroup,
  EmptyState,
  Icon,
  InfoNote,
  NumberStepper,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  Switch,
} from '@/shared/ui';

import {
  addTime,
  applyPreset,
  DOSE_UNITS,
  DOSES_PER_UNIT_RANGE,
  type DraftField,
  draftFromMed,
  draftProblems,
  draftToInput,
  emptyDraft,
  isTriggerDraft,
  type MedDraft,
  sortTimes,
  STOCK_RANGE,
} from '../model/form';
import { ConfirmSheet, DateSheet, TimeSheet } from './Pickers';

type DateField = 'triggerDate' | 'startsOn' | 'endsOn';
type OpenPicker =
  | { kind: 'date'; field: DateField; title: string }
  | { kind: 'time'; index: number | 'trigger'; title: string }
  | { kind: 'delete' }
  | null;

function Shell({ edit, children }: { edit: boolean; children: React.ReactNode }) {
  const t = useTranslations('ivf.meds.form');
  const router = useRouter();
  return (
    <div className="view ivfm-form-page">
      <SkyLayer />
      <div className="scroll is-form">
        <ScreenHeader
          title={edit ? t('editTitle') : t('addTitle')}
          onBack={() => router.push('/ivf/meds')}
          backLabel={t('back')}
        />
        {children}
      </div>
    </div>
  );
}

/**
 * «افزودن دارو از روی نسخه» / edit — `/ivf/meds/new`, `/ivf/meds/{id}`
 * (CB-IVF-03). Prescription presets (catalog `ivf_med_presets`, classes only)
 * pre-fill the type, route, unit, times and stock unit; the trigger takes an
 * exact date + time instead of daily times; the stock is optional. A form: no
 * bottom nav (IA_Nav).
 */
export function IvfMedFormPage({ medId }: { medId?: number }) {
  const t = useTranslations('ivf.meds.form');
  const mounted = useMounted();
  const query = useIvfMeds();
  const edit = medId !== undefined;

  if (!mounted || query.isPending) {
    return (
      <Shell edit={edit}>
        <SkeletonGroup label={t('loading')} className="ivfm-form">
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  const med = edit ? query.data?.meds.find((m) => m.id === medId) : undefined;
  if (query.isError || !query.data?.cycle || (edit && !med)) {
    return (
      <Shell edit={edit}>
        <div className="ivfm-form">
          <Card>
            <EmptyState
              icon="syringe"
              title={query.isError ? t('saveError') : !query.data?.cycle ? t('noCycle') : t('notFound')}
            />
          </Card>
        </div>
      </Shell>
    );
  }

  return (
    <Shell edit={edit}>
      <MedForm med={med} />
    </Shell>
  );
}

function MedForm({ med }: { med?: IvfMed }) {
  const t = useTranslations('ivf.meds.form');
  const tm = useTranslations('ivf');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const ids = useId();
  const presets = useIvfMedPresets(locale);
  const add = useAddIvfMed();
  const update = useUpdateIvfMed();
  const remove = useDeleteIvfMed();
  const todayIso = toApiDate(today());
  const [draft, setDraft] = useState<MedDraft>(() => (med ? draftFromMed(med, todayIso) : emptyDraft(todayIso)));
  const [picker, setPicker] = useState<OpenPicker>(null);
  const [touched, setTouched] = useState(false);

  const problems = draftProblems(draft);
  const blocked = Object.keys(problems).length > 0;
  const trigger = isTriggerDraft(draft);
  const save = med ? update : add;

  const set = <K extends keyof MedDraft>(key: K, value: MedDraft[K]) => setDraft((d) => ({ ...d, [key]: value }));
  const problemText = (field: DraftField) => (touched && problems[field] ? t(`errors.${field}`) : null);
  const dateValue = (value: string | null) => (value ? formatDayMonth(fromApiDate(value), locale) : t('choose'));
  const presetTitle = (preset: IvfMedPreset) => preset.title ?? preset.code;
  const previousPreset = presets.data?.find((p) => p.code === draft.presetCode);

  const submit = () => {
    setTouched(true);
    if (blocked) return;
    const input = draftToInput(draft);
    const done = { onSuccess: () => router.replace('/ivf/meds') };
    if (med) update.mutate({ id: med.id, input }, done);
    else add.mutate(input, done);
  };

  const unitOptions = draft.unit && !(DOSE_UNITS as readonly string[]).includes(draft.unit) ? [...DOSE_UNITS, draft.unit] : DOSE_UNITS;

  return (
    <div className="ivfm-form">
      {!med && presets.data?.length ? (
        <section className="ivfm-block" aria-labelledby={`${ids}-presets`}>
          <h2 id={`${ids}-presets`} className="ivfm-block-title">
            {t('presets')}
          </h2>
          <p className="ivfm-block-hint">{t('presetsHint')}</p>
          <ChipGroup label={t('presets')} className="ivfm-chips">
            {presets.data.map((preset) => (
              <PillChip
                key={preset.code}
                pressed={draft.presetCode === preset.code}
                onPressedChange={() =>
                  setDraft((d) =>
                    applyPreset(d, preset, presetTitle(preset), previousPreset ? presetTitle(previousPreset) : null),
                  )
                }
              >
                {presetTitle(preset)}
              </PillChip>
            ))}
          </ChipGroup>
        </section>
      ) : null}

      <Card className="ivfm-card">
        <label className="ivfm-group">
          <span className="ivfm-label">{t('name')}</span>
          <span className="ivfm-field">
            <Icon name="syringe" size={18} strokeWidth={1.8} />
            <input
              value={draft.name}
              maxLength={100}
              placeholder={t('namePlaceholder')}
              aria-invalid={!!problemText('name')}
              onChange={(e) => set('name', e.target.value)}
            />
          </span>
          {problemText('name') ? (
            <span className="ivfm-error" role="alert">
              {problemText('name')}
            </span>
          ) : null}
        </label>

        <ChoiceRow
          label={t('role')}
          options={IVF_ROLES}
          value={draft.role}
          labelOf={(role) => t(`roles.${role}`)}
          onChange={(role) => set('role', role)}
        />
        <ChoiceRow
          label={t('route')}
          options={IVF_ROUTES}
          value={draft.route}
          labelOf={(route) => tm(`doses.route.${route}`)}
          onChange={(route) => set('route', route)}
        />

        <div className="ivfm-dose">
          <label className="ivfm-group">
            <span className="ivfm-label">{t('dose')}</span>
            <span className="ivfm-field">
              <input
                // Shown in the locale's digits; stored and sent as ASCII.
                value={formatDecimal(draft.dose, locale)}
                inputMode="decimal"
                maxLength={50}
                dir="ltr"
                aria-invalid={!!problemText('dose')}
                onChange={(e) => set('dose', toAsciiDigits(e.target.value).replace(/٫/g, '.'))}
              />
            </span>
            {problemText('dose') ? (
              <span className="ivfm-error" role="alert">
                {problemText('dose')}
              </span>
            ) : null}
          </label>
          <ChoiceRow
            label={t('unit')}
            options={unitOptions}
            value={draft.unit ?? ''}
            labelOf={(unit) => ((DOSE_UNITS as readonly string[]).includes(unit) ? t(`units.${unit as (typeof DOSE_UNITS)[number]}`) : unit)}
            onChange={(unit) => set('unit', unit)}
          />
        </div>
      </Card>

      <Card className="ivfm-card">
        {trigger ? (
          <div className="ivfm-group">
            <span className="ivfm-label">{t('triggerAt')}</span>
            <div className="ivfm-two">
              <FieldButton
                label={t('triggerDate')}
                value={dateValue(draft.triggerDate)}
                invalid={!!problemText('trigger')}
                onClick={() => setPicker({ kind: 'date', field: 'triggerDate', title: t('triggerAt') })}
              />
              <FieldButton
                label={t('triggerTime')}
                value={formatNumber(draft.triggerTime, locale)}
                ltr
                onClick={() => setPicker({ kind: 'time', index: 'trigger', title: t('triggerAt') })}
              />
            </div>
            {problemText('trigger') ? (
              <span className="ivfm-error" role="alert">
                {problemText('trigger')}
              </span>
            ) : null}
          </div>
        ) : (
          <div className="ivfm-group">
            <span className="ivfm-label" id={`${ids}-times`}>
              {t('times')}
            </span>
            <ul className="ivfm-times" aria-labelledby={`${ids}-times`}>
              {draft.times.map((time, index) => {
                const shown = formatNumber(time, locale);
                return (
                  <li key={time} className="ivfm-time">
                    <button
                      type="button"
                      className="ivfm-time-value"
                      dir="ltr"
                      aria-label={t('editTime', { time: shown })}
                      onClick={() => setPicker({ kind: 'time', index, title: t('times') })}
                    >
                      {shown}
                    </button>
                    {draft.times.length > 1 ? (
                      <button
                        type="button"
                        className="ivfm-time-remove"
                        aria-label={t('removeTime', { time: shown })}
                        onClick={() => set('times', draft.times.filter((_, i) => i !== index))}
                      >
                        <Icon name="x" size={14} />
                      </button>
                    ) : null}
                  </li>
                );
              })}
              {draft.times.length < IVF_MAX_TIMES ? (
                <li>
                  <button type="button" className="ivfm-time-add" onClick={() => set('times', addTime(draft.times))}>
                    <Icon name="plus" size={14} />
                    {t('addTime')}
                  </button>
                </li>
              ) : null}
            </ul>
            {problemText('times') ? (
              <span className="ivfm-error" role="alert">
                {problemText('times')}
              </span>
            ) : null}
          </div>
        )}

        <div className="ivfm-two">
          <FieldButton
            label={t('startsOn')}
            value={dateValue(draft.startsOn)}
            onClick={() => setPicker({ kind: 'date', field: 'startsOn', title: t('startsOn') })}
          />
          <FieldButton
            label={t('endsOn')}
            hint={t('optional')}
            value={dateValue(draft.endsOn)}
            invalid={!!problemText('endsOn')}
            onClick={() => setPicker({ kind: 'date', field: 'endsOn', title: t('endsOn') })}
            onClear={draft.endsOn ? () => set('endsOn', null) : undefined}
            clearLabel={t('clear', { what: t('endsOn') })}
          />
        </div>
        {problemText('endsOn') ? (
          <span className="ivfm-error" role="alert">
            {problemText('endsOn')}
          </span>
        ) : null}
      </Card>

      <Card className="ivfm-card">
        <div className="ivfm-switch-row">
          <span className="ivfm-switch-text">
            <span id={`${ids}-stock`} className="ivfm-label">
              {t('trackStock')}
            </span>
            <span id={`${ids}-stock-hint`} className="ivfm-block-hint">
              {t('trackStockHint')}
            </span>
          </span>
          <Switch
            checked={draft.trackStock}
            onCheckedChange={(next) => set('trackStock', next)}
            labelledBy={`${ids}-stock`}
            describedBy={`${ids}-stock-hint`}
          />
        </div>
        {draft.trackStock ? (
          <>
            <ChoiceRow
              label={t('stockUnit')}
              options={IVF_STOCK_UNITS}
              value={draft.stockUnit}
              labelOf={(unit) => t(`stockUnits.${unit}`)}
              onChange={(unit) => set('stockUnit', unit)}
            />
            <NumberStepper
              label={t('stock')}
              unit={t(`stockUnits.${draft.stockUnit}`)}
              value={draft.stockUnits}
              min={STOCK_RANGE.min}
              max={STOCK_RANGE.max}
              onChange={(v) => set('stockUnits', v)}
              decrementLabel={t('decrease', { what: t('stock') })}
              incrementLabel={t('increase', { what: t('stock') })}
              locale={locale}
            />
            <NumberStepper
              label={t('dosesPerUnit')}
              value={draft.dosesPerUnit}
              min={DOSES_PER_UNIT_RANGE.min}
              max={DOSES_PER_UNIT_RANGE.max}
              onChange={(v) => set('dosesPerUnit', v)}
              decrementLabel={t('decrease', { what: t('dosesPerUnit') })}
              incrementLabel={t('increase', { what: t('dosesPerUnit') })}
              locale={locale}
            />
          </>
        ) : null}
      </Card>

      <Card className="ivfm-card">
        <label className="ivfm-group">
          <span className="ivfm-label">
            {t('notes')} <span className="ivfm-optional">{t('optional')}</span>
          </span>
          <textarea
            className="ivfm-textarea"
            value={draft.notes}
            maxLength={2000}
            rows={2}
            placeholder={t('notesPlaceholder')}
            onChange={(e) => set('notes', e.target.value)}
          />
        </label>
      </Card>

      <InfoNote>{t('disclaimer')}</InfoNote>

      <div className="ivfm-footer">
        {save.isError ? (
          <p className="ivfm-error" role="alert">
            {getApiSaveErrorMessage(save.error, t('saveError'))}
          </p>
        ) : null}
        <PrimaryButton loading={save.isPending} disabled={touched && blocked} onClick={submit}>
          {t('save')}
        </PrimaryButton>
        {med ? (
          <SecondaryButton variant="text" icon="trash" className="ivfm-delete" onClick={() => setPicker({ kind: 'delete' })}>
            {t('delete')}
          </SecondaryButton>
        ) : null}
      </div>

      {picker?.kind === 'time' ? (
        <TimeSheet
          title={picker.title}
          value={picker.index === 'trigger' ? draft.triggerTime : (draft.times[picker.index] ?? '08:00')}
          onClose={() => setPicker(null)}
          onPick={(clock) => {
            const index = picker.index;
            setDraft((d) =>
              index === 'trigger'
                ? { ...d, triggerTime: clock }
                : { ...d, times: sortTimes(d.times.map((v, i) => (i === index ? clock : v))) },
            );
            setPicker(null);
          }}
        />
      ) : null}
      {picker?.kind === 'date' ? (
        <DateSheet
          title={picker.title}
          value={draft[picker.field]}
          onClose={() => setPicker(null)}
          onPick={(value) => {
            const field = picker.field;
            setDraft((d) => ({ ...d, [field]: value }));
            setPicker(null);
          }}
        />
      ) : null}
      {picker?.kind === 'delete' && med ? (
        <ConfirmSheet
          title={t('deleteTitle')}
          body={t('deleteBody')}
          confirm={t('deleteYes')}
          busy={remove.isPending}
          error={
            remove.isError ? (
              <p className="ivfm-error" role="alert">
                {getApiSaveErrorMessage(remove.error, t('deleteError'))}
              </p>
            ) : null
          }
          onClose={() => setPicker(null)}
          onConfirm={() => remove.mutate(med.id, { onSuccess: () => router.replace('/ivf/meds') })}
        />
      ) : null}
    </div>
  );
}

/** A labelled single-choice chip row (radiogroup semantics via aria-pressed chips). */
function ChoiceRow<V extends string>({
  label,
  options,
  value,
  labelOf,
  onChange,
}: {
  label: string;
  options: readonly V[];
  value: V | string;
  labelOf: (value: V) => string;
  onChange: (value: V) => void;
}) {
  return (
    <div className="ivfm-group">
      <span className="ivfm-label">{label}</span>
      <ChipGroup label={label} className="ivfm-chips">
        {options.map((option) => (
          <PillChip key={option} pressed={option === value} onPressedChange={() => onChange(option)}>
            {labelOf(option)}
          </PillChip>
        ))}
      </ChipGroup>
    </div>
  );
}

/** «label … value» field that opens a picker (dates, times). */
function FieldButton({
  label,
  hint,
  value,
  invalid,
  ltr,
  onClick,
  onClear,
  clearLabel,
}: {
  label: string;
  hint?: string;
  value: string;
  invalid?: boolean;
  ltr?: boolean;
  onClick: () => void;
  onClear?: () => void;
  clearLabel?: string;
}) {
  return (
    <div className={clsx('ivfm-pick', invalid && 'is-invalid')}>
      <button type="button" className="ivfm-pick-btn" onClick={onClick}>
        <span className="ivfm-pick-label">
          {label}
          {hint ? <span className="ivfm-optional"> {hint}</span> : null}
        </span>
        <span className="ivfm-pick-value" dir={ltr ? 'ltr' : undefined}>
          {value}
        </span>
      </button>
      {onClear ? (
        <button type="button" className="ivfm-pick-clear" aria-label={clearLabel} onClick={onClear}>
          <Icon name="x" size={14} />
        </button>
      ) : null}
    </div>
  );
}
