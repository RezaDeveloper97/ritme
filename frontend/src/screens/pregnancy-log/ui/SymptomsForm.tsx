'use client';

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
import { addDays, diffInDays, formatJalaliDayMonth, toApiDate, today } from '@/shared/lib/date';
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
      <div className="card" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 12px' }}>
        <button className="iconbtn" onClick={() => shiftDay(-1)} aria-label={t('log.prevDay')}>
          <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
        </button>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Icon name="calendar" size={15} />
          <span style={{ fontSize: 14.5, fontWeight: 800, color: 'var(--ink)' }}>{formatJalaliDayMonth(date, locale)}</span>
          {isToday && <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--brand)', background: '#FFF1F7', borderRadius: 20, padding: '2px 9px' }}>{t('log.today')}</span>}
        </div>
        <button className="iconbtn" onClick={() => canGoNext && shiftDay(1)} disabled={!canGoNext} aria-label={t('log.nextDay')} style={{ opacity: canGoNext ? 1 : 0.3 }}>
          <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
        </button>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginTop: 12 }}>
        {SYMPTOM_GROUPS.map((group) => {
          const warning = group.key === 'warning';
          return (
            <PgCard
              key={group.key}
              title={t(`log.symptomGroups.${group.key}`)}
              hint={warning ? t('log.criticalHint') : undefined}
              accent={warning ? '#E5484D' : undefined}
              icon={warning ? 'shield' : 'sparkle'}
            >
              <div style={{ display: 'flex', flexDirection: 'column' }}>
                {group.symptoms.map((key) => {
                  const on = !!draft[`has_${key}`];
                  const critical = isCriticalSymptom(key);
                  return (
                    <div key={key} style={{ padding: '8px 0', borderBottom: '1px solid var(--line)' }}>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
                        <span style={{ fontSize: 13.5, fontWeight: 700, color: critical ? '#C13438' : 'var(--ink)' }}>
                          {dyn(`log.symptoms.${key}`)}
                        </span>
                        <Toggle on={on} onClick={() => toggleSymptom(key)} />
                      </div>
                      {on && (
                        <div style={{ marginTop: 8 }}>
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
        <div className="card" style={{ marginTop: 12, padding: '12px 13px', borderInlineStart: '3px solid #E9662E', display: 'flex', alignItems: 'center', gap: 8 }}>
          <Icon name="flame" size={16} style={{ color: '#E9662E' }} />
          <span style={{ fontSize: 12.5, fontWeight: 700, color: 'var(--ink)' }}>{t('alerts.raised', { n: alertsRaised })}</span>
        </div>
      )}
      {save.isError && (
        <p style={{ color: 'var(--brand)', fontSize: 12.5, fontWeight: 700, textAlign: 'center', marginTop: 12 }}>{t('log.saveError')}</p>
      )}

      <button className="btn btn-primary" onClick={handleSave} disabled={!filled || save.isPending} style={{ borderRadius: 16, marginTop: 16 }}>
        {save.isSuccess ? <Icon name="check" size={18} /> : null}
        {saveLabel}
      </button>
    </>
  );
}
