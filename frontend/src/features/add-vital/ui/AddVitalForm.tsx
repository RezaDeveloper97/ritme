'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import {
  ARMS,
  BP_SCALE,
  GLUCOSE_CONTEXTS,
  GLUCOSE_METHODS,
  GLUCOSE_UNITS,
  HR_CONTEXTS,
  LIMITS,
  POSITIONS,
  bpScalePosition,
  classifyBp,
  classifyGlucose,
  classifyHr,
  convertGlucose,
  fieldErrors,
  glucoseDecimals,
  glucoseStep,
  toMgDl,
  uiTone,
  unitSymbol,
  useCreateReading,
  useVitalFormat,
  useVitalThresholds,
  type Arm,
  type GlucoseContext,
  type GlucoseMethod,
  type GlucoseUnit,
  type HrContext,
  type Position,
  type ReadingInput,
  type SavedReading,
  type VitalAlert,
  type VitalReading,
  type VitalType,
} from '@/entities/vital';
import type { Locale } from '@/shared/i18n';
import { Card, ChipGroup, Icon, IconCircle, InfoNote, PillChip, PrimaryButton, StatusPill } from '@/shared/ui';

import {
  measuredAtBody,
  validateBp,
  validateGlucose,
  validateHr,
  validateMeasuredAt,
  validateNote,
  type FieldError,
  type FormErrors,
} from '../model/validate';
import { MeasuredAtField, type MeasuredAt } from './MeasuredAtField';
import { UrgentAlertSheet } from './UrgentAlertSheet';
import { ValueInput } from './ValueInput';

type T = ReturnType<typeof useTranslations<'vitals'>>;

function errorText(t: T, e: FieldError | undefined, f: ReturnType<typeof useVitalFormat>): string | undefined {
  if (!e) return undefined;
  if (e.key === 'range') return t('add.errors.range', { min: f.dec(e.min, Number.isInteger(e.min) ? 0 : 1), max: f.dec(e.max, Number.isInteger(e.max) ? 0 : 1) });
  if (e.key === 'noteTooLong') return t('add.errors.noteTooLong', { max: f.num(e.max) });
  return t(`add.errors.${e.key}`);
}

export interface AddVitalFormProps {
  type: VitalType;
  /** The newest reading of this type (hub cache) — the form starts from it. */
  latest?: VitalReading | null;
  /** Screen header, rendered at the top of the scroll. */
  header: ReactNode;
  /** After a save (and after the urgent modal is acknowledged). */
  onDone: () => void;
}

/**
 * Add a blood-pressure, glucose or heart-rate reading (nbl_Vitals_AddBP /
 * AddGlucose / AddHR): validation mirrors the server, `POST /vitals/readings`
 * stores it, and an urgent `alert` in the answer opens the call modal before
 * leaving — the reading is saved either way.
 */
export function AddVitalForm({ type, latest, header, onDone }: AddVitalFormProps) {
  const t = useTranslations('vitals');
  const create = useCreateReading();
  const [alert, setAlert] = useState<VitalAlert | null>(null);
  const [at, setAt] = useState<MeasuredAt>(null);
  const [note, setNote] = useState('');
  const [tried, setTried] = useState(false);

  const submit = (input: ReadingInput | null, errors: FormErrors) => {
    setTried(true);
    if (!input || Object.keys(errors).length || create.isPending) return;
    create.mutate(input, {
      onSuccess: (saved: SavedReading) => {
        if (saved.alert) setAlert(saved.alert);
        else onDone();
      },
    });
  };

  const common = {
    at,
    note,
    tried,
    server: fieldErrors(create.error),
    commonErrors: { ...validateMeasuredAt(at, Date.now()), ...validateNote(note) },
    input: { measuredAt: measuredAtBody(at), note: note.trim() || null },
  };

  const footer = (onSave: () => void) => (
    <div className="vt-foot">
      {create.isError && !Object.keys(common.server).length ? (
        <p className="vt-error" role="alert">
          {t('add.saveError')}
        </p>
      ) : null}
      <PrimaryButton loading={create.isPending} onClick={onSave}>
        {t('add.save')}
      </PrimaryButton>
    </div>
  );

  const tail = (
    <>
      <MeasuredAtField value={at} onChange={setAt} />
      <NoteField type={type} value={note} onChange={setNote} />
    </>
  );

  return (
    <>
      {type === 'bp' ? (
        <BpForm latest={latest} header={header} tail={tail} common={common} footer={footer} submit={submit} />
      ) : type === 'glucose' ? (
        <GlucoseForm latest={latest} header={header} tail={tail} common={common} footer={footer} submit={submit} />
      ) : (
        <HrForm latest={latest} header={header} tail={tail} common={common} footer={footer} submit={submit} />
      )}
      {alert ? (
        <UrgentAlertSheet
          alert={alert}
          onDone={() => {
            setAlert(null);
            onDone();
          }}
        />
      ) : null}
    </>
  );
}

