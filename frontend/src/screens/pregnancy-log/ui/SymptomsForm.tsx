'use client';

import clsx from 'clsx';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useState } from 'react';

import {
  isCriticalSymptom,
  SYMPTOM_GROUPS,
  useSaveSymptomLog,
  useSymptomEnums,
  useSymptomLog,
  type SymptomKey,
  type SymptomLogInput,
} from '@/entities/pregnancy';
import { NotesField, PgCard, Segmented, Toggle } from '@/features/track-pregnancy';
import type { Locale } from '@/shared/i18n';
import { addDays, diffInDays, formatDayMonth, toApiDate, today } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

type T = ReturnType<typeof useTranslations>;
type DynT = (key: string) => string;

/** Daily symptom log (upserts by date). Critical symptoms are flagged and can
 *  raise alerts on save, which we surface inline. */
export function SymptomsForm({ t }: { t: T }) {
  const locale = useLocale() as Locale;
  const isRtl = locale === 'fa';
  const dyn = t as unknown as DynT;

  const [date, setDate] = useState<Date>(() => today());
  const apiDate = useMemo(() => toApiDate(date), [date]);

  const enums = useSymptomEnums();
  const logQuery = useSymptomLog(apiDate);
  const save = useSaveSymptomLog();

  const [draft, setDraft] = useState<SymptomLogInput>(() => ({ log_date: apiDate }));

  useEffect(() => {
    setDraft({ log_date: apiDate });
  }, [apiDate]);
  useEffect(() => {
    if (logQuery.data) setDraft({ ...logQuery.data, log_date: apiDate });
  }, [logQuery.data, apiDate]);

  const severityOptions = (enums.data?.severity ?? ['mild', 'moderate', 'severe']).map((v) => ({
    value: v,
    label: dyn(`log.severity.${v}`),
  }));

  const toggleSymptom = (key: SymptomKey) => {
    setDraft((d) => {
      const next = { ...d };
      const flag = `has_${key}`;
      if (next[flag]) {
        delete next[flag];
        delete next[`${key}_severity`];
      } else {
        next[flag] = true;
      }
      return next;
    });
    save.reset();
  };

  const setSeverity = (key: SymptomKey, value: string | undefined) => {
    setDraft((d) => {
      const next = { ...d, [`has_${key}`]: true };
      if (value === undefined) delete next[`${key}_severity`];
      else next[`${key}_severity`] = value;
      return next;
    });
    save.reset();
  };

  const setNotes = (value: string | undefined) => {
    setDraft((d) => {
      const next = { ...d };
      if (value === undefined) delete next.notes;
      else next.notes = value;
      return next;
    });
    save.reset();
  };

  const filled = Object.keys(draft).some((k) => k.startsWith('has_') && draft[k]) || !!draft.notes;

  const shiftDay = (delta: number) => {
    setDate((d) => addDays(d, delta));
    save.reset();
  };
  const canGoNext = diffInDays(date, today()) < 0;
  const isToday = diffInDays(date, today()) === 0;

  const handleSave = () => {
    if (!filled) return;
    save.mutate({ ...draft, log_date: apiDate });
  };
  const saveLabel = save.isPending ? t('log.saving') : save.isSuccess ? t('log.saved') : t('log.save');
  const alertsRaised = save.isSuccess ? save.data.length : 0;

  return (
    <>
      {/* Day switcher */}
      <div className="card plog-daynav">
        <button className="iconbtn" onClick={() => shiftDay(-1)} aria-label={t('log.prevDay')}>
          <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
        </button>
        <div className="plog-daynav-mid">
          <Icon name="calendar" size={15} />
          <span className="plog-daynav-date">{formatDayMonth(date, locale)}</span>
          {isToday && <span className="plog-today">{t('log.today')}</span>}
        </div>
        <button className="iconbtn" onClick={() => canGoNext && shiftDay(1)} disabled={!canGoNext} aria-label={t('log.nextDay')}>
          <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
        </button>
      </div>

      <div className="plog-stack is-spaced">
        {SYMPTOM_GROUPS.map((group) => {
          const warning = group.key === 'warning';
          return (
            <PgCard
              key={group.key}
              title={t(`log.symptomGroups.${group.key}`)}
              hint={warning ? t('log.criticalHint') : undefined}
              accent={warning ? 'var(--danger)' : undefined}
              icon={warning ? 'shield' : 'sparkle'}
            >
              <div className="plog-rows">
                {group.symptoms.map((key) => {
                  const on = !!draft[`has_${key}`];
                  const critical = isCriticalSymptom(key);
                  return (
                    <div key={key} className="plog-row">
                      <div className="plog-row-top">
                        <span className={clsx('plog-row-name', critical && 'is-critical')}>
                          {dyn(`log.symptoms.${key}`)}
                        </span>
                        <Toggle on={on} onClick={() => toggleSymptom(key)} />
                      </div>
                      {on && (
                        <div className="plog-row-extra">
                          <Segmented
                            options={severityOptions}
                            value={typeof draft[`${key}_severity`] === 'string' ? (draft[`${key}_severity`] as string) : undefined}
                            onChange={(v) => setSeverity(key, v)}
                          />
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </PgCard>
          );
        })}

        <PgCard title={t('log.notes')} icon="pencil">
          <NotesField label="" value={draft.notes as string | undefined} onChange={setNotes} placeholder={t('log.notesPlaceholder')} />
        </PgCard>
      </div>

      {alertsRaised > 0 && (
        <div className="card plog-note">
          <Icon name="flame" size={16} className="plog-note-icon" />
          <span className="plog-note-text">{t('alerts.raised', { n: alertsRaised })}</span>
        </div>
      )}
      {save.isError && (
        <p className="plog-error">{t('log.saveError')}</p>
      )}

      <button className="btn btn-primary plog-save" onClick={handleSave} disabled={!filled || save.isPending}>
        {save.isSuccess ? <Icon name="check" size={18} /> : null}
        {saveLabel}
      </button>
    </>
  );
}
