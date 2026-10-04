'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  IVF_STAGES,
  type IvfCycle,
  type IvfMed,
  ivfValidationMessage,
  useIvfHome,
  useIvfMeds,
  useIvfProtocols,
  useIvfStages,
  useSetIvfMedActive,
  useUpdateIvfCycle,
} from '@/entities/ivf';
import { getApiSaveErrorMessage } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { AppSheet } from '@/shared/sheet';
import {
  Card,
  EmptyState,
  InfoNote,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import {
  type CycleDraft,
  type DateField,
  draftFromCycle,
  draftProblems,
  draftToPatch,
  medsToStop,
  setDraftStage,
} from '../model/cycle';
import { ChoiceRow, DateSheet, FieldButton, FieldError, TimeSheet } from './Fields';
import { protocolCodes, useProtocolLabel } from './protocols';

type ClockField = 'nextScan' | 'retrieval' | 'transfer';
type DayField = 'startedOn' | 'stimStartedOn' | 'betaOn';
type Picker = { kind: 'date'; field: DayField | ClockField } | { kind: 'time'; field: ClockField } | null;

function Shell({ children }: { children: React.ReactNode }) {
  const t = useTranslations('ivf.cycle');
  const router = useRouter();
  return (
    <div className="view ivfm-form-page ivfc-page">
      <SkyLayer />
      <div className="scroll is-form">
        <ScreenHeader title={t('editor.title')} onBack={() => router.push('/ivf')} backLabel={t('back')} />
        {children}
      </div>
    </div>
  );
}

/**
 * «مرحله و تاریخ‌ها» — `/ivf/cycle` (CB-IVF-06b), opened from the IVF home
 * timeline: the current stage, protocol and the cycle's dates (start,
 * stimulation start, next scan, retrieval, transfer, beta) → `PUT
 * /ivf/cycles/current` with only what changed; the API moves the linked care
 * appointments. Moving past stimulation offers to pause the stimulation
 * medicines. A form: no bottom nav.
 */
export function IvfCycleEditorPage() {
  const t = useTranslations('ivf.cycle');
  const mounted = useMounted();
  const home = useIvfHome();

  if (!mounted || home.isPending) {
    return (
      <Shell>
        <SkeletonGroup label={t('loading')} className="ivfm-form">
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  const cycle = home.data?.cycle;
  if (home.isError || !cycle) {
    return (
      <Shell>
        <div className="ivfm-form">
          <Card>
            <EmptyState
              icon="drop"
              title={home.isError ? t('editor.saveError') : t('editor.noCycle')}
              action={
                home.isError ? undefined : (
                  <Link href="/ivf/cycle/new" className="nb-btn is-primary is-block">
                    {t('editor.noCycleCta')}
                  </Link>
                )
              }
            />
          </Card>
        </div>
      </Shell>
    );
  }

  return (
    <Shell>
      {/* Keyed by cycle: a different cycle starts a fresh draft. */}
      <EditorForm key={cycle.id} cycle={cycle} />
    </Shell>
  );
}

function EditorForm({ cycle }: { cycle: IvfCycle }) {
  const t = useTranslations('ivf.cycle');
  const ti = useTranslations('ivf');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const stages = useIvfStages(locale);
  const protocols = useIvfProtocols(locale);
  const protocolLabel = useProtocolLabel(protocols.data);
  const meds = useIvfMeds();
  const update = useUpdateIvfCycle();
  const todayIso = toApiDate(today());
  const [draft, setDraft] = useState<CycleDraft>(() => draftFromCycle(cycle));
  const [picker, setPicker] = useState<Picker>(null);
  const [stopList, setStopList] = useState<IvfMed[]>([]);

  const problems = draftProblems(draft, todayIso);
  const blocked = Object.keys(problems).length > 0;
  const problemText = (field: DateField) => (problems[field] ? t(`editor.errors.${problems[field]}`) : null);
  const dateValue = (value: string | null) => (value ? formatDayMonth(fromApiDate(value), locale) : t('choose'));
  const stageTitle = (stage: (typeof IVF_STAGES)[number]) =>
    stages.data?.find((s) => s.code === stage)?.title ?? ti(`stages.${stage}.title`);
  const codes = protocolCodes(protocols.data);
  const protocolOptions = draft.protocol && !codes.includes(draft.protocol) ? [...codes, draft.protocol] : codes;

  const submit = () => {
    if (blocked) return;
    const patch = draftToPatch(draft, cycle);
    const toStop = medsToStop(meds.data?.meds ?? [], cycle.stage, draft.stage);
    const after = () => (toStop.length ? setStopList(toStop) : router.replace('/ivf'));
    if (Object.keys(patch).length === 0) {
      router.replace('/ivf');
      return;
    }
    update.mutate(patch, { onSuccess: after });
  };

  const dayField = (field: DayField, optional: boolean) => (
    <FieldButton
      label={t(`editor.${field === 'betaOn' ? 'beta' : field}`)}
      hint={optional ? t('optional') : undefined}
      value={dateValue(draft[field])}
      invalid={!!problemText(field)}
      onClick={() => setPicker({ kind: 'date', field })}
      onClear={optional && draft[field] ? () => setDraft((d) => ({ ...d, [field]: null })) : undefined}
      clearLabel={t('clear', { what: t(`editor.${field === 'betaOn' ? 'beta' : field}`) })}
    />
  );

  const clockField = (field: ClockField) => {
    const value = draft[field];
    const label = t(`editor.${field}`);
    return (
      <div className="ivfm-group" key={field}>
        <span className="ivfm-label">
          {label} <span className="ivfm-optional">{t('optional')}</span>
        </span>
        <div className="ivfm-two">
          <FieldButton
            label={t('date')}
            value={dateValue(value.date)}
            invalid={!!problemText(field)}
            onClick={() => setPicker({ kind: 'date', field })}
            onClear={value.date ? () => setDraft((d) => ({ ...d, [field]: { ...d[field], date: null } })) : undefined}
            clearLabel={t('clear', { what: label })}
          />
          {value.date ? (
            <FieldButton
              label={t('time')}
              value={formatNumber(value.time, locale)}
              ltr
              onClick={() => setPicker({ kind: 'time', field })}
            />
          ) : null}
        </div>
        <FieldError text={problemText(field)} />
      </div>
    );
  };

  const pickedDate = (field: DayField | ClockField) =>
    field === 'nextScan' || field === 'retrieval' || field === 'transfer' ? draft[field].date : draft[field];

  return (
    <div className="ivfm-form">
      <Card className="ivfm-card">
        <ChoiceRow
          label={t('editor.stage')}
          options={IVF_STAGES}
          value={draft.stage}
          labelOf={stageTitle}
          onChange={(stage) => stage && setDraft((d) => setDraftStage(d, stage, todayIso))}
        />
        <ChoiceRow
          label={t('editor.protocol')}
          options={protocolOptions}
          value={draft.protocol}
          labelOf={protocolLabel}
          allowNone
          onChange={(protocol) => setDraft((d) => ({ ...d, protocol }))}
        />
      </Card>

      <Card className="ivfm-card" as="section" aria-labelledby="ivfc-dates-title">
        <div className="ivfm-block">
          <h2 id="ivfc-dates-title" className="ivfm-block-title">
            {t('editor.dates')}
          </h2>
          <p className="ivfm-block-hint">{t('editor.datesHint')}</p>
        </div>
        <div className="ivfm-two">
          {dayField('startedOn', false)}
          {dayField('stimStartedOn', true)}
        </div>
        <FieldError text={problemText('startedOn') ?? problemText('stimStartedOn')} />
        {clockField('nextScan')}
        {clockField('retrieval')}
        {clockField('transfer')}
        <div className="ivfm-group">
          {dayField('betaOn', true)}
          <FieldError text={problemText('betaOn')} />
        </div>
        {(draft.stage === 'tww' || draft.stage === 'transfer') && !draft.betaOn ? (
          <InfoNote>{t('editor.betaHint')}</InfoNote>
        ) : null}
      </Card>

      <div className="ivfm-footer">
        {update.isError ? (
          <p className="ivfm-error" role="alert">
            {ivfValidationMessage(update.error) ?? getApiSaveErrorMessage(update.error, t('editor.saveError'))}
          </p>
        ) : null}
        <PrimaryButton loading={update.isPending} disabled={blocked} onClick={submit}>
          {t('editor.save')}
        </PrimaryButton>
      </div>

      {picker?.kind === 'date' ? (
        <DateSheet
          title={t(`editor.${picker.field === 'betaOn' ? 'beta' : picker.field}`)}
          value={pickedDate(picker.field)}
          onClose={() => setPicker(null)}
          onPick={(value) => {
            const field = picker.field;
            setDraft((d) =>
              field === 'nextScan' || field === 'retrieval' || field === 'transfer'
                ? { ...d, [field]: { ...d[field], date: value } }
                : { ...d, [field]: value },
            );
            setPicker(null);
          }}
        />
      ) : null}
      {picker?.kind === 'time' ? (
        <TimeSheet
          title={t(`editor.${picker.field}`)}
          value={draft[picker.field].time}
          onClose={() => setPicker(null)}
          onPick={(time) => {
            const field = picker.field;
            setDraft((d) => ({ ...d, [field]: { ...d[field], time } }));
            setPicker(null);
          }}
        />
      ) : null}
      {stopList.length ? <StopMedsSheet meds={stopList} onDone={() => router.replace('/ivf')} /> : null}
    </div>
  );
}

/** «داروهای تحریک متوقف شوند؟» — after a move past stimulation; «فعلاً بماند» leaves them as they are. */
function StopMedsSheet({ meds, onDone }: { meds: IvfMed[]; onDone: () => void }) {
  const t = useTranslations('ivf.cycle');
  const setActive = useSetIvfMedActive();
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState(false);
  const stopAll = async () => {
    setBusy(true);
    setFailed(false);
    try {
      for (const med of meds) {
        if (med.reminderId !== null) await setActive.mutateAsync({ reminderId: med.reminderId, active: false });
      }
      onDone();
    } catch {
      setFailed(true);
    } finally {
      setBusy(false);
    }
  };
  return (
    <AppSheet
      open
      onClose={onDone}
      size="half"
      title={t('stop.title')}
      footer={
        <div className="ivfm-sheet-btns">
          <SecondaryButton block={false} onClick={onDone} disabled={busy}>
            {t('stop.keep')}
          </SecondaryButton>
          <PrimaryButton block={false} loading={busy} onClick={() => void stopAll()}>
            {t('stop.confirm')}
          </PrimaryButton>
        </div>
      }
    >
      <p className="ivfm-sheet-body">{t('stop.body')}</p>
      <ul className="ivfc-stop-list">
        {meds.map((med) => (
          <li key={med.id}>{med.name}</li>
        ))}
      </ul>
      {failed ? (
        <p className="ivfm-error" role="alert">
          {t('stop.error')}
        </p>
      ) : null}
    </AppSheet>
  );
}