interface Common {
  at: MeasuredAt;
  note: string;
  tried: boolean;
  server: Record<string, string>;
  commonErrors: FormErrors;
  input: { measuredAt: string | null; note: string | null };
}

interface SubFormProps {
  latest?: VitalReading | null;
  header: ReactNode;
  tail: ReactNode;
  common: Common;
  footer: (onSave: () => void) => ReactNode;
  submit: (input: ReadingInput | null, errors: FormErrors) => void;
}

function useErr(common: Common) {
  const t = useTranslations('vitals');
  const f = useVitalFormat();
  return (errors: FormErrors, field: keyof FormErrors): string | undefined =>
    common.server[field] ?? (common.tried ? errorText(t, errors[field], f) : undefined);
}

function NoteField({ type, value, onChange }: { type: VitalType; value: string; onChange: (v: string) => void }) {
  const t = useTranslations('vitals.add');
  return (
    <label className="vt-note">
      <span className="vt-field-title">{t('note')}</span>
      <span className="nb-card vt-note-box">
        <Icon name="note" size={18} className="vt-note-icon" />
        <input
          className="vt-note-input"
          type="text"
          value={value}
          maxLength={LIMITS.noteMax}
          placeholder={t(`notePlaceholder.${type}`)}
          onChange={(e) => onChange(e.target.value)}
        />
      </span>
    </label>
  );
}

function Err({ id, text }: { id: string; text?: string }) {
  return text ? (
    <p id={id} className="vt-error" role="alert">
      {text}
    </p>
  ) : null;
}

function BpForm({ latest, header, tail, common, footer, submit }: SubFormProps) {
  const t = useTranslations('vitals');
  const locale = useLocale() as Locale;
  const th = useVitalThresholds();
  const err = useErr(common);
  const last = latest?.bloodPressure;
  const [sys, setSys] = useState<number | null>(last?.systolic ?? 120);
  const [dia, setDia] = useState<number | null>(last?.diastolic ?? 80);
  const [pulse, setPulse] = useState<number | null>(last?.pulse ?? null);
  const [arm, setArm] = useState<Arm | null>(last?.arm ?? 'left');
  const [position, setPosition] = useState<Position | null>(last?.position ?? 'sitting');
  const errors = { ...validateBp(sys, dia, pulse), ...common.commonErrors };
  const valid = !errors.systolic && !errors.diastolic && sys !== null && dia !== null;
  const cls = valid ? classifyBp(sys, dia, th) : null;
  const pos = valid ? bpScalePosition(sys, dia, th) : null;
  const input: ReadingInput | null =
    sys !== null && dia !== null ? { type: 'bp', systolic: sys, diastolic: dia, pulse, arm, position, ...common.input } : null;
  const tile = (key: 'systolic' | 'diastolic' | 'pulse', value: number | null, set: (v: number | null) => void, lim: { min: number; max: number }, unit: string, tone: 'brand' | 'bloom') => (
    <ValueInput
      label={t(`add.${key}`)}
      value={value}
      onChange={set}
      step={1}
      min={lim.min}
      max={lim.max}
      unit={unit}
      tone={tone}
      locale={locale}
      increaseLabel={t('add.increase', { label: t(`add.${key}`) })}
      decreaseLabel={t('add.decrease', { label: t(`add.${key}`) })}
      invalid={!!err(errors, key)}
      errorId={`vt-err-${key}`}
      startAt={key === 'pulse' ? 70 : undefined}
    />
  );
  return (
    <>
      <div className="scroll vt-scroll">
        {header}
        <div className="vt-form">
          <div className="vt-tiles">
            {tile('systolic', sys, setSys, LIMITS.systolic, t('units.mmhg'), 'brand')}
            {tile('diastolic', dia, setDia, LIMITS.diastolic, t('units.mmhg'), 'brand')}
            {tile('pulse', pulse, setPulse, LIMITS.pulse, t('units.bpm'), 'bloom')}
          </div>
          <Err id="vt-err-systolic" text={err(errors, 'systolic')} />
          <Err id="vt-err-diastolic" text={err(errors, 'diastolic')} />
          <Err id="vt-err-pulse" text={err(errors, 'pulse')} />

          <Card as="section" className="vt-card">
            <div className="vt-card-head">
              <h2 className="vt-card-title">{t('add.meaning')}</h2>
              {cls ? <StatusPill tone={uiTone(cls.tone)}>{t(`classes.bp.${cls.code}` as 'classes.bp.normal')}</StatusPill> : null}
            </div>
            <div className="vt-scale" role="img" aria-label={cls ? t('add.scaleLabel', { class: t(`classes.bp.${cls.code}` as 'classes.bp.normal') }) : t('add.meaning')}>
              <div className="vt-scale-bar">
                {BP_SCALE.map((c) => (
                  <span key={c} className={clsx('vt-scale-seg', `is-${c}`)} />
                ))}
                {pos !== null ? <span className="vt-scale-mark" style={{ insetInlineStart: `${pos}%` }} /> : null}
              </div>
              <div className="vt-scale-labels" aria-hidden>
                {BP_SCALE.map((c) => (
                  <span key={c}>{t(`classes.bp.${c}`)}</span>
                ))}
              </div>
            </div>
            <p className="vt-card-note">{t('add.disclaimer')}</p>
          </Card>

          <Card as="section" className="vt-card">
            <h2 className="vt-card-title">{t('add.conditions')}</h2>
            <ChipGroup label={t('add.arm')}>
              {ARMS.map((a) => (
                <PillChip key={a} pressed={arm === a} onPressedChange={(on) => setArm(on ? a : null)}>
                  {t(`arms.${a}`)}
                </PillChip>
              ))}
            </ChipGroup>
            <ChipGroup label={t('add.position')}>
              {POSITIONS.map((p) => (
                <PillChip key={p} pressed={position === p} onPressedChange={(on) => setPosition(on ? p : null)}>
                  {t(`positions.${p}`)}
                </PillChip>
              ))}
            </ChipGroup>
          </Card>
          {tail}
          <Err id="vt-err-note" text={err(errors, 'note')} />
          <Err id="vt-err-at" text={err(errors, 'measured_at')} />
        </div>
      </div>
      {footer(() => submit(input, errors))}
    </>
  );
}

