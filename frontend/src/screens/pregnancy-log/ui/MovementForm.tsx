'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useState } from 'react';

import {
  useFetalMovement,
  usePregnancyStatus,
  useSaveFetalMovement,
  useWeeklyEnums,
  type FetalMovementInput,
} from '@/entities/pregnancy';
import { NotesField, NumberField, PgCard, Segmented } from '@/features/track-pregnancy';
import type { Locale } from '@/shared/i18n';
import { addDays, diffInDays, formatJalaliDayMonth, toApiDate, today } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

type T = ReturnType<typeof useTranslations>;
type DynT = (key: string) => string;

/** Fetal-movement log (upserts by date). Reduced/absent movement from week 24+
 *  raises alerts on save. */
export function MovementForm({ t }: { t: T }) {
  const locale = useLocale() as Locale;
  const isRtl = locale === 'fa';
  const dyn = t as unknown as DynT;

  const statusQuery = usePregnancyStatus();
  const week = statusQuery.data?.currentWeek ?? 1;
  const trackingActive = statusQuery.data?.fetalMovementTrackingActive ?? week >= 18;

  const [date, setDate] = useState<Date>(() => today());
  const apiDate = useMemo(() => toApiDate(date), [date]);

  const enums = useWeeklyEnums();
  const logQuery = useFetalMovement(apiDate);
  const save = useSaveFetalMovement();

  const [draft, setDraft] = useState<FetalMovementInput>(() => ({
    log_date: apiDate,
    pregnancy_week: week,
    movement_status: '',
  }));

  useEffect(() => {
    setDraft({ log_date: apiDate, pregnancy_week: week, movement_status: '' });
  }, [apiDate, week]);
  useEffect(() => {
    if (logQuery.data) setDraft((d) => ({ ...d, ...logQuery.data, log_date: apiDate, pregnancy_week: week }));
  }, [logQuery.data, apiDate, week]);

  const statusOptions = (enums.data?.fetalMovementStatus ?? ['not_felt_yet', 'felt', 'normal', 'reduced', 'increased', 'none']).map((v) => ({
    value: v,
    label: dyn(`log.movement.status.${v}`),
  }));

  const set = <K extends keyof FetalMovementInput>(key: K, value: FetalMovementInput[K] | undefined) => {
    setDraft((d) => {
      const next = { ...d };
      if (value === undefined) delete next[key];
      else next[key] = value as FetalMovementInput[K];
      return next;
    });
    save.reset();
  };

  const shiftDay = (delta: number) => { setDate((d) => addDays(d, delta)); save.reset(); };
  const canGoNext = diffInDays(date, today()) < 0;
  const isToday = diffInDays(date, today()) === 0;

  const filled = !!draft.movement_status;
  const handleSave = () => {
    if (!filled) return;
    save.mutate({ ...draft, log_date: apiDate, pregnancy_week: week });
  };
  const saveLabel = save.isPending ? t('log.saving') : save.isSuccess ? t('log.saved') : t('log.save');

  return (
    <>
      <div className="card plog-daynav is-spaced">
        <button className="iconbtn" onClick={() => shiftDay(-1)} aria-label={t('log.prevDay')}>
          <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
        </button>
        <div className="plog-daynav-mid">
          <Icon name="calendar" size={15} />
          <span className="plog-daynav-date">{formatJalaliDayMonth(date, locale)}</span>
          {isToday && <span className="plog-today">{t('log.today')}</span>}
        </div>
        <button className="iconbtn" onClick={() => canGoNext && shiftDay(1)} disabled={!canGoNext} aria-label={t('log.nextDay')}>
          <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
        </button>
      </div>

      {!trackingActive && (
        <div className="card plog-hint">
          <Icon name="info" size={16} className="plog-hint-icon" />
          <span className="plog-hint-text">{t('log.movement.notYet')}</span>
        </div>
      )}

      <div className="plog-stack">
        <PgCard title={t('log.movement.statusLabel')} icon="heart">
          <Segmented options={statusOptions} value={draft.movement_status || undefined} onChange={(v) => set('movement_status', v ?? '')} />
        </PgCard>

        <PgCard title={t('log.movement.count')} icon="chart" hint={t('log.movement.trackingFrom')}>
          <NumberField label="" value={draft.movement_count} onChange={(v) => set('movement_count', v)} min={0} max={100} placeholder="10" />
          <div className="plog-pair is-lower">
            <label className="fld-label">
              <span className="fld-label-t">{t('log.movement.firstTime')}</span>
              <input className="field fld-input" type="time" value={draft.first_movement_time ?? ''} onChange={(e) => set('first_movement_time', e.target.value || undefined)} />
            </label>
            <label className="fld-label">
              <span className="fld-label-t">{t('log.movement.lastTime')}</span>
              <input className="field fld-input" type="time" value={draft.last_movement_time ?? ''} onChange={(e) => set('last_movement_time', e.target.value || undefined)} />
            </label>
          </div>
        </PgCard>

        <PgCard title={t('log.notes')} icon="pencil">
          <NotesField label="" value={draft.notes} onChange={(v) => set('notes', v)} placeholder={t('log.notesPlaceholder')} />
        </PgCard>
      </div>

      {save.isSuccess && save.data.length > 0 && (
        <div className="card plog-note is-danger">
          <Icon name="shield" size={16} className="plog-note-icon is-danger" />
          <span className="plog-note-text">{t('alerts.raised', { n: save.data.length })}</span>
        </div>
      )}
      {save.isError && <p className="plog-error">{t('log.saveError')}</p>}

      <button className="btn btn-primary plog-save" onClick={handleSave} disabled={!filled || save.isPending}>
        {save.isSuccess ? <Icon name="check" size={18} /> : null}
        {saveLabel}
      </button>
    </>
  );
}
