'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type FormEvent, useEffect, useId, useState } from 'react';

import {
  type Allergies,
  type Basics,
  BLOOD_TYPES,
  type BloodType,
  CHRONIC_ILLNESSES,
  type Conditions,
  GYN_CONDITIONS,
  MANUAL_OUTCOMES,
  MAX_ALLERGIES,
  MAX_ALLERGY_LENGTH,
  MAX_BABY_COUNT,
  type ManualOutcome,
  type PregnancyEntry,
  addAllergy,
  removeAllergy,
  toggleCode,
  useDeleteRecordPregnancy,
  useSaveRecordBasics,
  useSaveRecordConditions,
  useSaveRecordPregnancy,
} from '@/entities/health-record';
import type { QuickEditField } from '@/features/edit-profile';
import type { Locale } from '@/shared/i18n';
import { formatNumber, fromApiDate, partsToDate, toApiDate, todayParts, toParts } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  ChipGroup,
  Icon,
  ListGroup,
  ListRow,
  NumberStepper,
  PillChip,
  PrimaryButton,
  SecondaryButton,
  SectionTitle,
  Switch,
} from '@/shared/ui';

import { manualEntries } from '../model/format';

interface SheetProps {
  open: boolean;
  onClose: () => void;
}

function SaveError({ show }: { show: boolean }) {
  const t = useTranslations('healthRecord');
  return show ? (
    <p role="alert" className="hrec-error">
      {t('error.save')}
    </p>
  ) : null;
}

/* ── Basics: blood type here; height / weight through the profile quick-edit sheet ─────────────────────────────── */

