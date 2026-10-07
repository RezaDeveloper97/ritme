'use client';

import { useLocale, useTranslations } from 'next-intl';
import { type FormEvent, useEffect, useId, useState } from 'react';

import {
  type FamilyHistoryEntry,
  MAX_EXTRAS_ENTRIES,
  MAX_EXTRAS_TEXT,
  RELATIVES,
  type RecordExtras,
  RecordDateField,
  type Relative,
  type Surgery,
  useSaveRecordExtras,
} from '@/entities/health-record';
import type { Locale } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { ChipGroup, Icon, PillChip, PrimaryButton, SecondaryButton } from '@/shared/ui';

export type ExtrasSection = 'surgeries' | 'family_history';

/**
 * Surgeries / family history of the record (CB-REC-01 extras, «بستری و جراحی» / «سابقه خانوادگی» tiles of
 * nbl_Rec_Home): a local draft of the list, saved with one `PUT /health-record/extras` of that key only.
 */
export function ExtrasSheet({
  section,
  extras,
  onClose,
}: {
  section: ExtrasSection | null;
  extras: RecordExtras;
  onClose: () => void;
}) {
  const t = useTranslations('record.extras');
  const save = useSaveRecordExtras();
  const [surgeries, setSurgeries] = useState<Surgery[]>([]);
  const [family, setFamily] = useState<FamilyHistoryEntry[]>([]);
  useEffect(() => {
    if (section) {
      setSurgeries(extras.surgeries ?? []);
      setFamily(extras.familyHistory ?? []);
      save.reset();
    }
    // Re-seed only when the sheet opens.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [section]);

  const submit = () => {
    const input = section === 'surgeries' ? { surgeries } : { familyHistory: family };
    save.mutate(input, { onSuccess: onClose });
  };

  return (
    <AppSheet
      open={section !== null}
      onClose={onClose}
      size="full"
      title={section === 'family_history' ? t('familyTitle') : t('surgeriesTitle')}
      footer={
        <div className="rec-sheet-btns">
          <SecondaryButton block={false} onClick={onClose}>
            {t('cancel')}
          </SecondaryButton>
          <PrimaryButton block={false} loading={save.isPending} onClick={submit}>
            {t('save')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="hrec-sheet">
        {section === 'surgeries' ? <SurgeriesEditor items={surgeries} onChange={setSurgeries} /> : null}
        {section === 'family_history' ? <FamilyEditor items={family} onChange={setFamily} /> : null}
        {save.isError ? (
          <p className="hrec-error" role="alert">
            {t('saveError')}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}

function SurgeriesEditor({ items, onChange }: { items: Surgery[]; onChange: (next: Surgery[]) => void }) {
  const t = useTranslations('record.extras');
  const locale = useLocale() as Locale;
  const inputId = useId();
  const [title, setTitle] = useState('');
  const [date, setDate] = useState<string | null>(null);
  const full = items.length >= MAX_EXTRAS_ENTRIES;
  const add = (e: FormEvent) => {
    e.preventDefault();
    const v = title.trim().replace(/\s+/g, ' ');
    if (!v || full) return;
    onChange([...items, { title: v, date }]);
    setTitle('');
    setDate(null);
  };
  return (
    <>
      <EntryList
        empty={t('none')}
        rows={items.map((s) => ({ label: s.title, sub: s.date ? formatLongDate(fromApiDate(s.date), locale) : null }))}
        onRemove={(i) => onChange(items.filter((_, j) => j !== i))}
      />
      <form className="hrec-add" onSubmit={add}>
        <label htmlFor={inputId} className="hrec-label">
          {t('surgeriesTitle')}
        </label>
        <div className="hrec-add-row">
          <input
            id={inputId}
            className="hrec-input"
            value={title}
            maxLength={MAX_EXTRAS_TEXT}
            placeholder={t('surgeryPlaceholder')}
            disabled={full}
            onChange={(e) => setTitle(e.target.value)}
          />
          <button type="submit" className="hrec-add-btn" aria-label={t('add')} disabled={full || !title.trim()}>
            <Icon name="plus" size={20} />
          </button>
        </div>
        <RecordDateField label={t('pickDate')} value={date} unsetLabel={t('noDate')} clearLabel={t('noDate')} disabled={full} onChange={setDate} />
        {full ? <p className="hrec-hint">{t('max', { max: formatNumber(MAX_EXTRAS_ENTRIES, locale) })}</p> : null}
      </form>
    </>
  );
}

function FamilyEditor({ items, onChange }: { items: FamilyHistoryEntry[]; onChange: (next: FamilyHistoryEntry[]) => void }) {
  const t = useTranslations('record.extras');
  const locale = useLocale() as Locale;
  const inputId = useId();
  const [condition, setCondition] = useState('');
  const [relative, setRelative] = useState<Relative | null>(null);
  const full = items.length >= MAX_EXTRAS_ENTRIES;
  const add = (e: FormEvent) => {
    e.preventDefault();
    const v = condition.trim().replace(/\s+/g, ' ');
    if (!v || full) return;
    onChange([...items, { condition: v, relative }]);
    setCondition('');
    setRelative(null);
  };
  return (
    <>
      <EntryList
        empty={t('none')}
        rows={items.map((f) => ({ label: f.condition, sub: f.relative ? t(`relatives.${f.relative}`) : null }))}
        onRemove={(i) => onChange(items.filter((_, j) => j !== i))}
      />
      <form className="hrec-add" onSubmit={add}>
        <label htmlFor={inputId} className="hrec-label">
          {t('familyTitle')}
        </label>
        <div className="hrec-add-row">
          <input
            id={inputId}
            className="hrec-input"
            value={condition}
            maxLength={MAX_EXTRAS_TEXT}
            placeholder={t('conditionPlaceholder')}
            disabled={full}
            onChange={(e) => setCondition(e.target.value)}
          />
          <button type="submit" className="hrec-add-btn" aria-label={t('add')} disabled={full || !condition.trim()}>
            <Icon name="plus" size={20} />
          </button>
        </div>
        <span className="hrec-label">{t('relative')}</span>
        <ChipGroup label={t('relative')}>
          {RELATIVES.map((r) => (
            <PillChip key={r} pressed={relative === r} onPressedChange={(on) => setRelative(on ? r : null)} disabled={full}>
              {t(`relatives.${r}`)}
            </PillChip>
          ))}
        </ChipGroup>
        {full ? <p className="hrec-hint">{t('max', { max: formatNumber(MAX_EXTRAS_ENTRIES, locale) })}</p> : null}
      </form>
    </>
  );
}

function EntryList({
  rows,
  empty,
  onRemove,
}: {
  rows: { label: string; sub: string | null }[];
  empty: string;
  onRemove: (index: number) => void;
}) {
  const t = useTranslations('record.extras');
  if (rows.length === 0) return <p className="hrec-empty">{empty}</p>;
  return (
    <ul className="hrec-entries">
      {rows.map((r, i) => (
        <li key={`${r.label}-${i}`} className="hrec-entry">
          <span className="rec-entry-main">
            <span className="rec-entry-label">{r.label}</span>
            {r.sub ? <span className="rec-entry-sub">{r.sub}</span> : null}
          </span>
          <button type="button" className="hrec-entry-del" aria-label={t('remove', { item: r.label })} onClick={() => onRemove(i)}>
            <Icon name="trash" size={18} />
          </button>
        </li>
      ))}
    </ul>
  );
}