function GlucoseForm({ latest, header, tail, common, footer, submit }: SubFormProps) {
  const t = useTranslations('vitals');
  const locale = useLocale() as Locale;
  const f = useVitalFormat();
  const th = useVitalThresholds();
  const err = useErr(common);
  const last = latest?.glucose;
  const [unit, setUnit] = useState<GlucoseUnit>(last?.unit ?? 'mg_dl');
  const [value, setValue] = useState<number | null>(last ? last.value : unit === 'mmol_l' ? 5.3 : 95);
  const [context, setContext] = useState<GlucoseContext>('fasting');
  const [method, setMethod] = useState<GlucoseMethod | null>(last?.method ?? 'glucometer');
  const errors = { ...validateGlucose(value, unit), ...common.commonErrors };
  const lim = unit === 'mmol_l' ? LIMITS.mmolL : LIMITS.mgDl;
  const cls = value !== null && !errors.value ? classifyGlucose(toMgDl(value, unit), context, th) : null;
  const band = th.glucose.contexts[context];
  const inUnit = (mg: number) => {
    const v = unit === 'mmol_l' ? Math.round((mg / th.glucose.mmolFactor) * 10) / 10 : mg;
    return f.dec(v, glucoseDecimals(unit));
  };
  const verdict = cls
    ? t(`add.glucoseVerdict.${cls.code}` as 'add.glucoseVerdict.in_range', {
        context: t(`glucoseContextsShort.${context}`),
        min: inUnit(band.targetMin),
        max: inUnit(band.targetMax),
        value: inUnit(th.glucose.urgentBelow),
      })
    : null;
  const switchUnit = (next: GlucoseUnit) => {
    if (next === unit) return;
    setValue((v) => (v === null ? v : convertGlucose(v, unit, next)));
    setUnit(next);
  };
  const input: ReadingInput | null = value !== null ? { type: 'glucose', value, unit, context, method, ...common.input } : null;
  return (
    <>
      <div className="scroll vt-scroll">
        {header}
        <div className="vt-form">
          <Card as="section" className="vt-card vt-hero">
            <div className="vt-unit" role="radiogroup" aria-label={t('add.unit')}>
              {GLUCOSE_UNITS.map((u) => (
                <button key={u} type="button" role="radio" aria-checked={unit === u} className="vt-unit-opt" dir="ltr" onClick={() => switchUnit(u)}>
                  {unitSymbol(u)}
                </button>
              ))}
            </div>
            <ValueInput
              key={unit}
              layout="row"
              label={t('types.glucose')}
              hideLabel
              value={value}
              onChange={setValue}
              step={glucoseStep(unit)}
              decimals={glucoseDecimals(unit)}
              min={lim.min}
              max={lim.max}
              tone="warm"
              locale={locale}
              increaseLabel={t('add.increase', { label: t('types.glucose') })}
              decreaseLabel={t('add.decrease', { label: t('types.glucose') })}
              invalid={!!err(errors, 'value')}
              errorId="vt-err-value"
            />
            {verdict && cls ? (
              <StatusPill tone={uiTone(cls.tone)} className="vt-verdict">
                {verdict}
              </StatusPill>
            ) : null}
            <Err id="vt-err-value" text={err(errors, 'value')} />
          </Card>

          <Card as="section" className="vt-card">
            <h2 className="vt-card-title">{t('add.when')}</h2>
            <ChipGroup label={t('add.when')}>
              {GLUCOSE_CONTEXTS.map((c) => (
                <PillChip key={c} pressed={context === c} onPressedChange={() => setContext(c)}>
                  {t(`glucoseContexts.${c}`)}
                </PillChip>
              ))}
            </ChipGroup>
          </Card>
          <Card as="section" className="vt-card">
            <h2 className="vt-card-title">{t('add.method')}</h2>
            <ChipGroup label={t('add.method')}>
              {GLUCOSE_METHODS.map((m) => (
                <PillChip key={m} pressed={method === m} onPressedChange={(on) => setMethod(on ? m : null)}>
                  {t(`methods.${m}`)}
                </PillChip>
              ))}
            </ChipGroup>
          </Card>
          {tail}
          <Err id="vt-err-note" text={err(errors, 'note')} />
          <Err id="vt-err-at" text={err(errors, 'measured_at')} />
        </div>
      </div>
      {footer(() => submit(input, errors))}
    </>
  );
}

