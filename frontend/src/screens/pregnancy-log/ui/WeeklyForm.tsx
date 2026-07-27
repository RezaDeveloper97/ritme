'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import {
  usePregnancyStatus,
  useSaveWeeklyLog,
  useWeeklyEnums,
  useWeeklyLog,
  WEEKLY_MENTAL,
  type WeeklyLogInput,
} from '@/entities/pregnancy';
import { Chip, FieldRow, NotesField, NumberField, PgCard, Segmented, Toggle } from '@/features/track-pregnancy';
import { toApiDate, today } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

type T = ReturnType<typeof useTranslations>;
type DynT = (key: string) => string;

/** Weekly checkup (upserts by pregnancy week): weight, vitals, swelling, and
 *  mental wellbeing. Out-of-range vitals can raise alerts on save. */
export function WeeklyForm({ t }: { t: T }) {
  const dyn = t as unknown as DynT;

  const statusQuery = usePregnancyStatus();
  const week = statusQuery.data?.currentWeek ?? null;
  const enums = useWeeklyEnums();
  const logQuery = useWeeklyLog(week);
  const save = useSaveWeeklyLog();

  const [draft, setDraft] = useState<WeeklyLogInput>(() => ({
    log_date: toApiDate(today()),
    pregnancy_week: week ?? 1,
  }));

  useEffect(() => {
    if (week != null) setDraft((d) => ({ ...d, pregnancy_week: week }));
  }, [week]);
  useEffect(() => {
    if (logQuery.data) {
      // Decimal fields arrive as strings from the API — coerce to numbers.
      const raw = logQuery.data as unknown as Record<string, unknown>;
      const num = (k: string) => (raw[k] == null ? undefined : Number(raw[k]));
      setDraft((d) => ({
        ...d,
        ...logQuery.data,
        weight: num('weight'),
        fasting_blood_sugar: num('fasting_blood_sugar'),
        post_meal_blood_sugar: num('post_meal_blood_sugar'),
      }));
    }
  }, [logQuery.data]);

  const set = <K extends keyof WeeklyLogInput>(key: K, value: WeeklyLogInput[K] | undefined) => {
    setDraft((d) => {
      const next = { ...d };
      if (value === undefined) delete next[key];
      else next[key] = value as WeeklyLogInput[K];
      return next;
    });
    save.reset();
  };

  const toggleSwellingLoc = (loc: string) => {
    setDraft((d) => {
      const cur = d.swelling_locations ?? [];
      const next = cur.includes(loc) ? cur.filter((l) => l !== loc) : [...cur, loc];
      return { ...d, has_swelling: true, swelling_locations: next.length ? next : undefined };
    });
    save.reset();
  };

  const severityOptions = (enums.data?.severity ?? ['mild', 'moderate', 'severe']).map((v) => ({ value: v, label: dyn(`log.severity.${v}`) }));
  const moodOptions = (enums.data?.overallMood ?? ['good', 'moderate', 'poor']).map((v) => ({ value: v, label: dyn(`log.weekly.mood.${v}`) }));
  const swellingLocs = enums.data?.swellingLocations ?? ['feet', 'hands', 'face'];

  const filled = Object.keys(draft).some((k) => k !== 'log_date' && k !== 'pregnancy_week' && draft[k as keyof WeeklyLogInput] !== undefined);

  const handleSave = () => {
    if (!filled || week == null) return;
    save.mutate({ ...draft, pregnancy_week: week, log_date: toApiDate(today()) });
  };
  const saveLabel = save.isPending ? t('log.saving') : save.isSuccess ? t('log.saved') : t('log.save');

  if (week == null) {
    return <div className="card" style={{ padding: '18px 14px', textAlign: 'center', color: 'var(--muted)', fontSize: 13 }}>{t('loading')}</div>;
  }

  return (
    <>
      <div style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)', margin: '0 2px 12px', textAlign: 'start' }}>
        {t('log.weekly.weekLabel', { week })}
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <PgCard title={t('log.weekly.weight')} icon="chart">
          <NumberField label="" value={draft.weight} onChange={(v) => set('weight', v)} min={30} max={200} step={0.1} placeholder="65" />
        </PgCard>

        <PgCard title={t('log.weekly.swelling')} icon="drop">
          <FieldRow label={t('log.weekly.swelling')}>
            <Toggle on={!!draft.has_swelling} onClick={() => { set('has_swelling', draft.has_swelling ? undefined : true); if (draft.has_swelling) set('swelling_locations', undefined); }} />
          </FieldRow>
          {draft.has_swelling && (
            <div style={{ marginTop: 6 }}>
              <span style={{ fontSize: 12.5, fontWeight: 700, color: 'var(--muted)', display: 'block', marginBottom: 8 }}>{t('log.weekly.swellingWhere')}</span>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {swellingLocs.map((loc) => (
                  <Chip key={loc} on={(draft.swelling_locations ?? []).includes(loc)} label={dyn(`log.weekly.swellingLocations.${loc}`)} onClick={() => toggleSwellingLoc(loc)} />
                ))}
              </div>
            </div>
          )}
          <FieldRow label={t('log.weekly.shortnessOfBreath')}>
            <Toggle on={!!draft.has_shortness_of_breath} onClick={() => set('has_shortness_of_breath', draft.has_shortness_of_breath ? undefined : true)} />
          </FieldRow>
        </PgCard>

        <PgCard title={t('log.weekly.hasBpDevice')} icon="stetho">
          <FieldRow label={t('log.weekly.hasBpDevice')}>
            <Toggle on={!!draft.has_blood_pressure_device} onClick={() => set('has_blood_pressure_device', draft.has_blood_pressure_device ? undefined : true)} />
          </FieldRow>
          {draft.has_blood_pressure_device && (
            <div style={{ display: 'flex', gap: 10, marginTop: 6 }}>
              <div style={{ flex: 1 }}><NumberField label={t('log.weekly.systolic')} value={draft.systolic_pressure} onChange={(v) => set('systolic_pressure', v)} min={60} max={250} /></div>
              <div style={{ flex: 1 }}><NumberField label={t('log.weekly.diastolic')} value={draft.diastolic_pressure} onChange={(v) => set('diastolic_pressure', v)} min={40} max={150} /></div>
            </div>
          )}
          <div style={{ display: 'flex', gap: 10, marginTop: 12 }}>
            <div style={{ flex: 1 }}><NumberField label={t('log.weekly.fastingSugar')} value={draft.fasting_blood_sugar} onChange={(v) => set('fasting_blood_sugar', v)} min={40} max={400} /></div>
            <div style={{ flex: 1 }}><NumberField label={t('log.weekly.postMealSugar')} value={draft.post_meal_blood_sugar} onChange={(v) => set('post_meal_blood_sugar', v)} min={40} max={500} /></div>
          </div>
        </PgCard>

        <PgCard title={t('log.weekly.moodLabel')} icon="smile">
          <Segmented options={moodOptions} value={draft.overall_mood} onChange={(v) => set('overall_mood', v)} />
          <div style={{ marginTop: 12, display: 'flex', flexDirection: 'column' }}>
            {WEEKLY_MENTAL.map(({ flag, severity }) => {
              const on = !!draft[flag as keyof WeeklyLogInput];
              return (
                <div key={flag} style={{ padding: '8px 0', borderTop: '1px solid var(--line)' }}>
                  <FieldRow label={dyn(`log.weekly.mental.${flag}`)}>
                    <Toggle on={on} onClick={() => { const willOff = on; set(flag as keyof WeeklyLogInput, willOff ? undefined : (true as never)); if (willOff) set(severity as keyof WeeklyLogInput, undefined); }} />
                  </FieldRow>
                  {on && (
                    <Segmented options={severityOptions} value={draft[severity as keyof WeeklyLogInput] as string | undefined} onChange={(v) => set(severity as keyof WeeklyLogInput, v as never)} />
                  )}
                </div>
              );
            })}
          </div>
        </PgCard>

        <PgCard title={t('log.notes')} icon="pencil">
          <NotesField label="" value={draft.notes} onChange={(v) => set('notes', v)} placeholder={t('log.notesPlaceholder')} />
        </PgCard>
      </div>

      {save.isSuccess && save.data.length > 0 && (
        <div className="card" style={{ marginTop: 12, padding: '12px 13px', borderInlineStart: '3px solid var(--orange)', display: 'flex', alignItems: 'center', gap: 8 }}>
          <Icon name="flame" size={16} style={{ color: 'var(--orange)' }} />
          <span style={{ fontSize: 12.5, fontWeight: 700, color: 'var(--ink)' }}>{t('alerts.raised', { n: save.data.length })}</span>
        </div>
      )}
      {save.isError && <p style={{ color: 'var(--brand)', fontSize: 12.5, fontWeight: 700, textAlign: 'center', marginTop: 12 }}>{t('log.saveError')}</p>}

      <button className="btn btn-primary" onClick={handleSave} disabled={!filled || save.isPending} style={{ borderRadius: 16, marginTop: 16 }}>
        {save.isSuccess ? <Icon name="check" size={18} /> : null}
        {saveLabel}
      </button>
    </>
  );
}