export function BasicsSheet({
  open,
  onClose,
  basics,
  onEditBody,
}: SheetProps & { basics: Basics; onEditBody: (field: QuickEditField) => void }) {
  const t = useTranslations('healthRecord');
  const loc = useLocale() as Locale;
  const save = useSaveRecordBasics();
  const stored = basics.bloodTypeSource === 'record' ? (basics.bloodType as BloodType | null) : null;
  const [blood, setBlood] = useState<BloodType | null>(stored);
  useEffect(() => {
    if (open) {
      setBlood(stored);
      save.reset();
    }
    // Re-seed only when the sheet opens.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const submit = () => save.mutate({ bloodType: blood }, { onSuccess: onClose });
  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="full"
      title={t('bloodSheet.title')}
      footer={
        <PrimaryButton onClick={submit} loading={save.isPending}>
          {save.isPending ? t('saving') : t('save')}
        </PrimaryButton>
      }
    >
      <div className="hrec-sheet">
        <SectionTitle title={t('bloodSheet.bloodType')} level={3} />
        <ChipGroup label={t('bloodSheet.bloodType')} className="hrec-blood">
          {BLOOD_TYPES.map((b) => (
            <PillChip key={b} pressed={blood === b} onPressedChange={() => setBlood(b)}>
              <bdi dir="ltr">{b}</bdi>
            </PillChip>
          ))}
          <PillChip pressed={blood === null} onPressedChange={() => setBlood(null)}>
            {t('bloodSheet.unknown')}
          </PillChip>
        </ChipGroup>
        <SectionTitle title={t('bloodSheet.bodyTitle')} level={3} />
        <ListGroup>
          <ListRow
            icon="ruler"
            title={t('bloodSheet.editHeight')}
            value={basics.heightCm !== null ? t('basics.heightValue', { value: formatNumber(basics.heightCm, loc) }) : t('notSet')}
            onClick={() => onEditBody('height')}
          />
          <ListRow
            icon="scale"
            title={t('bloodSheet.editWeight')}
            value={basics.weightKg !== null ? t('basics.weightValue', { value: formatNumber(basics.weightKg, loc) }) : t('notSet')}
            onClick={() => onEditBody('weight')}
          />
        </ListGroup>
        <SaveError show={save.isError} />
      </div>
    </AppSheet>
  );
}

/* ── Conditions: the onboarding Conditions answers ─────────────────────────────────────────────────────────────── */

export function ConditionsSheet({
  open,
  onClose,
  conditions,
  medications,
}: SheetProps & { conditions: Conditions; medications: string[] | null }) {
  const t = useTranslations('healthRecord');
  const save = useSaveRecordConditions();
  const [chronic, setChronic] = useState<string[]>([]);
  const [gyn, setGyn] = useState<string[]>([]);
  useEffect(() => {
    if (open) {
      setChronic(conditions.chronicIllnesses ?? []);
      setGyn(conditions.gynConditions ?? []);
      save.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const submit = () =>
    save.mutate({ chronicIllnesses: chronic, gynConditions: gyn, medications }, { onSuccess: onClose });
  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="full"
      title={t('conditionsSheet.title')}
      footer={
        <PrimaryButton onClick={submit} loading={save.isPending}>
          {save.isPending ? t('saving') : t('save')}
        </PrimaryButton>
      }
    >
      <div className="hrec-sheet">
        <p className="hrec-hint">{t('conditionsSheet.hint')}</p>
        <SectionTitle title={t('conditions.chronicTitle')} level={3} />
        <ChipGroup label={t('conditions.chronicTitle')}>
          {CHRONIC_ILLNESSES.map((c) => (
            <PillChip
              key={c}
              mode="multi"
              tone="period"
              pressed={chronic.includes(c)}
              onPressedChange={() => setChronic((l) => toggleCode(l, c, CHRONIC_ILLNESSES))}
            >
              {t(`conditions.chronicIllness.${c}`)}
            </PillChip>
          ))}
        </ChipGroup>
        <SectionTitle title={t('conditions.gynTitle')} level={3} />
        <ChipGroup label={t('conditions.gynTitle')}>
          {GYN_CONDITIONS.map((c) => (
            <PillChip
              key={c}
              mode="multi"
              tone="period"
              pressed={gyn.includes(c)}
              onPressedChange={() => setGyn((l) => toggleCode(l, c, GYN_CONDITIONS))}
            >
              {t(`conditions.gynCondition.${c}`)}
            </PillChip>
          ))}
        </ChipGroup>
        <SaveError show={save.isError} />
      </div>
    </AppSheet>
  );
}

/* ── Allergies: a short free-text list, or «ندارم» ─────────────────────────────────────────────────────────────── */

export function AllergiesSheet({ open, onClose, allergies }: SheetProps & { allergies: Allergies }) {
  const t = useTranslations('healthRecord');
  const save = useSaveRecordBasics();
  const inputId = useId();
  const [items, setItems] = useState<string[]>([]);
  const [none, setNone] = useState(false);
  const [draft, setDraft] = useState('');
  useEffect(() => {
    if (open) {
      setItems(allergies.items ?? []);
      setNone(allergies.items !== null && allergies.items.length === 0);
      setDraft('');
      save.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const add = (e?: FormEvent) => {
    e?.preventDefault();
    const next = addAllergy(items, draft);
    if (next.length !== items.length) {
      setItems(next);
      setNone(false);
    }
    setDraft('');
  };
  const submit = () => {
    const pending = draft.trim() ? addAllergy(items, draft) : items;
    const value = none ? [] : pending.length ? pending : allergies.items === null ? null : [];
    save.mutate({ allergies: value }, { onSuccess: onClose });
  };
  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="full"
      title={t('sections.allergies')}
      footer={
        <PrimaryButton onClick={submit} loading={save.isPending}>
          {save.isPending ? t('saving') : t('save')}
        </PrimaryButton>
      }
    >
      <div className="hrec-sheet">
        <p className="hrec-hint">{t('allergies.hint')}</p>
        <form className="hrec-add" onSubmit={add}>
          <label htmlFor={inputId} className="hrec-label">
            {t('allergies.inputLabel')}
          </label>
          <div className="hrec-add-row">
            <input
              id={inputId}
              className="hrec-input"
              value={draft}
              maxLength={MAX_ALLERGY_LENGTH}
              placeholder={t('allergies.placeholder')}
              disabled={none || items.length >= MAX_ALLERGIES}
              onChange={(e) => setDraft(e.target.value)}
              autoComplete="off"
            />
            <button type="submit" className="hrec-add-btn" disabled={none || !draft.trim()} aria-label={t('add')}>
              <Icon name="plus" size={20} />
            </button>
          </div>
        </form>
        {items.length > 0 && !none ? (
          <ul className="hrec-tags is-removable" aria-label={t('sections.allergies')}>
            {items.map((a) => (
              <li key={a} className="hrec-tag">
                <span>{a}</span>
                <button
                  type="button"
                  className="hrec-tag-x"
                  aria-label={t('allergies.remove', { name: a })}
                  onClick={() => setItems((l) => removeAllergy(l, a))}
                >
                  <Icon name="x" size={14} />
                </button>
              </li>
            ))}
          </ul>
        ) : null}
        <ListGroup>
          <ListRow
            id={`${inputId}-none`}
            title={t('allergies.none')}
            trailing={<Switch checked={none} onCheckedChange={setNone} labelledBy={`${inputId}-none-title`} />}
          />
        </ListGroup>
        <SaveError show={save.isError} />
      </div>
    </AppSheet>
  );
}

/* ── Pregnancies: the entries the user adds by hand ───────────────────────────────────────────────────────────── */

const YEARS_BACK = 60;

interface Draft {
  id: number | null;
  outcome: ManualOutcome;
  year: number | null; // in the locale's calendar
  babies: number;
}

export function PregnanciesSheet({ open, onClose, items }: SheetProps & { items: PregnancyEntry[] }) {
  const t = useTranslations('healthRecord');
  const loc = useLocale() as Locale;
  const save = useSaveRecordPregnancy();
  const remove = useDeleteRecordPregnancy();
  const yearId = useId();
  const [draft, setDraft] = useState<Draft | null>(null);
  const manual = manualEntries(items);
  const thisYear = todayParts(loc).year;
  useEffect(() => {
    if (open) {
      setDraft(manual.length ? null : { id: null, outcome: 'vaginal', year: null, babies: 1 });
      save.reset();
      remove.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const yearOf = (iso: string | null) => (iso ? toParts(fromApiDate(iso), loc).year : null);
  const edit = (e: PregnancyEntry & { id: number }) =>
    setDraft({
      id: e.id,
      outcome: (MANUAL_OUTCOMES as readonly string[]).includes(e.outcome) ? (e.outcome as ManualOutcome) : 'vaginal',
      year: yearOf(e.date),
      babies: e.babyCount ?? 1,
    });

  const submit = () => {
    if (!draft) return;
    const endedOn = draft.year !== null ? toApiDate(partsToDate({ year: draft.year, month: 1, day: 1 }, loc)) : null;
    save.mutate(
      { id: draft.id, input: { outcome: draft.outcome, endedOn, babyCount: draft.outcome === 'ended' ? null : draft.babies } },
      { onSuccess: () => setDraft(null) },
    );
  };

  const entryLabel = (e: PregnancyEntry) => {
    const y = yearOf(e.date);
    return [t(`pregnancies.outcome.${e.outcome}`), y !== null ? formatNumber(y, loc) : null].filter(Boolean).join(' · ');
  };

  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="full"
      title={t('sections.pregnancies')}
      footer={
        draft ? (
          <div className="hrec-sheet-actions">
            <PrimaryButton onClick={submit} loading={save.isPending}>
              {save.isPending ? t('saving') : t('save')}
            </PrimaryButton>
            {manual.length ? (
              <SecondaryButton variant="text" onClick={() => setDraft(null)}>
                {t('cancel')}
              </SecondaryButton>
            ) : null}
          </div>
        ) : (
          <PrimaryButton icon="plus" onClick={() => setDraft({ id: null, outcome: 'vaginal', year: null, babies: 1 })}>
            {t('pregnancies.addTitle')}
          </PrimaryButton>
        )
      }
    >
      <div className="hrec-sheet">
        <p className="hrec-hint">{t('pregnancies.sheetHint')}</p>
        {!draft && manual.length > 0 ? (
          <ul className="hrec-entries">
            {manual.map((e) => (
              <li key={e.id} className="hrec-entry">
                <button type="button" className="hrec-entry-main" onClick={() => edit(e)}>
                  <span className="hrec-row-label">{entryLabel(e)}</span>
                  <span className="hrec-row-value">{t('pregnancies.manual')}</span>
                </button>
                <button
                  type="button"
                  className="hrec-entry-del"
                  aria-label={`${t('delete')} — ${entryLabel(e)}`}
                  disabled={remove.isPending}
                  onClick={() => {
                    if (window.confirm(t('pregnancies.deleteConfirm'))) remove.mutate(e.id);
                  }}
                >
                  <Icon name="trash" size={18} />
                </button>
              </li>
            ))}
          </ul>
        ) : null}
        {draft ? (
          <div className="hrec-form">
            <SectionTitle title={draft.id === null ? t('pregnancies.addTitle') : t('pregnancies.editTitle')} level={3} />
            <ChipGroup label={t('pregnancies.outcomeLabel')}>
              {MANUAL_OUTCOMES.map((o) => (
                <PillChip key={o} pressed={draft.outcome === o} onPressedChange={() => setDraft({ ...draft, outcome: o })}>
                  {t(`pregnancies.outcome.${o}`)}
                </PillChip>
              ))}
            </ChipGroup>
            <label htmlFor={yearId} className="hrec-label">
              {t('pregnancies.yearLabel')}
            </label>
            <select
              id={yearId}
              className="hrec-input"
              value={draft.year ?? ''}
              onChange={(e) => setDraft({ ...draft, year: e.target.value ? Number(e.target.value) : null })}
            >
              <option value="">{t('pregnancies.yearUnknown')}</option>
              {Array.from({ length: YEARS_BACK }, (_, i) => thisYear - i).map((y) => (
                <option key={y} value={y}>
                  {formatNumber(y, loc)}
                </option>
              ))}
            </select>
            {draft.outcome !== 'ended' ? (
              <NumberStepper
                label={t('pregnancies.babiesLabel')}
                value={draft.babies}
                min={1}
                max={MAX_BABY_COUNT}
                onChange={(n) => setDraft({ ...draft, babies: n })}
                decrementLabel={t('pregnancies.fewer')}
                incrementLabel={t('pregnancies.more')}
                locale={loc}
                boxed
              />
            ) : null}
          </div>
        ) : null}
        <SaveError show={save.isError || remove.isError} />
      </div>
    </AppSheet>
  );
}
