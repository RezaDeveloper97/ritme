'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import type { Locale } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate, partsToDate, toApiDate, toParts, type DateParts } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, Icon, PrimaryButton, SecondaryButton } from '@/shared/ui';

import { tehranNow, validateMeasuredAt } from '../model/validate';

export type MeasuredAt = { date: string; time: string } | null;

/**
 * «زمان · امروز · ۸:۱۰» row of the add forms. `null` = now (the server stamps
 * it); a tap opens a half sheet with the locale calendar (Jalali in fa) and a
 * clock field. Future times are refused here and by the server.
 */
export function MeasuredAtField({
  value,
  onChange,
  error,
}: {
  value: MeasuredAt;
  onChange: (value: MeasuredAt) => void;
  error?: string;
}) {
  const t = useTranslations('vitals');
  const locale = useLocale() as Locale;
  const [open, setOpen] = useState(false);
  const now = tehranNow();
  const at = value ?? now;
  const dayLabel = at.date === now.date ? t('common.today') : formatDayMonth(fromApiDate(at.date), locale);
  const clock = formatNumber(at.time.replace(/^0(\d)/, '$1'), locale);
  return (
    <>
      <button type="button" className="nb-card vt-time" onClick={() => setOpen(true)} aria-label={`${t('add.changeTime')}: ${dayLabel} ${clock}`}>
        <span className="vt-time-label">{t('add.time')}</span>
        <b className="vt-time-value">
          {dayLabel}
          {t('common.separator')}
          <bdi dir="ltr">{clock}</bdi>
        </b>
        <Icon name="clock" size={18} className="vt-time-icon" />
      </button>
      {error ? (
        <p className="vt-error" role="alert">
          {error}
        </p>
      ) : null}
      {open ? <TimeSheet initial={at} onClose={() => setOpen(false)} onPick={(v) => { onChange(v); setOpen(false); }} /> : null}
    </>
  );
}

function TimeSheet({
  initial,
  onClose,
  onPick,
}: {
  initial: { date: string; time: string };
  onClose: () => void;
  onPick: (value: MeasuredAt) => void;
}) {
  const t = useTranslations('vitals');
  const locale = useLocale() as Locale;
  const [parts, setParts] = useState<DateParts>(() => toParts(fromApiDate(initial.date), locale));
  const [time, setTime] = useState(initial.time);
  const date = toApiDate(partsToDate(parts, locale));
  const err = validateMeasuredAt({ date, time }, Date.now()).measured_at;
  const errText = err ? t(`add.errors.${err.key}` as 'add.errors.future') : null;
  return (
    <AppSheet
      open
      size="half"
      onClose={onClose}
      title={t('add.timeSheet.title')}
      footer={
        <div className="vt-sheet-foot">
          <SecondaryButton onClick={() => onPick(null)}>{t('add.timeSheet.now')}</SecondaryButton>
          <PrimaryButton disabled={!!err || !/^\d{2}:\d{2}$/.test(time)} onClick={() => onPick({ date, time })}>
            {t('add.timeSheet.done')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="vt-sheet">
        <p className="vt-field-title">{t('add.timeSheet.date')}</p>
        <CalendarPicker value={parts} onSelect={setParts} />
        <label className="fld-label">
          <span className="fld-label-t">{t('add.timeSheet.clock')}</span>
          <input className="field fld-input vt-clock" type="time" dir="ltr" value={time} onChange={(e) => setTime(e.target.value)} />
        </label>
        {errText ? (
          <p className="vt-error" role="alert">
            {errText}
          </p>
        ) : null}
      </div>
    </AppSheet>
  );
}