function HrForm({ latest, header, tail, common, footer, submit }: SubFormProps) {
  const t = useTranslations('vitals');
  const locale = useLocale() as Locale;
  const f = useVitalFormat();
  const th = useVitalThresholds();
  const err = useErr(common);
  const [bpm, setBpm] = useState<number | null>(latest?.heartRate?.bpm ?? 72);
  const [context, setContext] = useState<HrContext>('resting');
  const errors = { ...validateHr(bpm), ...common.commonErrors };
  const cls = bpm !== null && !errors.bpm ? classifyHr(bpm, context, th) : null;
  const state = t(`hrStates.${context}`);
  const verdict = bpm === null || errors.bpm
    ? null
    : cls
      ? t('add.hrVerdict', { state, class: t(`classes.hr.${cls.code}` as 'classes.hr.normal'), min: f.num(th.hr.min), max: f.num(th.hr.max) })
      : t('add.hrUnclassified', { state });
  const input: ReadingInput | null = bpm !== null ? { type: 'hr', bpm, context, ...common.input } : null;
  return (
    <>
      <div className="scroll vt-scroll">
        {header}
        <div className="vt-form">
          <Card as="section" className="vt-card vt-hero">
            <IconCircle icon="heartLine" tone="bloom" size="lg" className="vt-hero-icon" />
            <ValueInput
              layout="row"
              label={t('types.hr')}
              hideLabel
              value={bpm}
              onChange={setBpm}
              step={1}
              min={LIMITS.pulse.min}
              max={LIMITS.pulse.max}
              unit={t('units.bpmLong')}
              tone="bloom"
              locale={locale}
              increaseLabel={t('add.increase', { label: t('types.hr') })}
              decreaseLabel={t('add.decrease', { label: t('types.hr') })}
              invalid={!!err(errors, 'bpm')}
              errorId="vt-err-bpm"
            />
            {verdict ? (
              <StatusPill tone={cls ? uiTone(cls.tone) : 'neutral'} className="vt-verdict">
                {verdict}
              </StatusPill>
            ) : null}
            <Err id="vt-err-bpm" text={err(errors, 'bpm')} />
          </Card>
          <Card as="section" className="vt-card">
            <h2 className="vt-card-title">{t('add.hrState')}</h2>
            <ChipGroup label={t('add.hrState')}>
              {HR_CONTEXTS.map((c) => (
                <PillChip key={c} pressed={context === c} onPressedChange={() => setContext(c)}>
                  {t(`hrContexts.${c}`)}
                </PillChip>
              ))}
            </ChipGroup>
          </Card>
          <InfoNote>{t('add.hrTip')}</InfoNote>
          {tail}
          <Err id="vt-err-note" text={err(errors, 'note')} />
          <Err id="vt-err-at" text={err(errors, 'measured_at')} />
        </div>
      </div>
      {footer(() => submit(input, errors))}
    </>
  );
}
